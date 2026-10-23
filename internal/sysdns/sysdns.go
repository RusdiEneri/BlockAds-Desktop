package sysdns

// AdapterBackup represents saved DNS configuration for a network interface before BlockAds intervened.
type AdapterBackup struct {
	InterfaceAlias string   `json:"interfaceAlias"`
	InterfaceIndex int      `json:"interfaceIndex"`
	WasDHCP        bool     `json:"wasDhcp"`
	IPv4Servers    []string `json:"ipv4Servers"`
	IPv6Servers    []string `json:"ipv6Servers"`
}

// Manager controls operating system DNS server routing and cache flushing.
type Manager interface {
	// ApplyBlockAdsDNS forces active network adapters to route DNS through BlockAds (127.0.0.1 and ::1).
	ApplyBlockAdsDNS() error

	// RestoreDefaultDNS restores original DNS configuration (DHCP or previous static servers).
	RestoreDefaultDNS() error

	// IsActive reports whether active internet adapters are currently configured to 127.0.0.1.
	IsActive() (bool, error)

	// FlushDNSCache flushes the operating system DNS resolver cache.
	FlushDNSCache() error

	// GetBackups returns the currently recorded adapter configurations.
	GetBackups() []AdapterBackup
}

// DefaultManager creates a platform-appropriate Manager instance.
func DefaultManager() Manager {
	return newPlatformManager()
}
