package arch

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// NormalizeServiceConfig resolves the legacy and advertised endpoint aliases.
// Endpoint-free configurations remain valid for existing SDK callers.
func NormalizeServiceConfig(config ServiceLodestoneConfig) (ServiceLodestoneConfig, error) {
	legacy := trimServiceEndpoint(config.Endpoint)
	advertised := trimServiceEndpoint(config.AdvertisedEndpoint)
	if legacy != (ServiceEndpoint{}) && advertised != (ServiceEndpoint{}) && comparableServiceEndpoint(legacy) != comparableServiceEndpoint(advertised) {
		return config, errors.New("Lodestone.Service.endpoint and advertisedEndpoint conflict")
	}
	selected := advertised
	if selected == (ServiceEndpoint{}) {
		selected = legacy
	}
	config.Endpoint = selected
	config.AdvertisedEndpoint = selected
	if config.MetadataJSON != "" && !json.Valid([]byte(config.MetadataJSON)) {
		return config, errors.New("Lodestone.Service.metadataJSON must be valid JSON")
	}
	if err := ValidateRPCTransport(config.RPC); err != nil {
		return config, err
	}
	return config, nil
}

// ValidateRPCTransport requires an explicit choice between TLS and plaintext.
func ValidateRPCTransport(config RPCConfig) error {
	if config.Insecure == config.TLS.Enable {
		return errors.New("Lodestone.Service.rpc requires exactly one of tls.enable or insecure to be true")
	}
	certificatePath := strings.TrimSpace(config.TLS.CertificatePath)
	privateKeyPath := strings.TrimSpace(config.TLS.PrivateKeyPath)
	if !config.TLS.Enable {
		if strings.TrimSpace(config.TLS.ServerName) != "" || strings.TrimSpace(config.TLS.CAPath) != "" || certificatePath != "" || privateKeyPath != "" {
			return errors.New("Lodestone.Service.rpc TLS options require tls.enable")
		}
		return nil
	}
	if (certificatePath == "") != (privateKeyPath == "") {
		return errors.New("Lodestone.Service.rpc TLS certificatePath and privateKeyPath must be configured together")
	}
	return nil
}

func trimServiceEndpoint(endpoint ServiceEndpoint) ServiceEndpoint {
	endpoint.Protocol = strings.TrimSpace(endpoint.Protocol)
	endpoint.Host = strings.TrimSpace(endpoint.Host)
	endpoint.Path = strings.TrimSpace(endpoint.Path)
	endpoint.Address = strings.TrimSpace(endpoint.Address)
	return endpoint
}

func comparableServiceEndpoint(endpoint ServiceEndpoint) ServiceEndpoint {
	endpoint.Protocol = strings.ToLower(endpoint.Protocol)
	endpoint.Host = strings.ToLower(endpoint.Host)
	if address, err := url.Parse(endpoint.Address); err == nil && address.Scheme != "" && address.Host != "" {
		address.Scheme = strings.ToLower(address.Scheme)
		address.Host = strings.ToLower(address.Host)
		endpoint.Address = address.String()
	}
	return endpoint
}
