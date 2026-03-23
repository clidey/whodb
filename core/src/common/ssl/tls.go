/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ssl

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/clidey/whodb/core/src/log"
)

// Load resolves a CertificateInput to PEM bytes.
// Priority: Content first, then Path if Content is empty.
// Path-based loading is only safe for profile-based connections where paths
// are admin-controlled. Frontend connections should always use Content.
func (c *CertificateInput) Load() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	if c.Content != "" {
		log.Debug("[SSL] CertificateInput.Load: using Content")
		return []byte(c.Content), nil
	}

	if c.Path != "" {
		log.Debugf("[SSL] CertificateInput.Load: reading from path %s", c.Path)
		data, err := os.ReadFile(c.Path)
		if err != nil {
			log.Warnf("[SSL] CertificateInput.Load: failed to read file %s: %v", c.Path, err)
			return nil, fmt.Errorf("failed to read certificate file %s: %w", c.Path, err)
		}
		log.Debugf("[SSL] CertificateInput.Load: read %d bytes from %s", len(data), c.Path)
		return data, nil
	}

	return nil, nil
}

// IsEmpty returns true if no certificate is configured (neither Content nor Path).
func (c *CertificateInput) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.Content == "" && c.Path == ""
}

// BuildTLSConfig creates a *tls.Config from SSLConfig.
// serverHostname is used for ServerName verification if not explicitly set in SSLConfig.
// Returns nil if SSL is disabled or config is nil.
func BuildTLSConfig(cfg *SSLConfig, serverHostname string) (*tls.Config, error) {
	if cfg == nil || cfg.Mode == SSLModeDisabled || cfg.Mode == "" {
		return nil, nil
	}

	tlsConfig := &tls.Config{}

	// Handle modes that skip verification
	switch cfg.Mode {
	case SSLModeInsecure, SSLModePreferred, SSLModeRequired:
		tlsConfig.InsecureSkipVerify = true
		return tlsConfig, nil
	case SSLModeEnabled:
		// Enabled mode: use TLS with verification, but CA cert is optional
		// If no CA cert provided, use system root CAs
		if !cfg.CACert.IsEmpty() {
			if err := loadRootCAs(tlsConfig, &cfg.CACert); err != nil {
				return nil, err
			}
		}
		// Load client certs if provided
		if err := loadClientCerts(tlsConfig, &cfg.ClientCert, &cfg.ClientKey); err != nil {
			return nil, err
		}
		return tlsConfig, nil
	}

	// For verify-ca and verify-identity modes, CA certificate handling
	var rootCAs *x509.CertPool
	if !cfg.CACert.IsEmpty() {
		if err := loadRootCAs(tlsConfig, &cfg.CACert); err != nil {
			return nil, err
		}
		rootCAs = tlsConfig.RootCAs
	}

	// Load client certificates if provided (mutual TLS)
	if err := loadClientCerts(tlsConfig, &cfg.ClientCert, &cfg.ClientKey); err != nil {
		return nil, err
	}

	// Handle verify-ca vs verify-identity
	if cfg.Mode == SSLModeVerifyCA {
		// verify-ca: Verify cert chain but NOT hostname
		// We use InsecureSkipVerify + VerifyConnection to manually verify chain only
		tlsConfig.InsecureSkipVerify = true
		if rootCAs != nil {
			tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
				if len(cs.PeerCertificates) == 0 {
					return fmt.Errorf("no peer certificates presented")
				}
				opts := x509.VerifyOptions{
					Roots:         rootCAs,
					Intermediates: x509.NewCertPool(),
				}
				// Add intermediate certs
				for _, cert := range cs.PeerCertificates[1:] {
					opts.Intermediates.AddCert(cert)
				}
				_, err := cs.PeerCertificates[0].Verify(opts)
				if err != nil {
					log.Warnf("[SSL] verify-ca: certificate chain verification failed: %v", err)
				}
				return err
			}
		}
	} else if cfg.Mode == SSLModeVerifyIdentity {
		// verify-identity: Verify cert chain AND hostname
		if cfg.ServerName != "" {
			tlsConfig.ServerName = cfg.ServerName
		} else if serverHostname != "" {
			tlsConfig.ServerName = serverHostname
		}
	}

	return tlsConfig, nil
}

// loadRootCAs loads CA certificates into the TLS config's RootCAs pool.
func loadRootCAs(tlsConfig *tls.Config, caCert *CertificateInput) error {
	caPEM, err := caCert.Load()
	if err != nil {
		return fmt.Errorf("failed to load CA certificate: %w", err)
	}
	if caPEM == nil {
		return nil
	}

	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(caPEM) {
		return fmt.Errorf("failed to parse CA certificate PEM")
	}
	tlsConfig.RootCAs = rootCAs
	return nil
}

// loadClientCerts loads client certificate and key for mutual TLS.
func loadClientCerts(tlsConfig *tls.Config, clientCert, clientKey *CertificateInput) error {
	if clientCert.IsEmpty() || clientKey.IsEmpty() {
		log.Debug("[SSL] loadClientCerts: no client cert/key provided, skipping")
		return nil
	}

	log.Debug("[SSL] loadClientCerts: loading client certificate")
	certPEM, err := clientCert.Load()
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}
	log.Debugf("[SSL] loadClientCerts: cert PEM loaded, %d bytes, starts with: %.50s", len(certPEM), string(certPEM))

	log.Debug("[SSL] loadClientCerts: loading client key")
	keyPEM, err := clientKey.Load()
	if err != nil {
		return fmt.Errorf("failed to load client key: %w", err)
	}
	log.Debugf("[SSL] loadClientCerts: key PEM loaded, %d bytes, starts with: %.50s", len(keyPEM), string(keyPEM))

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Warnf("[SSL] loadClientCerts: X509KeyPair failed: %v", err)
		return fmt.Errorf("failed to load client key pair: %w", err)
	}

	log.Debug("[SSL] loadClientCerts: client certificate loaded successfully")
	tlsConfig.Certificates = []tls.Certificate{cert}
	return nil
}
