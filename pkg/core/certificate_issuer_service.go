/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprintpb"
	"github.com/sprintframework/sprint"
	"go.uber.org/zap"
	"math"
	"math/big"
	"net"
	pkcs12 "software.sslmate.com/src/go-pkcs12"
	"strings"
	"time"
)

type implCertificateIssuerService struct {

	Properties       glue.Properties      `inject`
	ConfigRepository sprint.ConfigRepository `inject`
	Log              *zap.Logger            `inject`

	RsaLen            int                      `value:"tls.certificate.rsa-len,default=2048"`

	Organization      string                   `value:"tls.certificate.organization,default="`
	Country           string                   `value:"tls.certificate.country,default="`
	Province          string                   `value:"tls.certificate.province,default="`
	City              string                   `value:"tls.certificate.city,default="`
	Street            string                   `value:"tls.certificate.street,default="`
	Zip               string                   `value:"tls.certificate.zip,default="`

}

type certificateIssuer struct {
	service   *implCertificateIssuerService
	parent    *certificateIssuer
	cert      *issuedCertificate
}

type issuedCertificate struct {
	certContents []byte
	keyContents  []byte
	x509Cert     *x509.Certificate
	key          crypto.Signer
}

func CertificateIssueService() sprint.CertificateIssueService {
	return &implCertificateIssuerService{}
}

func (t *issuedCertificate) KeyFileContents() []byte {
	return t.keyContents
}

func (t *issuedCertificate) CertFileContents() []byte {
	return t.certContents
}

func (t *issuedCertificate) PrivateKey() crypto.Signer {
	return t.key
}

func (t *issuedCertificate) Certificate() *x509.Certificate {
	return t.x509Cert
}

func (t *implCertificateIssuerService) LoadCertificateDesc() (*sprint.CertificateDesc, error) {

	certName := &sprint.CertificateDesc{
		Organization: t.Organization,
		Country:      t.Country,
		Province:     t.Province,
		City:         t.City,
		Street:       t.Street,
		Zip:          t.Zip,
	}

	return certName, nil
}

func (t *implCertificateIssuerService) CreateIssuer(cn string, info *sprint.CertificateDesc) (sprint.CertificateIssuer, error) {

	rootKeyContents, rootCertContents, err := t.makeRootIssuer(cn, info)
	if err != nil {
		return nil, err
	}

	rootKey, err := readPrivateKey(rootKeyContents)
	if err != nil {
		return nil, errors.Errorf("reading root private key, %v", err)
	}

	rootCert, err := readCert(rootCertContents)
	if err != nil {
		return nil, errors.Errorf("reading root certificate, %v", err)
	}

	equal, err := publicKeysEqual(rootKey.Public(), rootCert.PublicKey)
	if err != nil {
		return nil, errors.Errorf("comparing public keys for root certificate: %s", err)
	} else if !equal {
		return nil, errors.New("public root key in root certificate doesn't match private root key")
	}

	cert := &issuedCertificate{
		certContents: rootCertContents,
		keyContents:  rootKeyContents,
		x509Cert:     rootCert,
		key:          rootKey,
	}
	
	return &certificateIssuer{service: t, parent: nil, cert: cert}, nil
}

func (t *implCertificateIssuerService) LoadIssuer(issuer *sprintpb.SelfSigner) (sprint.CertificateIssuer, error) {
	return t.loadIssuerRecursive(issuer)
}

func (t *implCertificateIssuerService) loadIssuerRecursive(issuer *sprintpb.SelfSigner) (*certificateIssuer, error) {

	key, err := readPrivateKey(issuer.PrivateKey)
	if err != nil {
		return nil, errors.Errorf("reading root private key, %v", err)
	}

	x509Cert, err := readCert(issuer.Certificate)
	if err != nil {
		return nil, errors.Errorf("reading root x509Cert, %v", err)
	}

	equal, err := publicKeysEqual(key.Public(), x509Cert.PublicKey)
	if err != nil {
		return nil, errors.Errorf("comparing public keys for x509Cert: %s", err)
	} else if !equal {
		return nil, errors.New("public key in x509Cert doesn't match private key")
	}

	cert := &issuedCertificate{
		certContents: issuer.Certificate,
		keyContents:  issuer.PrivateKey,
		x509Cert:     x509Cert,
		key:          key,
	}
	
	self := &certificateIssuer {
		service: t,
		cert: cert,
	}

	if issuer.Issuer != nil {
		self.parent, err = t.loadIssuerRecursive(issuer.Issuer)
		if err != nil {
			 return nil, err
		}
	}

	return self, nil
}

func (t *certificateIssuer) Parent() (sprint.CertificateIssuer, bool) {
	return t.parent, t.parent != nil
}

func (t *certificateIssuer) Certificate() sprint.IssuedCertificate {
	return t.cert
}

func (t *certificateIssuer) IssueInterCert(cn string) (sprint.CertificateIssuer, error) {

	interKeyContents, interCertContents, err := t.service.makeIntermediateIssuer(cn, t.cert.x509Cert, t.cert.key)
	if err != nil {
		return nil, err
	}

	interKey, err := readPrivateKey(interKeyContents)
	if err != nil {
		return nil, errors.Errorf("reading private key from tls.certificate.inter.key: %v", err)
	}

	interCert, err := readCert(interCertContents)
	if err != nil {
		return nil, errors.Errorf("reading intermediate CA certificate from tls.certificate.inter.crt: %v", err)
	}

	equal, err := publicKeysEqual(interKey.Public(), interCert.PublicKey)
	if err != nil {
		return nil, errors.Errorf("comparing intermediate public keys: %s", err)
	} else if !equal {
		return nil, errors.New("intermediate public key in CA certificate doesn't match intermediate private key")
	}

	cert := &issuedCertificate{
		certContents: interCertContents,
		keyContents:  interKeyContents,
		x509Cert:     interCert,
		key:          interKey,
	}

	return &certificateIssuer{service: t.service, parent: t, cert: cert}, nil
}

func (t *certificateIssuer) IssueClientCert(cn string, password string) (cert sprint.IssuedCertificate, pfxData []byte, err error) {

	cnName := strings.Replace(cn, "*", "_", -1)
	cnName = strings.Replace(cnName, " ", "-", -1)
	cnName = strings.ToLower(cnName)

	key, keyPemFile, err := t.service.makeKey()
	if err != nil {
		return nil, nil, err
	}

	desc := getCertificateDesc(t.cert.x509Cert)

	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("Client %s %x", cn, serial.Bytes()[:3]),
			Organization:  []string{desc.Organization},
			Country:       []string{desc.Country},
			Province:      []string{desc.Province},
			Locality:      []string{desc.City},
			StreetAddress: []string{desc.Street},
			PostalCode:    []string{desc.Zip},
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, t.cert.x509Cert, key.Public(), t.cert.key)
	if err != nil {
		return nil, nil, err
	}

	var certPemFile bytes.Buffer
	err = pem.Encode(&certPemFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
	if err != nil {
		return nil, nil, err
	}

	x509Cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}

	var caCerts []*x509.Certificate
	for i := t; i != nil; i = i.parent {
		caCerts = append(caCerts, i.cert.x509Cert)
	}

	pfxData, err = pkcs12.Encode(rand.Reader, key, x509Cert, caCerts, password)
	if err != nil {
		return nil, nil, err
	}

	return &issuedCertificate{certContents: certPemFile.Bytes(), keyContents: keyPemFile, x509Cert: x509Cert, key: key}, pfxData, nil
}

func (t *certificateIssuer) IssueServerCert(cn string, domains []string, ipAddresses []net.IP) (cert sprint.IssuedCertificate, err error) {

	if cn == "" {
		if len(domains) > 0 {
			cn = domains[0]
		} else if len(ipAddresses) > 0 {
			cn = ipAddresses[0].String()
		} else {
			return nil, errors.Errorf("must specify at least one domain name or IP address")
		}
	}
	cn = strings.ReplaceAll(cn, "*", "_")

	desc := getCertificateDesc(t.cert.x509Cert)

	key, keyPemFile, err := t.service.makeKey()
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		DNSNames:    domains,
		IPAddresses: ipAddresses,
		Subject: pkix.Name{
			CommonName:    cn,
			Organization:  []string{desc.Organization},
			Country:       []string{desc.Country},
			Province:      []string{desc.Province},
			Locality:      []string{desc.City},
			StreetAddress: []string{desc.Street},
			PostalCode:    []string{desc.Zip},
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		// Set the validity period to 2 years and 30 days, to satisfy the iOS and
		// macOS requirements that all server certificates must have validity
		// shorter than 825 days:
		// https://derflounder.wordpress.com/2019/06/06/new-tls-security-requirements-for-ios-13-and-macos-catalina-10-15/
		NotAfter: time.Now().AddDate(2, 0, 30),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, t.cert.x509Cert, key.Public(), t.cert.key)
	if err != nil {
		return nil, err
	}

	var certPemFile bytes.Buffer
	err = pem.Encode(&certPemFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
	if err != nil {
		return nil, err
	}

	x509Cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	return &issuedCertificate{certContents: certPemFile.Bytes(), keyContents: keyPemFile, x509Cert: x509Cert, key: key}, nil
}

func readPrivateKey(keyContents []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(keyContents)
	if block == nil {
		return nil, fmt.Errorf("no PEM found")
	} else if block.Type != "RSA PRIVATE KEY" && block.Type != "ECDSA PRIVATE KEY" {
		return nil, fmt.Errorf("incorrect PEM type %s", block.Type)
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func readCert(certContents []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certContents)
	if block == nil {
		return nil, fmt.Errorf("no PEM found")
	} else if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("incorrect PEM type %s", block.Type)
	}
	return x509.ParseCertificate(block.Bytes)
}

func publicKeysEqual(a, b interface{}) (bool, error) {
	aBytes, err := x509.MarshalPKIXPublicKey(a)
	if err != nil {
		return false, err
	}
	bBytes, err := x509.MarshalPKIXPublicKey(b)
	if err != nil {
		return false, err
	}
	return bytes.Compare(aBytes, bBytes) == 0, nil
}

func (t *implCertificateIssuerService) makeKey() (*rsa.PrivateKey, []byte, error) {
	var pemFile bytes.Buffer
	key, err := rsa.GenerateKey(rand.Reader, t.RsaLen)
	if err != nil {
		return nil, nil, err
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	err = pem.Encode(&pemFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	})
	if err != nil {
		return nil, nil, err
	}
	return key, pemFile.Bytes(), nil
}

func calculateSKID(pubKey crypto.PublicKey) ([]byte, error) {
	spkiASN1, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	var spki struct {
		Algorithm        pkix.AlgorithmIdentifier
		SubjectPublicKey asn1.BitString
	}
	_, err = asn1.Unmarshal(spkiASN1, &spki)
	if err != nil {
		return nil, err
	}
	skid := sha1.Sum(spki.SubjectPublicKey.Bytes)
	return skid[:], nil
}

func firstElement(arr []string) string {
	if len(arr) == 0 {
		return ""
	}
	return arr[0]
}

func getCertificateDesc(cert *x509.Certificate) *sprint.CertificateDesc {
	return &sprint.CertificateDesc{
		Organization: firstElement(cert.Subject.Organization),
		Country: firstElement(cert.Subject.Country),
		Province: firstElement(cert.Subject.Province),
		City: firstElement(cert.Subject.Locality),
		Street: firstElement(cert.Subject.StreetAddress),
		Zip: firstElement(cert.Subject.PostalCode),
	}
}

func(t *implCertificateIssuerService) makeRootCert(cn string, desc *sprint.CertificateDesc, key crypto.Signer) (*x509.Certificate, []byte, error) {
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, nil, err
	}
	skid, err := calculateSKID(key.Public())
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName:    fmt.Sprintf("Root %s %x", cn, serial.Bytes()[:3]),
			Organization:  []string{desc.Organization},
			Country:       []string{desc.Country},
			Province:      []string{desc.Province},
			Locality:      []string{desc.City},
			StreetAddress: []string{desc.Street},
			PostalCode:    []string{desc.Zip},
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(100, 0, 0),

		SubjectKeyId:   skid,
		AuthorityKeyId: skid,
		KeyUsage:       x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		//ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		//MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		return nil, nil, err
	}
	var file bytes.Buffer
	err = pem.Encode(&file, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	return cert, file.Bytes(), err
}

func (t *implCertificateIssuerService) makeIntermediateCert(cn string, desc *sprint.CertificateDesc, key crypto.Signer, rootCert *x509.Certificate, rootKey crypto.Signer) (*x509.Certificate, []byte, error) {
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, nil, err
	}
	skid, err := calculateSKID(key.Public())
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName:    fmt.Sprintf("Intermediate %s %x", cn, serial.Bytes()[:3]),
			Organization:  []string{desc.Organization},
			Country:       []string{desc.Country},
			Province:      []string{desc.Province},
			Locality:      []string{desc.City},
			StreetAddress: []string{desc.Street},
			PostalCode:    []string{desc.Zip},
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),

		SubjectKeyId:          skid,
		AuthorityKeyId:        skid,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, rootCert, key.Public(), rootKey)
	if err != nil {
		return nil, nil, err
	}

	var file bytes.Buffer
	err = pem.Encode(&file, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	return cert, file.Bytes(), err
}

func (t *implCertificateIssuerService) makeRootIssuer(cn string, desc *sprint.CertificateDesc) ([]byte, []byte, error) {
	key, keyFile, err := t.makeKey()
	if err != nil {
		return nil, nil, err
	}
	_, certFile, err := t.makeRootCert(cn, desc, key)
	if err != nil {
		return nil, nil, err
	}
	return keyFile, certFile, nil
}

func (t *implCertificateIssuerService) makeIntermediateIssuer(cn string, rootCert *x509.Certificate, rootKey crypto.Signer) ([]byte, []byte, error) {
	desc := getCertificateDesc(rootCert)
	key, keyFile, err := t.makeKey()
	if err != nil {
		return nil, nil, err
	}
	_, certFile, err := t.makeIntermediateCert(cn, desc, key, rootCert, rootKey)
	if err != nil {
		return nil, nil, err
	}
	return keyFile, certFile, nil
}

var IPv4Local = net.IPv4(127, 0, 0, 1)

func  (t *implCertificateIssuerService) LocalIPAddresses(addLocalhost bool) ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var list []net.IP
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil {
				if !addLocalhost {
					if bytes.Equal(ip, IPv4Local) || bytes.Equal(ip, net.IPv6loopback) {
						continue
					}
				}
				list = append(list, ip)
			}

		}
	}
	return list, nil
}

