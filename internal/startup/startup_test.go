package startup

import (
	"runtime"
	"testing"
)

func TestStartupManager_IsolatedKey(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping Windows registry test on non-windows platform")
	}

	testSubkey := `Software\BlockAdsDesktopTest`
	testValName := "TestAppStartup"

	mgr := NewWindowsManagerWithKey(testSubkey, testValName)

	// Ensure cleanup
	t.Cleanup(func() {
		_ = mgr.SetEnabled(false)
	})

	// 1. Initial status should be disabled
	status, err := mgr.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error getting status: %v", err)
	}
	if status.Enabled {
		t.Errorf("expected initially disabled")
	}

	// 2. Enable startup
	if err := mgr.SetEnabled(true); err != nil {
		t.Fatalf("failed to enable startup: %v", err)
	}

	// 3. Status should now be enabled and path valid
	status, err = mgr.GetStatus()
	if err != nil {
		t.Fatalf("failed to get status after enable: %v", err)
	}
	if !status.Enabled {
		t.Errorf("expected enabled=true, got false")
	}
	if !status.IsPathValid {
		t.Errorf("expected path to be valid, message: %s", status.Message)
	}

	// 4. Disable startup
	if err := mgr.SetEnabled(false); err != nil {
		t.Fatalf("failed to disable startup: %v", err)
	}

	// 5. Status should be disabled again
	status, err = mgr.GetStatus()
	if err != nil {
		t.Fatalf("failed to get status after disable: %v", err)
	}
	if status.Enabled {
		t.Errorf("expected disabled after SetEnabled(false)")
	}
}
