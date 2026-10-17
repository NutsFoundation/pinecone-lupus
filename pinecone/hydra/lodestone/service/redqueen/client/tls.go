package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func (c Config) transportCredentials() (credentials.TransportCredentials, error) {
	if err := servicearch.ValidateRPCTransport(servicearch.RPCConfig{Insecure: c.Insecure, TLS: c.TLS}); err != nil {
		return nil, err
	}
	if c.Insecure {
		return insecure.NewCredentials(), nil
	}
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: strings.TrimSpace(c.TLS.ServerName),
	}
	if caPath := strings.TrimSpace(c.TLS.CAPath); caPath != "" {
		certificatePEM, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("read Lodestone RPC TLS CA: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(certificatePEM) {
			return nil, fmt.Errorf("Lodestone RPC TLS CA %q contains no valid certificates", caPath)
		}
		config.RootCAs = roots
	}
	if certificatePath := strings.TrimSpace(c.TLS.CertificatePath); certificatePath != "" {
		certificate, err := tls.LoadX509KeyPair(certificatePath, strings.TrimSpace(c.TLS.PrivateKeyPath))
		if err != nil {
			return nil, fmt.Errorf("load Lodestone RPC TLS client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{certificate}
	}
	return credentials.NewTLS(config), nil
}
