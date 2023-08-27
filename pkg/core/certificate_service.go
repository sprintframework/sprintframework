/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"bytes"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/codeallergy/seal"
	"github.com/go-acme/lego/v4/acme"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/codeallergy/properties"
	"github.com/sprintframework/sprintpb"
	"github.com/codeallergy/sealmod"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/net/idna"
	legolog "github.com/go-acme/lego/v4/log"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ClientCertFile = "client.crt"
	ClientKeyFile  = "client.key"
)

type implCertificateService struct {
	Application sprint.Application `inject`
	Properties  glue.Properties `inject`
	Log         *zap.Logger       `inject`

	CertificateRepository   sprint.CertificateRepository   `inject`
	SealService             seal.SealService             `inject`
	CertificateIssueService sprint.CertificateIssueService `inject`
	WhoisService            sprint.WhoisService            `inject`

	Algorithm     string   `value="tls.certificate.algorithm,default=RSA2048"`

	DNSProviders  map[string]sprint.DNSProvider `inject:"optional"`
	providerMap   map[string]sprint.DNSProvider // key is the provider name, not bean_name
	providerList  []string

	CompanyName   string        `value:"application.company,default=sprint"`

	acmeMutex  sync.Mutex

}

func CertificateService() sprint.CertificateService {
	return &implCertificateService{
		providerMap: make(map[string]sprint.DNSProvider),
	}
}

func (t *implCertificateService) PostConstruct() (err error) {

	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = errors.Errorf("%v", v)
			}
			t.Log.Error("PostConstruct", zap.Error(err))
		}
	}()

	for beanName, prov := range t.DNSProviders {
		name := beanName
		if strings.HasSuffix(name, "_provider") {
			name = name[:len(name) - len("_provider")]
		}
		t.providerMap[name] = prov
		t.providerList = append(t.providerList, name)
	}

	entry, err := t.CertificateRepository.FindSelfSigner("localhost")
	if err != nil {
		return err
	}
	if entry.Certificate == nil || entry.PrivateKey == nil {
		err = t.CreateSelfSigner("localhost", false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *implCertificateService) CreateAcmeAccount(email string) error {

	sealer, err := t.SealService.IssueSealer("RSA", 2048)
	if err != nil {
		return err
	}

	pubKey, err := sealer.EncodePublicKey()
	if err != nil {
		return err
	}

	privKey, err := sealer.EncodePrivateKey()
	if err != nil {
		return err
	}

	acc := &sprintpb.AcmeAccount{
		Email:      email,
		PublicKey:  []byte(pubKey),
		PrivateKey: []byte(privKey),
	}

	return t.CertificateRepository.SaveAccount(acc)
}

func (t *implCertificateService) GetOrCreateAcmeUser(email string) (user *sprint.AcmeUser, err error) {

	user = &sprint.AcmeUser{
		Email: email,
	}

	account, err := t.CertificateRepository.FindAccount(email)
	if err != nil {
		return nil, err
	}

	if account.PublicKey == nil || account.PrivateKey == nil {
		err = t.CreateAcmeAccount(email)
		if err != nil {
			return nil, errors.Errorf("create acme account '%s'", email)
		}
		account, err = t.CertificateRepository.FindAccount(email)
		if err != nil {
			return nil, err
		}
	}

	rsa, err := t.SealService.Sealer(
		sealmod.WithEncodedRSAPublicKey(string(account.PublicKey)),
		sealmod.WithEncodedRSAPrivateKey(string(account.PrivateKey)))
	if err != nil {
		return nil, err
	}

	user.PrivateKey = rsa.PrivateKey()
	return user, nil
}

func (t *implCertificateService) getOrCreateSelfIssuer(cn string) (sprint.CertificateIssuer, error) {

	selfSigner, err := t.CertificateRepository.FindSelfSigner(cn)
	if err != nil {
		return nil, err
	}

	if selfSigner.Name == "" {
		err = t.CreateSelfSigner(cn, false)
		if err != nil {
			return nil, errors.Wrapf(err, "create self signer '%s'", cn)
		}
		selfSigner, err = t.CertificateRepository.FindSelfSigner(cn)
		if err != nil {
			return nil, err
		}
	}

	issuer, err := t.CertificateIssueService.LoadIssuer(selfSigner)
	if err != nil {
		return nil, errors.Wrapf(err, "load self issuer '%s'", cn)
	}

	return issuer, nil
}

func (t *implCertificateService) CreateSelfSigner(cn string, withInter bool) error {

	info, err := t.CertificateIssueService.LoadCertificateDesc()
	if err != nil {
		return err
	}

	issuer, err := t.CertificateIssueService.CreateIssuer(cn, info)
	if err != nil {
		return err
	}

	if withInter {
		issuer, err = issuer.IssueInterCert(cn)
		if err != nil {
			return err
		}
	}

	entry := new(sprintpb.SelfSigner)

	for i, ok, e := issuer, true, entry; ok; i, ok = i.Parent() {
		e.Issuer = &sprintpb.SelfSigner{
			Certificate: i.Certificate().CertFileContents(),
			PrivateKey:  i.Certificate().KeyFileContents(),
		}
		e = e.Issuer
	}

	entry.Issuer.Name = cn
	return t.CertificateRepository.SaveSelfSigner(entry.Issuer)
}

func (t *implCertificateService) RenewCertificate(zone string) error {

	entry, err := t.CertificateRepository.FindZone(zone)
	if err != nil {
		return err
	}
	if entry.Zone != zone {
		return errors.Errorf("certificate zone '%s' is not found", zone)
	}

	switch entry.CertProvider {

	case "self":
		err = t.IssueSelfSignedCertificate(entry)
		if err != nil {
			return err
		}
		return t.CertificateRepository.SaveZone(entry)
	case "acme":
		_, err = t.IssueAcmeCertificate(entry)
		if err != nil {
			return err
		}
		return t.CertificateRepository.SaveZone(entry)
	case "custom":
		return errors.New("can not issue custom certificates")
	default:
		return errors.Errorf("unknown cert provider '%s'", entry.CertProvider)
	}

}

func (t *implCertificateService) IssueSelfSignedCertificate(entry *sprintpb.Zone) error {

	if entry.SelfSigner == "" {
		entry.SelfSigner = "localhost"
	}

	issuer, err := t.getOrCreateSelfIssuer(entry.SelfSigner)
	if err != nil {
		return err
	}

	options := indexStrings(entry.Options)
	domains := indexStrings(entry.Domains)

	if options["localhost"] {
		domains["localhost"] = true
	}

	var ipAddresses []net.IP
	if options["ip"] {
		ipAddresses, err = t.CertificateIssueService.LocalIPAddresses(options["localhost"])
		if err != nil {
			return err
		}
	}

	issuedCert, err := issuer.IssueServerCert(entry.Zone, asList(domains), ipAddresses)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	for i, ok := issuer, true; ok; i, ok = i.Parent() {
		buf.Write(i.Certificate().CertFileContents())
	}

	entry.Certificates = &sprintpb.Certificates{
		Domain:            entry.Zone,
		CertUrl:           "",
		CertStableUrl:     "",
		PrivateKey:        issuedCert.KeyFileContents(),
		Certificate:       issuedCert.CertFileContents(),
		IssuerCertificate: buf.Bytes(),
	}
	
	return nil

}

func (t *implCertificateService) IssueAcmeCertificate(entry *sprintpb.Zone) (string, error) {

	if len(entry.Domains) == 0 {
		return "", errors.Errorf("zone name '%s' has empty domains", entry.Zone)
	}

	if entry.DnsProvider == "" {
		return "", errors.Errorf("zone name '%s' has empty DNS provider", entry.Zone)
	}

	if !strings.Contains(strings.Trim(entry.Zone, "."), ".") {
		return "", errors.Errorf("zone name '%s' component count invalid", entry.Zone)
	}

	if entry.AcmeEmail == "" {
		entry.AcmeEmail = fmt.Sprintf("admin@%s", entry.Domains[0])
	}

	user, err := t.GetOrCreateAcmeUser(entry.AcmeEmail)
	if err != nil {
		return "", err
	}

	config := lego.NewConfig(acmeUserAdapter{user})

	config.Certificate = lego.CertificateConfig{
		KeyType: getKeyType(t.Algorithm),
		Timeout: 60 * time.Second,
	}

	config.UserAgent = fmt.Sprintf("%s/%s", t.Application.Name(), t.Application.Version())

	client, err := lego.NewClient(config)
	if err != nil {
		return "", err
	}

	prov, ok := t.providerMap[entry.DnsProvider]
	if !ok {
		return "", errors.Errorf("DNS provider '%s' not found", entry.DnsProvider)
	}

	if err := prov.RegisterChallenge(client, entry.DnsProviderToken); err != nil {
		return "", errors.Errorf("DNS provider '%s' does not have credentials, %v", entry.DnsProvider, err)
	}

	var logContent []byte
	var certificates *certificate.Resource
	if entry.Certificates != nil && entry.Certificates.Certificate != nil && entry.Certificates.IssuerCertificate != nil {

		renew := entry.Certificates

		t.Log.Info("RenewRequest",
			zap.String("commonName", entry.Zone),
			zap.String("renew.Domain", renew.Domain),
			zap.Strings("domains", entry.Domains),
			zap.String("email", user.Email))

		certs := certificate.Resource{
			Domain:            renew.Domain,
			CertURL:           renew.CertUrl,
			CertStableURL:     renew.CertStableUrl,
			Certificate:       renew.Certificate,
			IssuerCertificate: renew.IssuerCertificate,
			CSR:               renew.Csr,
		}

		logContent = t.doAcmeCall(func() {

			reg, err := client.Registration.QueryRegistration()

			if err != nil {
				reg, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
				if err != nil {
					return
				}
			}

			user.Registration = wrapAcmeResource(reg)
			certificates, err = client.Certificate.Renew(certs, true, true, "")
		})

		if err != nil {
			t.Log.Warn("CertificateRenew", zap.String("zone", entry.Zone), zap.String("log", string(logContent)), zap.Error(err))
			return "", err
		}else {
			t.Log.Info("CertificateRenew", zap.String("zone", entry.Zone), zap.String("log", string(logContent)))
		}

	} else {

		t.Log.Info("ObtainRequest",
			zap.String("commonName", entry.Zone),
			zap.Strings("domains", entry.Domains),
			zap.String("email", user.Email))

		request := certificate.ObtainRequest{
			Domains: entry.Domains,
			Bundle:  true,
			MustStaple: true,
		}

		logContent = t.doAcmeCall(func() {

			reg, err := client.Registration.QueryRegistration()

			if err != nil {
				reg, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
				if err != nil {
					return
				}
			}

			user.Registration = wrapAcmeResource(reg)
			certificates, err = client.Certificate.Obtain(request)
		})

		if err != nil {
			t.Log.Warn("CertificateObtain", zap.String("zone", entry.Zone), zap.String("log", string(logContent)), zap.Error(err))
			return "", err
		} else {
			t.Log.Info("CertificateObtain", zap.String("zone", entry.Zone), zap.String("log", string(logContent)))
		}

	}

	entry.Certificates = &sprintpb.Certificates{
		Domain:            certificates.Domain,
		CertUrl:           certificates.CertURL,
		CertStableUrl:     certificates.CertStableURL,
		PrivateKey:        certificates.PrivateKey,
		Certificate:       certificates.Certificate,
		IssuerCertificate: certificates.IssuerCertificate,
		Csr:               certificates.CSR,
	}

	return string(logContent), err
}

type acmeUserAdapter struct {
	user *sprint.AcmeUser
}

func (u acmeUserAdapter) GetEmail() string {
	return u.user.Email
}
func (u acmeUserAdapter) GetRegistration() *registration.Resource {
	r := u.user.Registration
	if r == nil {
		return nil
	}
	return &registration.Resource {
		Body: acme.Account{
			Status:                 r.Body.Status,
			Contact:                r.Body.Contact,
			TermsOfServiceAgreed:   r.Body.TermsOfServiceAgreed,
			Orders:                 r.Body.Orders,
			OnlyReturnExisting:     r.Body.OnlyReturnExisting,
			ExternalAccountBinding: r.Body.ExternalAccountBinding,
		},
		URI: r.URI,
	}
}
func (u acmeUserAdapter) GetPrivateKey() crypto.PrivateKey {
	return u.user.PrivateKey
}

func wrapAcmeResource(r *registration.Resource) *sprint.AcmeResource {
	return &sprint.AcmeResource{
		Body: sprint.AcmeAccount{
			Status:                 r.Body.Status,
			Contact:                r.Body.Contact,
			TermsOfServiceAgreed:   r.Body.TermsOfServiceAgreed,
			Orders:                 r.Body.Orders,
			OnlyReturnExisting:     r.Body.OnlyReturnExisting,
			ExternalAccountBinding: r.Body.ExternalAccountBinding,
		},
		URI:  r.URI,
	}
}

func (t *implCertificateService) doAcmeCall(cb func()) []byte {

	t.acmeMutex.Lock()
	saveLogger := legolog.Logger

	var buf bytes.Buffer
	legolog.Logger = log.New(&buf, "", log.LstdFlags)

	cb()

	legolog.Logger = saveLogger
	t.acmeMutex.Unlock()

	return buf.Bytes()

}

func (t *implCertificateService) ExecuteCommand(cmd string, args []string) (string, error) {

	switch cmd {
	case "list":
		return t.listCerts(args)

	case "dump":
		return t.dumpCert(args)

	case "upload":
		return t.uploadCert(args)

	case "create":
		return t.createCert(args)

	case "renew":
		return t.renewCert(args)

	case "remove":
		return t.removeCert(args)

	case "client":
		return t.clientCert(args)

	case "acme":
		return t.acmeCommand(args)

	case "self":
		return t.selfCommand(args)

	default:
		return "", errors.Errorf("unknown command '%s'", cmd)
	}

}

func (t *implCertificateService) listCerts(args []string) (string, error) {

	var prefix string
	if len(args) > 0 {
		prefix = args[0]
		args = args[1:]
	}

	var out strings.Builder
	out.WriteString("Zone,Domains,Provider,Options,Issued\n")
	err := t.CertificateRepository.ListZones(prefix, func(entry *sprintpb.Zone) bool {
		provider := entry.CertProvider
		switch entry.CertProvider {
		case "self":
			provider = fmt.Sprintf("%s(%s)", entry.CertProvider, entry.SelfSigner)
		case "acme":
			provider = fmt.Sprintf("%s(%s) with DNS-01 by %s", entry.CertProvider, entry.AcmeEmail, entry.DnsProvider)
		case "custom":
			provider = "custom"
		}
		out.WriteString(fmt.Sprintf("%s,%+v,%s,%+v,%v\n", entry.Zone, entry.Domains, provider, entry.Options, entry.Certificates != nil))
		return true
	})
	return out.String(), err
}

func (t *implCertificateService) dumpCert(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert dump zone", t.Application.Name()), nil
	}

	zone := args[0]

	entry, err := t.CertificateRepository.FindZone(zone)
	if err != nil {
		return "", err
	}

	return protojson.Format(entry), nil
}

func (t *implCertificateService) uploadCert(args []string) (string, error) {
	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert upload dump_file.json", t.Application.Name()), nil
	}
	jsonContents, err := ioutil.ReadFile(args[0])
	if err != nil {
		return "", err
	}

	opts := protojson.UnmarshalOptions{DiscardUnknown: true}

	entry := new(sprintpb.Zone)
	err = opts.Unmarshal(jsonContents, entry)
	if err != nil {
		return "", err
	}

	if entry.Zone == "" {
		return "", errors.New("empty zone in entry")
	}

	if len(entry.Domains) == 0 {
		return "", errors.New("empty domains in entry")
	}

	if entry.CertProvider == "" {
		return "", errors.New("empty certificate provider in entry")
	}

	if err := t.CertificateRepository.SaveZone(entry); err != nil {
		return "", err
	} else {
		return "OK", nil
	}
}


func (t *implCertificateService) createCert(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert create cert_provider [self,acme,custom]", t.Application.Name()), nil
	}

	certProvider := args[0]
	args = args[1:]

	switch certProvider {
	case "self":
		return t.createSelfCert(args)
	case "acme":
		return t.createAcmeCert(args)
	case "custom":
		return t.createCustomCert(args)
	}

	return "", nil
}

func (t *implCertificateService) createSelfCert(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert create self domain [self-signer]", t.Application.Name()), nil
	}

	domain := util.UnFqdn(args[0])
	args = args[1:]

	punycode, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return"", errors.Wrapf(err, "domain name '%s' contains invalid character", domain)
	}

	zone, err := util.ToZone(punycode)
	if err != nil {
		return "", err
	}

	exist, err := t.CertificateRepository.FindZone(zone)
	if err != nil {
		return "", err
	}

	if exist.Zone != "" {
		return "", errors.Errorf("zone '%s' already exist", zone)
	}

	domains := []string {
		zone,
		fmt.Sprintf("*.%s", zone),
	}

	selfSigner := "localhost"
	if len(args) > 0 {
		selfSigner = args[0]
	}

	ss, err := t.CertificateRepository.FindSelfSigner(selfSigner)
	if err != nil {
		return "", err
	}

	var warning string
	if ss.Name != selfSigner {
		t.Log.Info("NotFound", zap.String("selfSigner", selfSigner))
		warning = fmt.Sprintf("Warning: self signed root certificates '%s' were not found", selfSigner)
	}

	entry := &sprintpb.Zone{
		Zone:         zone,
		Domains:      domains,
		CertProvider: "self",
		SelfSigner:   selfSigner,
	}

	err = t.IssueSelfSignedCertificate(entry)
	if err != nil {
		return "", errors.Wrapf(err, "issue self signed certificate for zone '%s', domains '%v'", zone, domains)
	}

	x509Cert, err := t.parseCertificate(entry.Certificates)
	if err != nil {
		return "", err
	}

	err = t.CertificateRepository.SaveZone(entry)
	if err != nil {
		return "", err
	}

	domains = certcrypto.ExtractDomains(x509Cert)
	timeLeft := x509Cert.NotAfter.Sub(time.Now().UTC())

	msg := fmt.Sprintf("Created self-signed certificate [%s] for domains %+v with %d hours remaining", zone, domains, int(timeLeft.Hours()))
	if warning != "" {
		msg = fmt.Sprintf("%s\n%s", msg, warning)
	}
	return msg, nil
}

func (t *implCertificateService) createAcmeCert(args []string) (string, error) {

	if len(args) < 2 {
		return fmt.Sprintf("Usage: ./%s cert create acme domain email [dns_provider]", t.Application.Name()), nil
	}

	domain := util.UnFqdn(args[0])
	email := strings.ToLower(args[1])

	var dnsProvider string
	if len(args) > 2 {
		dnsProvider = strings.ToLower(args[2])
	}

	acc, err := t.CertificateRepository.FindAccount(email)
	if err != nil {
		return "", err
	}

	var warning string
	if acc.Email != email {
		t.Log.Info("NotFound", zap.String("acmeAccount", email))
		warning = fmt.Sprintf("Warning: ACME account '%s' was not found", email)
	}

	punycode, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return"", errors.Wrapf(err, "domain name '%s' contains invalid character", domain)
	}

	zone, err := util.ToZone(punycode)
	if err != nil {
		return "", err
	}

	exist, err := t.CertificateRepository.FindZone(zone)
	if err != nil {
		return "", err
	}

	if exist.Zone != "" {
		return "", errors.Errorf("zone '%s' already exist", zone)
	}

	domains := []string {
		zone,
		fmt.Sprintf("*.%s", zone),
	}

	if dnsProvider == "" {
		dnsProvider, err = t.delectProviderFromWhois(zone)
		if err != nil {
			return "", errors.Wrapf(err, "detect provider from whois for zone '%s", zone)
		}
	}

	prov, ok := t.providerMap[dnsProvider]
	if !ok {
		return "", errors.Errorf("dns provider '%s' not found in supported list %+v", dnsProvider, t.providerList)
	}

	var ask error
	_, err = prov.NewClient()
	if err != nil {
		t.Log.Error("DNSProvider", zap.String("zone", zone), zap.String("provider", dnsProvider), zap.Error(err))
		ask = errors.Errorf("Warning: token for DNS provider %s not found, %v", dnsProvider, err)
	}

	entry := &sprintpb.Zone{
		Zone:         zone,
		Domains:      domains,
		CertProvider: "acme",
		AcmeEmail:    email,
		DnsProvider: dnsProvider,
	}

	msg, err := t.IssueAcmeCertificate(entry)
	if err != nil {
		return "", errors.Wrapf(err, "issue acme certificate for zone '%s', domains '%v'", zone, domains)
	}

	x509Cert, err := t.parseCertificate(entry.Certificates)
	if err != nil {
		return "", err
	}

	err = t.CertificateRepository.SaveZone(entry)
	if err != nil {
		return "", err
	}

	domains = certcrypto.ExtractDomains(x509Cert)
	timeLeft := x509Cert.NotAfter.Sub(time.Now().UTC())

	msg = fmt.Sprintf("%s\nAcme issued certificate [%s] for domains %+v with %d hours remaining", msg, zone, domains, int(timeLeft.Hours()))
	if ask != nil {
		msg = fmt.Sprintf("%s\n%v", msg, ask)
	}
	if warning != "" {
		msg = fmt.Sprintf("%s\n%s", msg, warning)
	}
	return msg, nil

}

func (t *implCertificateService) delectProviderFromWhois(zone string) (string, error) {

	whoisResult, err := t.WhoisService.Whois(zone)
	if err != nil {
		return "", errors.Wrapf(err, "get whois for zone '%s", zone)
	}

	whois := t.WhoisService.Parse(whoisResult)

	for name, prov := range t.providerMap {

		if prov.Detect(whois) {
			return name, nil
		}

	}

	return "", errors.Errorf("DNS provider not found for zone '%s' in whois response '%s'", zone, whoisResult)
}

func (t *implCertificateService) createCustomCert(args []string) (string, error) {

	if len(args) < 4 {
		return fmt.Sprintf("Usage: ./%s cert create custom domain cert_file key_file issuer_cert_file", t.Application.Name()), nil
	}

	domain := util.UnFqdn(args[0])
	certFile := args[1]
	keyFile := args[2]
	issuerCertFile := args[3]

	punycode, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return"", errors.Wrapf(err, "domain name '%s' contains invalid character", domain)
	}

	zone, err := util.ToZone(punycode)
	if err != nil {
		return "", err
	}

	exist, err := t.CertificateRepository.FindZone(zone)
	if err != nil {
		return "", err
	}

	if exist.Zone != "" {
		return "", errors.Errorf("zone '%s' already exist", zone)
	}

	certContents, err := ioutil.ReadFile(certFile)
	if err != nil {
		return "", errors.Wrapf(err, "read file '%s'", certFile)
	}

	keyContents, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return "", errors.Wrapf(err, "read file '%s'", keyFile)
	}

	issuerCertContents, err := ioutil.ReadFile(issuerCertFile)
	if err != nil {
		return "", errors.Wrapf(err, "read file '%s'", issuerCertFile)
	}

	entry := &sprintpb.Zone{
		Zone:         zone,
		CertProvider: "custom",
		Certificates:  &sprintpb.Certificates{
			Domain:            domain,
			PrivateKey:        keyContents,
			Certificate:       certContents,
			IssuerCertificate: issuerCertContents,
		},
	}

	x509Cert, err := t.parseCertificate(entry.Certificates)
	if err != nil {
		return "", err
	}

	domains := certcrypto.ExtractDomains(x509Cert)
	timeLeft := x509Cert.NotAfter.Sub(time.Now().UTC())

	entry.Domains = domains

	err = t.CertificateRepository.SaveZone(entry)
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf("Uploaded certificate [%s] for domains %+v with %d hours remaining", zone, domains, int(timeLeft.Hours()))
	return msg, nil
}

func (t *implCertificateService) loadCertificate(zone string) (*x509.Certificate, error) {

	entry, err := t.CertificateRepository.FindZone(zone)
	if err != nil {
		return nil, err
	}

	if entry.Zone != zone {
		return nil, errors.Errorf("zone '%s' not found", zone)
	}

	return t.parseCertificate(entry.Certificates)
}

func (t *implCertificateService) parseCertificate(cert *sprintpb.Certificates) (*x509.Certificate, error) {

	if cert == nil || cert.Certificate == nil || cert.PrivateKey == nil {
		return nil, errors.New("empty certificates")
	}

	var buf bytes.Buffer
	buf.Write(cert.Certificate)
	if cert.IssuerCertificate != nil {
		buf.Write(cert.IssuerCertificate)
	}

	tlsCert, err := tls.X509KeyPair(buf.Bytes(), cert.PrivateKey)
	if err != nil {
		return nil, err
	}

	if len(tlsCert.Certificate) == 0 {
		return nil, errors.New("leaf certificate not found")
	}

	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "parse x509 certificate")
	}

	return x509Cert, nil
}

func (t *implCertificateService) renewCert(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert renew zone", t.Application.Name()), nil
	}

	zone := args[0]

	err := t.RenewCertificate(zone)
	if err != nil {
		return "", err
	}

	x509Cert, err := t.loadCertificate(zone)
	if err != nil {
		return "", errors.Wrapf(err, "load certificate for zone '%s'", zone)
	}

	domains := certcrypto.ExtractDomains(x509Cert)
	timeLeft := x509Cert.NotAfter.Sub(time.Now().UTC())

	msg := fmt.Sprintf("Succesfull renewal of certificate [%s] for domains %+v with %d hours remaining", zone, domains, int(timeLeft.Hours()))
	return msg, nil
}

func (t *implCertificateService) removeCert(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert remove zone", t.Application.Name()), nil
	}

	zone := args[0]

	exist, err := t.CertificateRepository.FindZone(zone)
	if err != nil {
		return "", err
	}

	if exist.Zone == "" {
		return "", errors.Errorf("zone '%s' not found", zone)
	}

	err = t.CertificateRepository.DeleteZone(zone)
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf("Succesfull removed certificate [%s] for domains %+v ", zone, exist.Domains)
	return msg, nil
}

func (t *implCertificateService) clientCert(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert client [install]", t.Application.Name()), nil
	}

	cmd := args[0]
	args = args[1:]

	switch cmd {
		case "install":
			return t.installClientCert(args)
		default:
			return "", errors.Errorf("unknown cert client command '%s'", cmd)
	}

}

func (t *implCertificateService) installClientCert(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert client install zone", t.Application.Name()), nil
	}

	zone := args[0]

	entry, err := t.CertificateRepository.FindZone(zone)
	if err != nil {
		return "", err
	}

	if entry.Zone == "" {
		return "", errors.Errorf("zone '%s' not found", zone)
	}

	if entry.Certificates == nil || entry.Certificates.Certificate == nil || entry.Certificates.IssuerCertificate == nil {
		return "", errors.Errorf("zone '%s' has empty certificates", zone)
	}

	if entry.CertProvider != "self" {
		return "", errors.New("only self signed zones are supported")
	}

	if entry.SelfSigner == "" {
		entry.SelfSigner = "localhost"
	}

	issuer, err := t.getOrCreateSelfIssuer(entry.SelfSigner)
	if err != nil {
		return "", err
	}

	issuedCert, _, err := issuer.IssueClientCert(entry.Zone, "")
	if err != nil {
		return "", err
	}

	appFolder, err := properties.Locate(t.CompanyName).MakeDir(t.Application.Name())
	if err != nil {
		return "", err
	}

	file := filepath.Join(appFolder, ClientCertFile)
	err = ioutil.WriteFile(file, issuedCert.CertFileContents(), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "writing certificate to file '%s'", file)
	}

	file = filepath.Join(appFolder, ClientKeyFile)
	err = ioutil.WriteFile(file, issuedCert.KeyFileContents(), 0600)
	if err != nil {
		return "", errors.Wrapf(err, "writing key to file '%s'", file)
	}

	return "OK", nil
}

func (t *implCertificateService) acmeCommand(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert acme [list create upload dump]", t.Application.Name()), nil
	}

	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "list":
		return t.acmeList(args)
	case "create":
		return t.acmeCreate(args)
	case "upload":
		return t.acmeUpload(args)
	case "dump":
		return t.acmeDump(args)

	default:
		return "", errors.Errorf("unknown acme command: %s", cmd)
	}

}

func (t *implCertificateService) acmeList(args []string) (string, error) {
	var out strings.Builder
	out.WriteString("Email\n")
	err := t.CertificateRepository.ListAccounts("", func(account *sprintpb.AcmeAccount) bool {
		out.WriteString(account.Email)
		out.WriteByte('\n')
		return true
	})
	return out.String(), err
}

func (t *implCertificateService) acmeCreate(args []string) (string, error) {
	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert acme create email", t.Application.Name()), nil
	}
	email := strings.ToLower(args[0])
	if err := t.CreateAcmeAccount(email); err != nil {
		return "", err
	} else {
		return fmt.Sprintf("Created ACME Account for %s", email), nil
	}
}

func (t *implCertificateService) acmeUpload(args []string) (string, error) {
	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert acme upload dump_file.json", t.Application.Name()), nil
	}
	jsonContents, err := ioutil.ReadFile(args[0])

	opts := protojson.UnmarshalOptions{DiscardUnknown: true}

	acc := new(sprintpb.AcmeAccount)
	err = opts.Unmarshal(jsonContents, acc)
	if err != nil {
		return "", err
	}

	if acc.Email == "" {
		return "", errors.New("empty email in account")
	}

	_, err = t.SealService.Sealer(
		sealmod.WithEncodedRSAPublicKey(string(acc.PublicKey)),
		sealmod.WithEncodedRSAPrivateKey(string(acc.PrivateKey)))
	if err != nil {
		return "", errors.Wrapf(err, "parse acme account '%s'", acc.Email)
	}

	if err := t.CertificateRepository.SaveAccount(acc); err != nil {
		return "", err
	} else {
		return "OK", nil
	}
}

func (t *implCertificateService) acmeDump(args []string) (string, error) {
	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert acme dump email", t.Application.Name()), nil
	}
	email := strings.ToLower(args[0])

	entry, err := t.CertificateRepository.FindAccount(email)
	if err != nil {
		return "", err
	}

	if entry.Email == "" {
		return "", errors.Errorf("acme account '%s' not found", email)
	}

	return protojson.Format(entry), nil
}

func (t *implCertificateService) selfCommand(args []string) (string, error) {

	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert self [list create upload dump]", t.Application.Name()), nil
	}

	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "list":
		return t.selfList(args)
	case "create":
		return t.selfCreate(args)
	case "upload":
		return t.selfUpload(args)
	case "dump":
		return t.selfDump(args)

	default:
		return "", errors.Errorf("unknown self command: %s", cmd)
	}

}

func (t *implCertificateService) selfList(args []string) (string, error) {
	var out strings.Builder
	out.WriteString("Name\n")
	err := t.CertificateRepository.ListSelfSigners("", func(self *sprintpb.SelfSigner) bool {
		out.WriteString(self.Name)
		out.WriteByte('\n')
		return true
	})
	return out.String(), err
}

func (t *implCertificateService) selfCreate(args []string) (string, error) {
	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert self create cn", t.Application.Name()), nil
	}
	cn := strings.ToLower(args[0])
	if err := t.CreateSelfSigner(cn, false); err != nil {
		return "", err
	} else {
		return fmt.Sprintf("Created Self Signer with name '%s'", cn), nil
	}
}

func (t *implCertificateService) selfUpload(args []string) (string, error) {
	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert self upload dump_file_json", t.Application.Name()), nil
	}
	jsonContents, err := ioutil.ReadFile(args[0])
	if err != nil {
		return "", err
	}

	opts := protojson.UnmarshalOptions{DiscardUnknown: true}

	signer := new(sprintpb.SelfSigner)
	err = opts.Unmarshal(jsonContents, signer)
	if err != nil {
		return "", err
	}

	if signer.Name == "" {
		return "", errors.New("empty name in self signer")
	}

	_, err = t.CertificateIssueService.LoadIssuer(signer)
	if err != nil {
		return "", errors.Wrapf(err, "parse self issuer '%s'", signer.Name)
	}

	if err := t.CertificateRepository.SaveSelfSigner(signer); err != nil {
		return "", err
	} else {
		return "OK", nil
	}
}

func (t *implCertificateService) selfDump(args []string) (string, error) {
	if len(args) < 1 {
		return fmt.Sprintf("Usage: ./%s cert self dump cn", t.Application.Name()), nil
	}
	cn := strings.ToLower(args[0])

	entry, err := t.CertificateRepository.FindSelfSigner(cn)
	if err != nil {
		return "", err
	}

	if entry.Name == "" {
		return "", errors.Errorf("self signer '%s' not found", cn)
	}

	return protojson.Format(entry), nil
}



