//go:build !windows

package startup

import (
	"errors"

	"blockads-desktop/internal/models"
)

type StubManager struct{}

func newPlatformManager() Manager {
	return &StubManager{}
}

func (s *StubManager) GetStatus() (*models.StartupStatus, error) {
	return &models.StartupStatus{
		Supported:      false,
		Enabled:        false,
		ExecutablePath: "",
		IsPathValid:    false,
		Message:        "Automatic Windows startup is not supported on this platform.",
	}, nil
}

func (s *StubManager) SetEnabled(enabled bool) error {
	return errors.New("platform does not support Windows startup registry integration")
}
