package rpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// TLSConfigKey is used to store a TLS configuration within the connection context
type TLSConfigKey struct{}

const (
	DefaultTLSCertFile = "tlscert.pem"
	DefaultTLSKeyFile  = "tlskey.pem"
)

var (
	validFor = 365 * 24 * time.Hour
	rsaBits  = 2048
)

// TLSConfigFromContext fetches TLS configuration from context.
// Fetched config is to be normally used to configure client connection's transport.
func TLSConfigFromContext(ctx context.Context) *tls.Config {
	config, ok := ctx.Value(TLSConfigKey{}).(*tls.Config)
	if !ok {
		return &tls.Config{}
	}
	return config
}

// MakeTLSContext packs TLS configuration into context.
func MakeTLSConfigContext(config *tls.Config) context.Context {
	return context.WithValue(context.Background(), TLSConfigKey{}, config)
}

func MakeServerTLSConfig(host string, certPath string, keyPath string) (*tls.Config, error) {
	config := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	// Make sure certificate and key files do exist
	if err := EnsureCerts(certPath, keyPath, host); err != nil {
		return config, err
	}

	// Load certificate/key
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err == nil {
		config.Certificates = []tls.Certificate{cert}
	}

	return config, nil
}

// MakeClientTLSConfig is used to configure connection context. Particularly,
// setting up TLS support for connection transport.
//
// For auto-generated certificates, isCA is always true, that's generated
// certificate is used as its own parent when signing.
func MakeClientTLSConfig(certPath string, keyPath string, isCA bool, certNoVerify bool) *tls.Config {
	config := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	// Load certificate/key
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err == nil {
		config.Certificates = []tls.Certificate{cert}

		// Provide CA certs
		if isCA {
			if rootCert, err := ioutil.ReadFile(certPath); err == nil {
				certPool := x509.NewCertPool()
				certPool.AppendCertsFromPEM(rootCert)

				config.RootCAs = certPool
				config.ClientCAs = certPool
			}
		}
	}

	// Allow to skip TLS certificate verification
	if certNoVerify {
		config.InsecureSkipVerify = true
	}

	return config
}

// EnsureCerts check certificate/key files, and auto-generates both if necessary
func EnsureCerts(certPath string, keyPath string, host string) error {
	var certFileExists, keyFileExists bool

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		certFileExists = false
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		keyFileExists = false
	}

	// generate cert and private key (if they are not already available)
	if !certFileExists || !keyFileExists {
		if err := Generate(certPath, keyPath, host); err != nil {
			return err
		}
	}

	return nil
}

func Generate(certPath string, keyPath string, host string) error {
	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("Failed to create certificate: %s", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to open "+certPath+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	glog.V(logger.Info).Infof("TLS: %s written", certPath)

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open "+keyPath+" for writing: %s", err)
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
	glog.V(logger.Info).Infof("TLS: %s written", keyPath)
	return nil
}
