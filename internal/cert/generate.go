package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	keyBits    = 2048
	validYears = 10
)

// CertInfo holds certificate subject information
type CertInfo struct {
	Country      string
	State        string
	Locality     string
	Organization string
	OrgUnit      string
	Email        string
}

// DefaultCertInfo returns default certificate info
func DefaultCertInfo() CertInfo {
	return CertInfo{
		Country:      "US",
		State:        "CA",
		Locality:     "MV",
		Organization: "Docker Bootapp",
		OrgUnit:      "Development",
		Email:        "dev@localhost",
	}
}

// GenerateCert creates a self-signed certificate for the given domain
func GenerateCert(domain, certDir string, info CertInfo) error {
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{info.Country},
			Province:           []string{info.State},
			Locality:           []string{info.Locality},
			Organization:       []string{info.Organization},
			OrganizationalUnit: []string{info.OrgUnit},
			CommonName:         domain,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(validYears, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{domain}, // SAN
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	sslname := filepath.Join(certDir, domain)

	// Save .crt
	certFile, _ := os.Create(sslname + ".crt")
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	// Save .key
	keyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	keyFile, _ := os.Create(sslname + ".key")
	pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER})
	keyFile.Close()

	// Save .pem (cert + key)
	pemFile, _ := os.Create(sslname + ".pem")
	pem.Encode(pemFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	pem.Encode(pemFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER})
	pemFile.Close()

	return nil
}

// CertExists checks if certificate exists
func CertExists(domain, certDir string) bool {
	_, err := os.Stat(filepath.Join(certDir, domain+".crt"))
	return err == nil
}

// ListCerts lists all certificates
func ListCerts(certDir string) ([]string, error) {
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(certDir)
	if err != nil {
		return nil, err
	}

	var domains []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && filepath.Ext(name) == ".crt" {
			domains = append(domains, name[:len(name)-4])
		}
	}
	return domains, nil
}

// RemoveCert removes certificate files
func RemoveCert(domain, certDir string) error {
	for _, ext := range []string{".crt", ".key", ".pem"} {
		os.Remove(filepath.Join(certDir, domain+ext))
	}
	return nil
}
