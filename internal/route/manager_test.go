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

	// subnet is dummy value - Linux skips routing entirely
	err := SetupRoute("dummy")
	if err != nil {
		t.Errorf("SetupRoute on Linux should return nil, got %v", err)
	}
}

func TestSetupRoute_Darwin(t *testing.T) {
	if !IsDarwin() {
		t.Skip("macOS-specific test")
	}

	// subnet is dummy value - only tests OrbStack/docker-mac-net-connect detection
	err := SetupRoute("dummy")
	if CheckOrbStack() || CheckDockerMacNetConnect() {
		if err != nil {
			t.Errorf("SetupRoute should succeed when OrbStack or docker-mac-net-connect is running, got %v", err)
		}
	} else {
		if err == nil {
			t.Error("SetupRoute should fail when neither OrbStack nor docker-mac-net-connect is running")
		}
	}
}

func TestSetupRouteWithConnectivity(t *testing.T) {
	if !IsDarwin() {
		t.Skip("macOS-specific test")
	}

	// Test with actual connectivity - requires running container
	// This is an integration test, skip if no containers running
	testIP := "192.168.156.4" // hae_app container
	if !checkConnectivity(testIP) {
		t.Skipf("No container at %s, skipping connectivity test", testIP)
	}

	err := SetupRouteWithTest("192.168.156.0/24", testIP)
	if err != nil {
		t.Errorf("SetupRouteWithTest should succeed with reachable IP, got %v", err)
	}
}

func TestRemoveRoute(t *testing.T) {
	// RemoveRoute is a no-op, subnet value doesn't matter
	err := RemoveRoute("dummy")
	if err != nil {
		t.Errorf("RemoveRoute should always return nil, got %v", err)
	}
}

func TestPrintRouteInfo(t *testing.T) {
	// Just ensure it doesn't panic, subnet value doesn't matter
	PrintRouteInfo("dummy")
}
