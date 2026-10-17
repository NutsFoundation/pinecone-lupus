package arch

import (
	"strings"
	"testing"
)

func TestNormalizeServiceConfigEndpointAliases(t *testing.T) {
	legacy := ServiceEndpoint{Protocol: "http", Host: "localhost", Port: 8080, Path: "/api", Address: "http://localhost:8080/api"}
	advertised := ServiceEndpoint{Protocol: "HTTP", Host: "LOCALHOST", Port: 8080, Path: "/api", Address: "http://LOCALHOST:8080/api"}
	tests := []struct {
		name       string
		legacy     ServiceEndpoint
		advertised ServiceEndpoint
		want       ServiceEndpoint
		wantError  bool
	}{
		{name: "absent"},
		{name: "legacy", legacy: legacy, want: legacy},
		{name: "advertised", advertised: advertised, want: advertised},
		{name: "equal normalized", legacy: legacy, advertised: advertised, want: advertised},
		{name: "trim only", advertised: ServiceEndpoint{Protocol: " HTTP ", Host: " LOCALHOST ", Port: 8080, Path: " /api ", Address: " http://LOCALHOST:8080/api "}, want: advertised},
		{name: "conflict", legacy: legacy, advertised: ServiceEndpoint{Host: "other"}, wantError: true},
		{name: "address not derived", legacy: ServiceEndpoint{Protocol: "http", Host: "localhost", Port: 8080}, want: ServiceEndpoint{Protocol: "http", Host: "localhost", Port: 8080}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := NormalizeServiceConfig(ServiceLodestoneConfig{Endpoint: test.legacy, AdvertisedEndpoint: test.advertised, RPC: RPCConfig{Insecure: true}})
			if (err != nil) != test.wantError {
				t.Fatalf("NormalizeServiceConfig() error = %v, want error %v", err, test.wantError)
			}
			if err == nil && (config.Endpoint != test.want || config.AdvertisedEndpoint != test.want) {
				t.Fatalf("resolved aliases = %#v, %#v; want %#v", config.Endpoint, config.AdvertisedEndpoint, test.want)
			}
		})
	}
}

func TestNormalizeServiceConfigMetadata(t *testing.T) {
	for _, metadata := range []string{"", ` {"instance":"one"} `, `[]`, `null`} {
		config, err := NormalizeServiceConfig(ServiceLodestoneConfig{MetadataJSON: metadata, RPC: RPCConfig{Insecure: true}})
		if err != nil || config.MetadataJSON != metadata {
			t.Fatalf("metadata %q changed or rejected: %q, %v", metadata, config.MetadataJSON, err)
		}
	}
	for _, metadata := range []string{" ", "{invalid", `{"x":1} {"y":2}`} {
		if _, err := NormalizeServiceConfig(ServiceLodestoneConfig{MetadataJSON: metadata, RPC: RPCConfig{Insecure: true}}); err == nil || !strings.Contains(err.Error(), "metadataJSON") {
			t.Fatalf("invalid metadata %q should fail: %v", metadata, err)
		}
	}
}

func TestValidateRPCTransport(t *testing.T) {
	tests := []struct {
		name      string
		config    RPCConfig
		wantError bool
	}{
		{name: "neither", wantError: true},
		{name: "plaintext", config: RPCConfig{Insecure: true}},
		{name: "TLS", config: RPCConfig{TLS: RPCTLSConfig{Enable: true}}},
		{name: "both", config: RPCConfig{Insecure: true, TLS: RPCTLSConfig{Enable: true}}, wantError: true},
		{name: "disabled server name", config: RPCConfig{Insecure: true, TLS: RPCTLSConfig{ServerName: "server"}}, wantError: true},
		{name: "disabled CA", config: RPCConfig{Insecure: true, TLS: RPCTLSConfig{CAPath: "ca.pem"}}, wantError: true},
		{name: "disabled certificate", config: RPCConfig{Insecure: true, TLS: RPCTLSConfig{CertificatePath: "cert.pem", PrivateKeyPath: "key.pem"}}, wantError: true},
		{name: "missing key", config: RPCConfig{TLS: RPCTLSConfig{Enable: true, CertificatePath: "cert.pem"}}, wantError: true},
		{name: "missing certificate", config: RPCConfig{TLS: RPCTLSConfig{Enable: true, PrivateKeyPath: "key.pem"}}, wantError: true},
		{name: "certificate pair", config: RPCConfig{TLS: RPCTLSConfig{Enable: true, CertificatePath: "cert.pem", PrivateKeyPath: "key.pem"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateRPCTransport(test.config); (err != nil) != test.wantError {
				t.Fatalf("ValidateRPCTransport() = %v, want error %v", err, test.wantError)
			}
		})
	}
}
