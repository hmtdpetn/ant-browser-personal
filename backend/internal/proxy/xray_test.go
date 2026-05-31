package proxy

import "testing"

func TestXrayRegisterBridgeStoresNewBridge(t *testing.T) {
	t.Parallel()

	manager := &XrayManager{
		Bridges: make(map[string]*XrayBridge),
	}
	bridge := &XrayBridge{
		NodeKey: "node-a",
		Port:    21001,
		Running: true,
	}

	socksURL, reused := manager.registerBridge("node-a", bridge, false)
	if reused {
		t.Fatalf("expected new bridge registration, got reused with %q", socksURL)
	}
	if socksURL != "" {
		t.Fatalf("expected empty socksURL for new bridge registration, got %q", socksURL)
	}
	if manager.Bridges["node-a"] != bridge {
		t.Fatalf("bridge was not stored in manager")
	}
}

func TestXrayRegisterBridgeIgnoresSamePointer(t *testing.T) {
	t.Parallel()

	manager := &XrayManager{
		Bridges: make(map[string]*XrayBridge),
	}
	bridge := &XrayBridge{
		NodeKey: "node-a",
		Port:    21001,
		Running: true,
	}
	manager.Bridges["node-a"] = bridge

	socksURL, reused := manager.registerBridge("node-a", bridge, false)
	if reused {
		t.Fatalf("same bridge pointer must not be treated as duplicate, got reused with %q", socksURL)
	}
	if socksURL != "" {
		t.Fatalf("expected empty socksURL when registering same pointer, got %q", socksURL)
	}
	if manager.Bridges["node-a"] != bridge {
		t.Fatalf("bridge mapping changed unexpectedly")
	}
	if bridge.Stopping {
		t.Fatalf("same bridge pointer should not be marked as stopping")
	}
}
