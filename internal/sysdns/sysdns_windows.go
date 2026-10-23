//go:build windows

package sysdns

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

// CommandRunner abstracts shell execution for testing.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

type defaultRunner struct{}

func (r *defaultRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.CombinedOutput()
}

// WindowsManager enforces system DNS policies via Windows PowerShell cmdlets.
type WindowsManager struct {
	mu      sync.RWMutex
	runner  CommandRunner
	backups map[int]AdapterBackup
}

func newPlatformManager() Manager {
	return NewWindowsManager(nil)
}

// NewWindowsManager constructs a WindowsManager with an optional custom runner.
func NewWindowsManager(runner CommandRunner) *WindowsManager {
	if runner == nil {
		runner = &defaultRunner{}
	}
	return &WindowsManager{
		runner:  runner,
		backups: make(map[int]AdapterBackup),
	}
}

type adapterProbeResult struct {
	InterfaceIndex int      `json:"InterfaceIndex"`
	InterfaceAlias string   `json:"InterfaceAlias"`
	Dhcp           any      `json:"Dhcp"` // can be int (1/0) or bool or string
	IPv4           []string `json:"IPv4"`
	IPv6           []string `json:"IPv6"`
}

func isDhcpEnabled(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case float64:
		return v != 0
	case int:
		return v != 0
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "enabled") || v == "1"
	}
	return true
}

func (m *WindowsManager) getActiveAdapters() ([]adapterProbeResult, error) {
	psScript := `Get-NetIPInterface -AddressFamily IPv4 | Where-Object { $_.ConnectionState -eq 'Connected' -and $_.InterfaceAlias -notmatch 'Loopback|Tailscale|vEthernet' } | ForEach-Object {
    $idx = $_.InterfaceIndex
    $alias = $_.InterfaceAlias
    $dhcp = $_.Dhcp
    $v4 = (Get-DnsClientServerAddress -InterfaceIndex $idx -AddressFamily IPv4 -ErrorAction SilentlyContinue).ServerAddresses
    $v6 = (Get-DnsClientServerAddress -InterfaceIndex $idx -AddressFamily IPv6 -ErrorAction SilentlyContinue).ServerAddresses
    [PSCustomObject]@{
        InterfaceIndex = $idx
        InterfaceAlias = $alias
        Dhcp = $dhcp
        IPv4 = @($v4)
        IPv6 = @($v6)
    }
} | ConvertTo-Json -Compress`

	out, err := m.runner.Run("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	if err != nil {
		return nil, fmt.Errorf("failed to probe network adapters: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var results []adapterProbeResult
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &results); err != nil {
			return nil, fmt.Errorf("failed to parse adapter array: %w", err)
		}
	} else if strings.HasPrefix(trimmed, "{") {
		var single adapterProbeResult
		if err := json.Unmarshal([]byte(trimmed), &single); err != nil {
			return nil, fmt.Errorf("failed to parse adapter object: %w", err)
		}
		results = append(results, single)
	}

	return results, nil
}

// ApplyBlockAdsDNS forces active network adapters to route DNS through BlockAds (127.0.0.1 and ::1).
func (m *WindowsManager) ApplyBlockAdsDNS() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	adapters, err := m.getActiveAdapters()
	if err != nil {
		return err
	}
	if len(adapters) == 0 {
		return errors.New("no active physical/connected network adapters found")
	}

	var errs []string
	for _, a := range adapters {
		// Only backup if not already configured to BlockAds
		isAlreadyBlockAds := false
		if len(a.IPv4) > 0 && a.IPv4[0] == "127.0.0.1" {
			isAlreadyBlockAds = true
		}

		if !isAlreadyBlockAds {
			if _, exists := m.backups[a.InterfaceIndex]; !exists {
				m.backups[a.InterfaceIndex] = AdapterBackup{
					InterfaceAlias: a.InterfaceAlias,
					InterfaceIndex: a.InterfaceIndex,
					WasDHCP:        isDhcpEnabled(a.Dhcp),
					IPv4Servers:    a.IPv4,
					IPv6Servers:    a.IPv6,
				}
			}
		}

		// Set IPv4 DNS to 127.0.0.1
		setCmd := fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceIndex %d -ServerAddresses @('127.0.0.1')`, a.InterfaceIndex)
		if out, err := m.runner.Run("powershell", "-NoProfile", "-NonInteractive", "-Command", setCmd); err != nil {
			errs = append(errs, fmt.Sprintf("failed to set IPv4 DNS on %s (#%d): %v (%s)", a.InterfaceAlias, a.InterfaceIndex, err, strings.TrimSpace(string(out))))
		}

		// Set IPv6 DNS to ::1 (to prevent IPv6 DNS leaks)
		setV6Cmd := fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceIndex %d -ServerAddresses @('::1') -ErrorAction SilentlyContinue`, a.InterfaceIndex)
		_, _ = m.runner.Run("powershell", "-NoProfile", "-NonInteractive", "-Command", setV6Cmd)
	}

	_ = m.flushDNSCacheLocked()

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// RestoreDefaultDNS restores original DNS configuration (DHCP or previous static servers).
func (m *WindowsManager) RestoreDefaultDNS() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	adapters, err := m.getActiveAdapters()
	if err != nil {
		// Even if probe fails, try to restore from recorded backups
		adapters = nil
	}

	targetIndices := make(map[int]bool)
	for _, a := range adapters {
		targetIndices[a.InterfaceIndex] = true
	}
	for idx := range m.backups {
		targetIndices[idx] = true
	}

	var errs []string
	for idx := range targetIndices {
		backup, hasBackup := m.backups[idx]
		if hasBackup && !backup.WasDHCP && len(backup.IPv4Servers) > 0 {
			// Restore custom static IPv4 servers
			var quoted []string
			for _, s := range backup.IPv4Servers {
				quoted = append(quoted, fmt.Sprintf("'%s'", s))
			}
			cmd := fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceIndex %d -ServerAddresses @(%s)`, idx, strings.Join(quoted, ","))
			if out, err := m.runner.Run("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd); err != nil {
				errs = append(errs, fmt.Sprintf("failed to restore static IPv4 DNS on adapter #%d: %v (%s)", idx, err, strings.TrimSpace(string(out))))
			}

			// Restore static IPv6 servers if any
			if len(backup.IPv6Servers) > 0 {
				var v6Quoted []string
				for _, s := range backup.IPv6Servers {
					v6Quoted = append(v6Quoted, fmt.Sprintf("'%s'", s))
				}
				v6Cmd := fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceIndex %d -ServerAddresses @(%s) -ErrorAction SilentlyContinue`, idx, strings.Join(v6Quoted, ","))
				_, _ = m.runner.Run("powershell", "-NoProfile", "-NonInteractive", "-Command", v6Cmd)
			} else {
				resetV6 := fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceIndex %d -ResetServerAddresses -ErrorAction SilentlyContinue`, idx)
				_, _ = m.runner.Run("powershell", "-NoProfile", "-NonInteractive", "-Command", resetV6)
			}
		} else {
			// Reset to DHCP
			cmd := fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceIndex %d -ResetServerAddresses`, idx)
			if out, err := m.runner.Run("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd); err != nil {
				errs = append(errs, fmt.Sprintf("failed to reset DNS to DHCP on adapter #%d: %v (%s)", idx, err, strings.TrimSpace(string(out))))
			}
		}
	}

	m.backups = make(map[int]AdapterBackup)
	_ = m.flushDNSCacheLocked()

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// IsActive reports whether active internet adapters are currently configured to 127.0.0.1.
func (m *WindowsManager) IsActive() (bool, error) {
	adapters, err := m.getActiveAdapters()
	if err != nil {
		return false, err
	}
	if len(adapters) == 0 {
		return false, nil
	}
	for _, a := range adapters {
		if len(a.IPv4) > 0 && a.IPv4[0] == "127.0.0.1" {
			return true, nil
		}
	}
	return false, nil
}

// FlushDNSCache flushes the operating system DNS resolver cache.
func (m *WindowsManager) FlushDNSCache() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.flushDNSCacheLocked()
}

func (m *WindowsManager) flushDNSCacheLocked() error {
	_, err := m.runner.Run("ipconfig", "/flushdns")
	return err
}

// GetBackups returns the currently recorded adapter configurations.
func (m *WindowsManager) GetBackups() []AdapterBackup {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []AdapterBackup
	for _, b := range m.backups {
		result = append(result, b)
	}
	return result
}
