package arch

import (
	"encoding/json"
	"testing"
)

func TestNetworkMetadataUsesRedQueenContract(t *testing.T) {
	metadata := NewNetworkMetadata().WithEndpoint(NewNetworkEndpoint("HTTP", "ManagementConsole", "192.168.1.173", 6781, "/"))
	metadata.PublicPorts = []NetworkPort{{Protocol: "HTTP", Role: "ManagementConsole", Port: 6781}}
	metadata.LANIPs = []string{"192.168.1.173"}

	value, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(value)
	for _, expected := range []string{`"version":1`, `"lanIps":["192.168.1.173"]`, `"publicPorts":[{"protocol":"HTTP","role":"ManagementConsole","port":6781}]`, `"endpoints"`} {
		if !contains(actual, expected) {
			t.Fatalf("expected %q in %s", expected, actual)
		}
	}
}

func TestMergeRuntimeNodeMetadataOnlyAddsNetworkWhenRequested(t *testing.T) {
	metadata := map[string]any{"role": "edge"}
	withoutNetwork, err := MergeRuntimeNodeMetadata(metadata, nil)
	if err != nil || withoutNetwork["role"] != "edge" {
		t.Fatalf("unexpected metadata without network: %#v, %v", withoutNetwork, err)
	}
	withNetwork, err := MergeRuntimeNodeMetadata(metadata, &NetworkMetadata{Version: NetworkMetadataVersion})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := withNetwork["network"]; !ok {
		t.Fatalf("network metadata was not injected: %#v", withNetwork)
	}
}

func contains(value string, expected string) bool {
	for index := 0; index+len(expected) <= len(value); index++ {
		if value[index:index+len(expected)] == expected {
			return true
		}
	}
	return false
}
