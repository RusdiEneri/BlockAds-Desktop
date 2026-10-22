package startup

import (
	"blockads-desktop/internal/models"
)

// Manager coordinates checking and updating user-session startup registration.
type Manager interface {
	GetStatus() (*models.StartupStatus, error)
	SetEnabled(enabled bool) error
}

// DefaultManager returns the platform-specific startup manager.
func DefaultManager() Manager {
	return newPlatformManager()
}
