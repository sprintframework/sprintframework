/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprintpb"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/net/idna"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrCertificateNotFound        = errors.New("domain not found")
	ErrNotValidYetCertificate     = errors.New("not valid yet certificate")
	ErrExpiredCertificate         = errors.New("expired certificate")
	ErrInvalidCertificate         = errors.New("invalid certificate")
	ErrCertificateIssue           = errors.New("issue certificate")
	ErrCertificateIssueAfterGrace = errors.New("issue certificate after grace period")
	ErrCertificateNotIssued       = errors.New("certificate not issued")
	ErrCertificateNotReady        = errors.New("certificate not ready")
	ErrLeafCertificateNotFound    = errors.New("leaf certificate not found")

	RenewBefore = time.Hour * 24
	IssueGraceInterval = time.Hour
)

type implCertificateManager struct {

	Application sprint.Application `inject`
	Properties  glue.Properties `inject`
	Log         *zap.Logger       `inject`

	CertificateRepository sprint.CertificateRepository `inject`
	CertificateService    sprint.CertificateService    `inject`

	cache    sync.Map   // key is string, value is *certState
	renewal  sync.Map   // key is string, value is *certRenewal
	unknown  sync.Map   // key is string, value is *certUnknown

	zoneWatchCancel  context.CancelFunc
}

func CertificateManager() sprint.CertificateManager {
	return &implCertificateManager{
	}
}

func (t *implCertificateManager) PostConstruct() error {
	entry, err := t.CertificateRepository.FindZone("localhost")
	if err != nil {
		return err
	}
	if entry.Zone == "" {
		entry.Zone = "localhost"
		entry.Domains = []string {"localhost"}
		entry.CertProvider = "self"
		entry.SelfSigner = "localhost"
		entry.Options = []string { "localhost", "ip" }
		err = t.CertificateRepository.SaveZone(entry)
		if err != nil {
			return err
		}
	}
	if entry.Certificates == nil {
		 err := t.CertificateService.RenewCertificate( "localhost")
		if err != nil {
			return err
		}
	}
	s := t.getCertificate("localhost")
	t.cache.Store("127.0.0.1", s)

	t.zoneWatchCancel, err = t.CertificateRepository.Watch(context.Background(), t.onZoneChangeEvent)
	return err
}

func (t *implCertificateManager) onZoneChangeEvent(zone, event string) bool {
	t.InvalidateCache(zone)
	return true
}

func (t *implCertificateManager) InvalidateCache(zone string) {
	t.cache.Delete(zone)
	if value, ok := t.renewal.Load(zone); ok {
		if r, ok := value.(*certRenewal); ok {
			r.Stop()
		}
	}
	t.renewal.Delete(zone)
}

func (t *implCertificateManager) ListActive() map[string]error {
	result := make(map[string]error)
	t.cache.Range(func(key, value interface{}) bool {
		if zone, ok := key.(string); ok {
			if s, ok := value.(*certState); ok {
				result[zone] = s.Err()
			}
		}
		return true
	})
	return result
}

func (t *implCertificateManager) ListRenewal() map[string]time.Time {
	result := make(map[string]time.Time)
	t.renewal.Range(func(key, value interface{}) bool {
		if zone, ok := key.(string); ok {
			if s, ok := value.(*certRenewal); ok {
				result[zone] = s.at
			}
		}
		return true
	})
	return result
}

func (t *implCertificateManager) ListUnknown() map[string]time.Time {
	result := make(map[string]time.Time)
	t.unknown.Range(func(key, value interface{}) bool {
		if unk, ok := value.(*certUnknown); ok {
			unk.domains.Range(func(key, value interface{}) bool {
				if domain, ok := key.(string); ok {
					if at, ok := value.(time.Time); ok {
						result[domain] = at
					}
				}
				return true
			})
		}
		return true
	})
	return result
}

func (t *implCertificateManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	name := hello.ServerName

	if name == "" {
		name = "localhost"
	}

	domain := util.UnFqdn(name)

	punycode, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return nil, errors.Errorf("domain name '%s' contains invalid character", domain)
	}

	var zone string
	dots := strings.Count(punycode, ".")
	if dots <= 1 {
		zone = punycode
	} else {
		zone, err = util.ToZone(punycode)
		if err != nil {
			return nil, err
		}
	}

	if t.Application.IsDev() {
		t.Log.Info("GetCertificate",
			zap.String("serverName", hello.ServerName),
			zap.String("domain", domain),
			zap.String("punycode", punycode),
			zap.String("zone", zone))
	}

	for {
		s := t.getCertificate(zone)
		cert, err := s.Clone()
		if err == nil {
			return cert, nil
		}
		if err == ErrCertificateNotFound {
			t.addUnknown(zone, domain)
		}
		if err == ErrCertificateIssueAfterGrace {
			// try to clean and reload again
			t.InvalidateCache(zone)
		} else if zone != "localhost" {
			// fallback to localhost cert and try again
			zone = "localhost"
		} else {
			// no more tries
			return nil, err
		}
	}

}

// cert returns an existing certificate either from cache or repository.
func (t *implCertificateManager) getCertificate(zone string) *certState {

	s := &certState{
	}
	actual, loaded := t.cache.LoadOrStore(zone, s)
	if loaded {
		l, ok := actual.(*certState)
		if !ok {
			t.cache.Store(zone, s)
		} else {
			return l
		}
	}

	s.loadOnce.Do(func() {
		var cert *tls.Certificate
		cert, s.loadErr = t.loadCertificate(zone)
		if s.loadErr != nil {
			if t.Application.IsDev() {
				t.Log.Error("LoadCertificate", zap.String("zone", zone), zap.Error(s.loadErr))
			}
		} else {
			s.tlsCert.Store(cert)
		}
	})

	return s
}

func (t *implCertificateManager) loadCertificate(zone string) (*tls.Certificate, error) {

	tryAgain:
	entry, err := t.CertificateRepository.FindZone(zone)
	if err != nil || entry.Certificates == nil {
		return nil, ErrCertificateNotFound
	}

	if entry.Certificates == nil {
		if entry.CertProvider == "self" {
			err = t.CertificateService.RenewCertificate(zone)
			if err != nil {
				return nil, err
			}
			goto tryAgain
		} else if entry.CertProvider == "acme" {
			go t.issueCertificate(zone)
		}
		return nil, ErrCertificateNotIssued
	}

	cert, err := t.parseCertificate(entry)
	if err != nil {
		return nil, err
	}

	leaf, err := getLeaf(cert)
	if err != nil {
		return nil, err
	}

	err = validateCertificate(leaf, time.Now())

	if entry.CertProvider == "self" || entry.CertProvider == "acme" {
		if err != nil {
			go t.startRenew(zone, time.Now())
		} else {
			go t.startRenew(zone, leaf.NotAfter.Add(-RenewBefore))
		}
	}

	return cert, nil
}

func (t *implCertificateManager) startRenew(zone string, at time.Time) {
	tryAgain:
	// clean up old one
	if pre, loaded := t.renewal.LoadAndDelete(zone); loaded {
		if r, ok := pre.(*certRenewal); ok {
			if r.at.After(at) {
				r.Stop()
			} else {
				return
			}
		}
	}
	r := &certRenewal{
		manager: t,
		zone:    zone,
		at:      at,
	}
	_, loaded := t.renewal.LoadOrStore(zone, r)
	if loaded {
		// concurrent renewal
		goto tryAgain
	}
	after := at.Sub(time.Now())
	if after <= 0 {
		after = time.Millisecond
	}
	r.timer.Store(time.AfterFunc(after, r.renew))
}

func (t *implCertificateManager) Destroy() error {
	t.renewal.Range(func(key, value interface{}) bool {
		t.zoneWatchCancel()
		if r, ok := value.(*certRenewal); ok {
			r.Stop()
		}
		return true
	})
	return nil
}

// parseCertificate always returns a valid certificate, or an error otherwise.
func (t *implCertificateManager) parseCertificate(entry *sprintpb.Zone) (*tls.Certificate, error) {

	var buf bytes.Buffer
	buf.Write(entry.Certificates.Certificate)
	if entry.Certificates.IssuerCertificate != nil {
		buf.Write(entry.Certificates.IssuerCertificate)
	}

	tlsCert, err := tls.X509KeyPair(buf.Bytes(), entry.Certificates.PrivateKey)
	if err != nil {
		return nil, err
	}

	leaf, err := getLeaf(&tlsCert)
	if err != nil {
		return nil, err
	}

	// cache leaf
	tlsCert.Leaf = leaf

	return &tlsCert, nil
}

// async call in goroutine
func (t *implCertificateManager) issueCertificate(zone string) {

	actual, ok := t.cache.Load(zone)
	if !ok {
		return
	}

	s, ok := actual.(*certState)
	if !ok {
		return
	}

	s.issueOnce.Do(func() {
		s.issueAttempt = time.Now()
		// long run
		err := t.CertificateService.RenewCertificate(zone)
		if err != nil {
			if t.Application.IsDev() {
				t.Log.Error("RenewCertificate", zap.String("zone", zone), zap.Error(err))
			}
			s.issueErr = err
		} else {
			// trigger reload
			t.cache.Delete(zone)
		}

	})

}

type certUnknown struct {
	domains   sync.Map   // string domain, value is requested time.Time
}

func (t *implCertificateManager) addUnknown(zone, domain string) {
	actual, _ := t.unknown.LoadOrStore(zone, &certUnknown{})
	if unk, ok := actual.(*certUnknown); ok {
		unk.domains.Store(domain, time.Now())
	}
}

type certState struct {
	tlsCert       atomic.Value  // *tls.Certificate

	loadOnce      sync.Once
	loadErr       error

	issueOnce     sync.Once
	issueErr      error
	issueAttempt  time.Time
}

// possible errors: ErrInvalidCertificate, ErrCertificateIssue, ErrCertificateRecentIssue, ErrCertificateNotReady
func (t *certState) Clone() (*tls.Certificate, error) {

	if value := t.tlsCert.Load(); value != nil {
		if tlsCert, ok := value.(*tls.Certificate); ok {
			return &tls.Certificate{
				Certificate: tlsCert.Certificate,
				PrivateKey:  tlsCert.PrivateKey,
				Leaf:        tlsCert.Leaf,
			}, nil
		}
	}

	if t.issueErr != nil {
		if t.issueAttempt.Add(IssueGraceInterval).Before(time.Now()) {
			return nil, ErrCertificateIssueAfterGrace
		} else {
			return nil, ErrCertificateIssue
		}
	}

	if t.loadErr != nil {
		if t.loadErr == ErrCertificateNotFound {
			return nil, ErrCertificateNotFound
		} else {
			return nil, ErrInvalidCertificate
		}
	}
	return nil, ErrCertificateNotReady
}

func (t *certState) Err() error {
	if t.issueErr != nil {
		return errors.Errorf("issue certificate with last attempt at %v cause error %v", t.issueAttempt, t.issueErr)
	}
	if t.loadErr != nil {
		return errors.Errorf("load certificate cause error %v", t.loadErr)
	}
	return nil
}

type certRenewal struct {
	manager   *implCertificateManager
	zone      string
	at        time.Time

	renewOnce  sync.Once
	timer      atomic.Value   // *time.Timer

	stopOnce   sync.Once
}

func (t *certRenewal) Stop() {
	t.stopOnce.Do(func() {
		if value := t.timer.Load(); value != nil {
			if timerInst, ok := value.(*time.Timer); ok {
				if !timerInst.Stop() {
					<- timerInst.C
				}
			}
		}
	})
}

func (t *certRenewal) renew() {
	t.renewOnce.Do(func() {

		err := t.manager.CertificateService.RenewCertificate(t.zone)
		if err == nil {
			t.manager.InvalidateCache(t.zone)
		} else {
			t.manager.startRenew(t.zone, time.Now().Add(IssueGraceInterval))
		}

	})
}

func validateCertificate(leaf *x509.Certificate, now time.Time) error {
	if now.Before(leaf.NotBefore) {
		return ErrNotValidYetCertificate
	}
	if now.After(leaf.NotAfter) {
		return ErrExpiredCertificate
	}
	return nil
}

func getLeaf(cert *tls.Certificate) (leaf *x509.Certificate, err error) {
	if cert.Leaf != nil {
		return cert.Leaf, nil
	}
	if len(cert.Certificate) == 0 {
		return nil, ErrLeafCertificateNotFound
	}
	return x509.ParseCertificate(cert.Certificate[0])
}

func (t *implCertificateManager) ExecuteCommand(cmd string, args []string) (string, error) {

	if cmd != "manager" {
		return "", errors.Errorf("unknown command '%s'", cmd)
	}

	if len(args) == 0 {
		return fmt.Sprintf("Usage: ./%s cert manager [list active renewal unknown]", t.Application.Name()), nil
	}

	cmd = args[0]
	args = args[1:]

	switch cmd {

	case "list":

		var out strings.Builder
		for domain, certErr := range t.ListActive() {
			if certErr != nil {
				out.WriteString(fmt.Sprintf("%s: %v\n", domain, certErr))
			} else {
				out.WriteString(fmt.Sprintf("%s: serving\n", domain))
			}
		}
		return out.String(), nil

	case "active":

		var out strings.Builder
		for domain, certErr := range t.ListActive() {
			if certErr == nil {
				out.WriteString(fmt.Sprintf("%s: serving\n", domain))
			}
		}
		return out.String(), nil

	case "renewal":

		var out strings.Builder
		for domain, at := range t.ListRenewal() {
			out.WriteString(fmt.Sprintf("%s renewal scheduled at %v\n", domain, at))
		}
		return out.String(), nil

	case "unknown":

		var out strings.Builder
		for domain, at := range t.ListUnknown() {
			out.WriteString(fmt.Sprintf("%s requested at %v\n", domain, at))
		}
		return out.String(), nil

	default:
		return "", errors.Errorf("unknown cert manager command '%s'", cmd)
	}
}

