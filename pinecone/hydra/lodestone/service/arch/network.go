package arch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
)

const NetworkMetadataVersion = 1

type NetworkEndpoint struct {
	Protocol string `json:"protocol"`
	Role     string `json:"role,omitempty"`
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	Path     string `json:"path,omitempty"`
	Address  string `json:"address,omitempty"`
}

type NetworkPort struct {
	Protocol string `json:"protocol"`
	Role     string `json:"role,omitempty"`
	Port     int32  `json:"port"`
}

type NetworkMetadata struct {
	Version     int               `json:"version"`
	LANIPs      []string          `json:"lanIps"`
	PublicIPs   []string          `json:"publicIps"`
	Domains     []string          `json:"domains"`
	PublicPorts []NetworkPort     `json:"publicPorts"`
	Endpoints   []NetworkEndpoint `json:"endpoints"`
}

type NetworkDiscoveryConfig struct {
	AdvertiseHost    string
	IncludeLoopback  bool
	IncludeLinkLocal bool
}

func NewNetworkEndpoint(protocol string, role string, host string, port int32, path string) NetworkEndpoint {
	return NetworkEndpoint{
		Protocol: protocol,
		Role:     role,
		Host:     host,
		Port:     port,
		Path:     path,
		Address:  endpointAddress(protocol, host, port, path),
	}
}

func NewNetworkMetadata() NetworkMetadata {
	return NetworkMetadata{
		Version:     NetworkMetadataVersion,
		LANIPs:      []string{},
		PublicIPs:   []string{},
		Domains:     []string{},
		PublicPorts: []NetworkPort{},
		Endpoints:   []NetworkEndpoint{},
	}
}

func DiscoverLocalNetwork(config NetworkDiscoveryConfig) NetworkMetadata {
	metadata := NewNetworkMetadata()
	addresses := make(map[string]struct{})
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, networkInterface := range interfaces {
			if !config.IncludeLoopback && networkInterface.Flags&net.FlagLoopback != 0 {
				continue
			}
			interfaceAddresses, addressErr := networkInterface.Addrs()
			if addressErr != nil {
				continue
			}
			for _, address := range interfaceAddresses {
				ip := addressIP(address)
				if !isAdvertisableIPv4(ip, config) {
					continue
				}
				addresses[ip] = struct{}{}
			}
		}
	}
	for ip := range addresses {
		metadata.LANIPs = append(metadata.LANIPs, ip)
	}
	sort.Strings(metadata.LANIPs)
	if host := strings.TrimSpace(config.AdvertiseHost); host != "" && isAdvertisableIPv4(host, config) {
		metadata.LANIPs = []string{host}
	}
	return metadata
}

func (metadata NetworkMetadata) WithEndpoint(endpoint NetworkEndpoint) NetworkMetadata {
	if metadata.Version == 0 {
		metadata.Version = NetworkMetadataVersion
	}
	metadata.Endpoints = append(metadata.Endpoints, endpoint)
	return metadata
}

func (endpoint NetworkEndpoint) ServiceEndpoint() ServiceEndpoint {
	return ServiceEndpoint{
		Protocol: endpoint.Protocol,
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Path:     endpoint.Path,
		Address:  endpoint.Address,
	}
}

func MergeRuntimeNodeMetadata(metadata any, network *NetworkMetadata) (map[string]any, error) {
	result := make(map[string]any)
	if metadata != nil {
		switch value := metadata.(type) {
		case map[string]any:
			for key, item := range value {
				result[key] = item
			}
		case string:
			if strings.TrimSpace(value) != "" {
				if err := json.Unmarshal([]byte(value), &result); err != nil {
					return nil, fmt.Errorf("decode runtime node metadata: %w", err)
				}
			}
		default:
			return nil, errors.New("runtime node metadata must be map[string]any or JSON string when network metadata is enabled")
		}
	}
	if network != nil {
		if network.Version == 0 {
			network.Version = NetworkMetadataVersion
		}
		result["network"] = network
	}
	return result, nil
}

func addressIP(address net.Addr) string {
	switch value := address.(type) {
	case *net.IPNet:
		return value.IP.To4().String()
	case *net.IPAddr:
		return value.IP.To4().String()
	default:
		return ""
	}
}

func isAdvertisableIPv4(raw string, config NetworkDiscoveryConfig) bool {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil || ip.To4() == nil {
		return false
	}
	ip = ip.To4()
	if !config.IncludeLoopback && ip.IsLoopback() {
		return false
	}
	if !config.IncludeLinkLocal && ip.IsLinkLocalUnicast() {
		return false
	}
	return true
}

func endpointAddress(protocol string, host string, port int32, path string) string {
	protocol = strings.TrimSpace(protocol)
	host = strings.TrimSpace(host)
	path = strings.TrimSpace(path)
	if protocol == "" || host == "" || port <= 0 {
		return ""
	}
	if path == "" {
		return fmt.Sprintf("%s://%s:%d", strings.ToLower(protocol), host, port)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("%s://%s:%d%s", strings.ToLower(protocol), host, port, path)
}
