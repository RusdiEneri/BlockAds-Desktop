package sysdns

import (
	"errors"
	"strings"
	"testing"
)

type mockRunner struct {
	calls        [][]string
	probePayload string
	probeErr     error
}

func (m *mockRunner) Run(name string, args ...string) ([]byte, error) {
	cmdLine := append([]string{name}, args...)
	m.calls = append(m.calls, cmdLine)

	if name == "powershell" && len(args) >= 4 && strings.Contains(args[3], "Get-NetIPInterface") {
		if m.probeErr != nil {
			return nil, m.probeErr
		}
		return []byte(m.probePayload), nil
	}

	return []byte("ok"), nil
}

func TestWindowsManager_ApplyAndRestore(t *testing.T) {
	mock := &mockRunner{
		probePayload: `[
			{
				"InterfaceIndex": 19,
				"InterfaceAlias": "Wi-Fi",
				"Dhcp": 1,
				"IPv4": ["10.16.0.1", "1.1.1.1"],
				"IPv6": []
			},
			{
				"InterfaceIndex": 16,
				"InterfaceAlias": "Ethernet",
				"Dhcp": 0,
				"IPv4": ["192.168.1.1"],
				"IPv6": ["2001:db8::1"]
			}
		]`,
	}

	mgr := NewWindowsManager(mock)

	// 1. Initial State
	active, err := mgr.IsActive()
	if err != nil {
		t.Fatalf("unexpected error checking active state: %v", err)
	}
	if active {
		t.Errorf("expected not active initially")
	}

	// 2. Apply BlockAds DNS
	err = mgr.ApplyBlockAdsDNS()
	if err != nil {
		t.Fatalf("failed to apply BlockAds DNS: %v", err)
	}

	backups := mgr.GetBackups()
	if len(backups) != 2 {
		t.Fatalf("expected 2 adapter backups, got %d", len(backups))
	}

	var foundWiFi, foundEth bool
	for _, b := range backups {
		if b.InterfaceIndex == 19 {
			foundWiFi = true
			if !b.WasDHCP {
				t.Errorf("expected Wi-Fi to be marked as DHCP")
			}
			if len(b.IPv4Servers) != 2 {
				t.Errorf("expected 2 original IPv4 servers for Wi-Fi")
			}
		}
		if b.InterfaceIndex == 16 {
			foundEth = true
			if b.WasDHCP {
				t.Errorf("expected Ethernet to be marked as static (not DHCP)")
			}
			if len(b.IPv6Servers) != 1 || b.IPv6Servers[0] != "2001:db8::1" {
				t.Errorf("expected Ethernet IPv6 backup 2001:db8::1, got %v", b.IPv6Servers)
			}
		}
	}
	if !foundWiFi || !foundEth {
		t.Errorf("missing expected adapter in backups")
	}

	// Verify flushdns called
	var flushed bool
	for _, call := range mock.calls {
		if call[0] == "ipconfig" && len(call) > 1 && call[1] == "/flushdns" {
			flushed = true
			break
		}
	}
	if !flushed {
		t.Errorf("expected ipconfig /flushdns to be called")
	}

	// 3. Restore Default DNS
	err = mgr.RestoreDefaultDNS()
	if err != nil {
		t.Fatalf("failed to restore default DNS: %v", err)
	}

	// Verify Wi-Fi received ResetServerAddresses and Ethernet received static IPs
	var resetWiFi, restoredEth bool
	for _, call := range mock.calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, "InterfaceIndex 19 -ResetServerAddresses") {
			resetWiFi = true
		}
		if strings.Contains(joined, "InterfaceIndex 16 -ServerAddresses @('192.168.1.1')") {
			restoredEth = true
		}
	}

	if !resetWiFi {
		t.Errorf("expected Wi-Fi (DHCP) to receive -ResetServerAddresses")
	}
	if !restoredEth {
		t.Errorf("expected Ethernet (static) to receive restored static IP")
	}

	if len(mgr.GetBackups()) != 0 {
		t.Errorf("expected backups to be cleared after restore")
	}
}

func TestWindowsManager_SingleObjectJSON(t *testing.T) {
	mock := &mockRunner{
		probePayload: `{
			"InterfaceIndex": 19,
			"InterfaceAlias": "Wi-Fi",
			"Dhcp": "Enabled",
			"IPv4": ["127.0.0.1"],
			"IPv6": ["::1"]
		}`,
	}

	mgr := NewWindowsManager(mock)
	active, err := mgr.IsActive()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !active {
		t.Errorf("expected active when 127.0.0.1 is present")
	}
}

func TestWindowsManager_NoAdapters(t *testing.T) {
	mock := &mockRunner{
		probePayload: ``,
	}

	mgr := NewWindowsManager(mock)
	err := mgr.ApplyBlockAdsDNS()
	if err == nil {
		t.Errorf("expected error when no adapters found")
	}
}

func TestWindowsManager_ProbeError(t *testing.T) {
	mock := &mockRunner{
		probeErr: errors.New("access denied"),
	}

	mgr := NewWindowsManager(mock)
	err := mgr.ApplyBlockAdsDNS()
	if err == nil {
		t.Errorf("expected error when probe fails")
	}
}

func TestIsDhcpEnabled(t *testing.T) {
	tests := []struct {
		input    any
		expected bool
	}{
		{nil, true},
		{float64(1), true},
		{float64(0), false},
		{int(1), true},
		{int(0), false},
		{true, true},
		{false, false},
		{"Enabled", true},
		{"Disabled", false},
		{"1", true},
		{"0", false},
	}

	for _, tt := range tests {
		got := isDhcpEnabled(tt.input)
		if got != tt.expected {
			t.Errorf("isDhcpEnabled(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
