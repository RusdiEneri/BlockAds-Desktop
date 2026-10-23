//go:build !windows

package sysdns

type otherManager struct{}

func newPlatformManager() Manager {
	return &otherManager{}
}

func (m *otherManager) ApplyBlockAdsDNS() error {
	return nil
}

func (m *otherManager) RestoreDefaultDNS() error {
	return nil
}

func (m *otherManager) IsActive() (bool, error) {
	return false, nil
}

func (m *otherManager) FlushDNSCache() error {
	return nil
}

func (m *otherManager) GetBackups() []AdapterBackup {
	return nil
}
