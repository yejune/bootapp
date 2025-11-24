package route

import (
	"runtime"
	"testing"
)

func TestIsLinux(t *testing.T) {
	result := IsLinux()
	expected := runtime.GOOS == "linux"
	if result != expected {
		t.Errorf("IsLinux() = %v, want %v (GOOS=%s)", result, expected, runtime.GOOS)
	}
}

func TestIsDarwin(t *testing.T) {
	result := IsDarwin()
	expected := runtime.GOOS == "darwin"
	if result != expected {
		t.Errorf("IsDarwin() = %v, want %v (GOOS=%s)", result, expected, runtime.GOOS)
	}
}

func TestCheckDockerMacNetConnect(t *testing.T) {
	// Just ensure it doesn't panic
	result := CheckDockerMacNetConnect()
	t.Logf("docker-mac-net-connect running: %v", result)
}

func TestSetupRoute_Linux(t *testing.T) {
	if !IsLinux() {
		t.Skip("Linux-specific test")
	}

	err := SetupRouteWithTest("172.18.0.0/16", "172.18.0.2")
	if err != nil {
		t.Errorf("SetupRouteWithTest on Linux should return nil, got %v", err)
	}
}

func TestSetupRoute_Darwin(t *testing.T) {
	if !IsDarwin() {
		t.Skip("macOS-specific test")
	}

	// Test depends on whether docker-mac-net-connect is running
	err := SetupRouteWithTest("172.18.0.0/16", "172.18.0.2")
	if CheckDockerMacNetConnect() {
		if err != nil {
			t.Errorf("SetupRouteWithTest should succeed when docker-mac-net-connect is running, got %v", err)
		}
	} else {
		if err == nil {
			t.Error("SetupRouteWithTest should fail when docker-mac-net-connect is not running")
		}
	}
}

func TestRemoveRoute(t *testing.T) {
	// RemoveRoute is now a no-op
	err := RemoveRoute("172.18.0.0/16")
	if err != nil {
		t.Errorf("RemoveRoute should always return nil, got %v", err)
	}
}

func TestPrintRouteInfo(t *testing.T) {
	// Just ensure it doesn't panic
	PrintRouteInfo("172.18.0.0/16")
}
