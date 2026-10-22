//go:build windows

package startup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	"blockads-desktop/internal/models"
)

const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "BlockAds Desktop"
)

// WindowsManager manages the HKCU Run key for user-level startup.
type WindowsManager struct {
	keyPath   string
	valueName string
}

func newPlatformManager() Manager {
	return &WindowsManager{
		keyPath:   runKeyPath,
		valueName: runValueName,
	}
}

// NewWindowsManagerWithKey creates a WindowsManager with customized registry keys (used for test isolation).
func NewWindowsManagerWithKey(keyPath, valueName string) *WindowsManager {
	return &WindowsManager{
		keyPath:   keyPath,
		valueName: valueName,
	}
}

// GetStatus checks whether the startup key is configured and points to a valid file.
func (m *WindowsManager) GetStatus() (*models.StartupStatus, error) {
	currentExe, err := os.Executable()
	if err != nil {
		currentExe = ""
	} else {
		currentExe, _ = filepath.Abs(currentExe)
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, m.keyPath, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return &models.StartupStatus{
				Supported:      true,
				Enabled:        false,
				ExecutablePath: currentExe,
				IsPathValid:    true,
				Message:        "Startup launch is currently disabled.",
			}, nil
		}
		return nil, fmt.Errorf("failed to open startup registry key: %w", err)
	}
	defer k.Close()

	val, _, err := k.GetStringValue(m.valueName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return &models.StartupStatus{
				Supported:      true,
				Enabled:        false,
				ExecutablePath: currentExe,
				IsPathValid:    true,
				Message:        "Startup launch is currently disabled.",
			}, nil
		}
		return nil, fmt.Errorf("failed to read startup registry value: %w", err)
	}

	cleanRegisteredPath := strings.Trim(strings.TrimSpace(val), "\"")
	fi, statErr := os.Stat(cleanRegisteredPath)
	pathExists := statErr == nil && !fi.IsDir()

	status := &models.StartupStatus{
		Supported:      true,
		Enabled:        true,
		ExecutablePath: cleanRegisteredPath,
		IsPathValid:    pathExists,
	}

	if !pathExists {
		status.Message = fmt.Sprintf("Startup entry points to nonexistent or moved path: %s", cleanRegisteredPath)
	} else if currentExe != "" && !strings.EqualFold(cleanRegisteredPath, currentExe) {
		status.Message = fmt.Sprintf("Startup entry points to a different binary location: %s", cleanRegisteredPath)
	} else {
		status.Message = "Application is configured to start automatically on Windows login."
	}

	return status, nil
}

// SetEnabled creates or deletes the current-user registry Run key.
func (m *WindowsManager) SetEnabled(enabled bool) error {
	if enabled {
		currentExe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot resolve current executable path: %w", err)
		}
		absExe, err := filepath.Abs(currentExe)
		if err != nil {
			absExe = currentExe
		}

		k, _, err := registry.CreateKey(registry.CURRENT_USER, m.keyPath, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("failed to open or create startup registry key: %w", err)
		}
		defer k.Close()

		quoted := fmt.Sprintf("\"%s\"", absExe)
		if err := k.SetStringValue(m.valueName, quoted); err != nil {
			return fmt.Errorf("failed to write startup registry value: %w", err)
		}
		return nil
	}

	// Disable: delete the value
	k, err := registry.OpenKey(registry.CURRENT_USER, m.keyPath, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to open startup registry key: %w", err)
	}
	defer k.Close()

	if err := k.DeleteValue(m.valueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("failed to delete startup registry value: %w", err)
	}

	return nil
}
