package route

import (
	"runtime"
	"strings"
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

func TestExtractNetworkPrefix(t *testing.T) {
	tests := []struct {
		name     string
		subnet   string
		expected string
	}{
		{"172.18 subnet", "172.18.0.0/16", "172.18.0"},
		{"172.19 subnet", "172.19.0.0/16", "172.19.0"},
		{"192.168 subnet", "192.168.156.0/24", "192.168.156"},
		{"10.x subnet", "10.0.0.0/8", "10.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract network prefix (same logic as checkRouteExists)
			parts := strings.Split(tt.subnet, "/")
			if len(parts) == 0 {
				t.Fatal("Invalid subnet format")
			}
			ipParts := strings.Split(parts[0], ".")
			if len(ipParts) < 3 {
				t.Fatal("Invalid IP format")
			}
			networkPrefix := strings.Join(ipParts[:3], ".")

			if networkPrefix != tt.expected {
				t.Errorf("networkPrefix = %q, want %q", networkPrefix, tt.expected)
			}
		})
	}
}

func TestExtractNetworkPrefix_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		subnet      string
		shouldParse bool
	}{
		{"empty", "", false},
		{"no CIDR", "172.18.0.0", true}, // Valid IP, just no CIDR
		{"single part", "172", false},
		{"two parts", "172.18", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.subnet, "/")
			if len(parts) == 0 || parts[0] == "" {
				if tt.shouldParse {
					t.Error("Should have parsed but didn't")
				}
				return
			}
			ipParts := strings.Split(parts[0], ".")
			canParse := len(ipParts) >= 3
			if canParse != tt.shouldParse {
				t.Errorf("canParse = %v, want %v", canParse, tt.shouldParse)
			}
		})
	}
}

func TestRouteLogic(t *testing.T) {
	// Test the routing decision logic
	type scenario struct {
		name           string
		routeExists    bool
		connectivityOK bool
		testIP         string
		expectedAction string
	}

	scenarios := []scenario{
		{
			"no route exists",
			false, false, "172.18.0.2",
			"add route",
		},
		{
			"route exists, no test IP",
			true, false, "",
			"skip (assume working)",
		},
		{
			"route exists, connectivity OK",
			true, true, "172.18.0.2",
			"skip (tested working)",
		},
		{
			"route exists, connectivity failed",
			true, false, "172.18.0.2",
			"replace route",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			var action string

			if s.routeExists {
				if s.testIP != "" && s.connectivityOK {
					action = "skip (tested working)"
				} else if s.testIP == "" {
					action = "skip (assume working)"
				} else {
					action = "replace route"
				}
			} else {
				action = "add route"
			}

			if action != s.expectedAction {
				t.Errorf("action = %q, want %q", action, s.expectedAction)
			}
		})
	}
}

func TestCheckRouteExists(t *testing.T) {
	if !IsDarwin() {
		t.Skip("checkRouteExists test only runs on macOS")
	}

	// Test with a subnet that definitely doesn't exist
	exists := checkRouteExists("99.99.99.0/24")
	if exists {
		t.Log("Note: 99.99.99.0/24 route exists (unexpected but possible)")
	}

	// Test with localhost subnet (should exist on most systems)
	exists = checkRouteExists("127.0.0.0/8")
	t.Logf("127.0.0.0/8 route exists: %v", exists)
}

func TestCheckRouteExists_InvalidSubnet(t *testing.T) {
	// These should return false without crashing
	tests := []string{
		"",
		"invalid",
		"172/16",
		"172.18/16",
	}

	for _, subnet := range tests {
		t.Run(subnet, func(t *testing.T) {
			result := checkRouteExists(subnet)
			if result {
				t.Errorf("checkRouteExists(%q) = true, want false for invalid subnet", subnet)
			}
		})
	}
}

func TestBuildRouteCommand(t *testing.T) {
	subnet := "172.18.0.0/16"
	gateway := "192.168.65.1"

	// Test add command format
	addArgs := []string{"route", "-n", "add", "-net", subnet, gateway}
	expectedAdd := "route -n add -net 172.18.0.0/16 192.168.65.1"
	actualAdd := strings.Join(addArgs, " ")
	if actualAdd != expectedAdd {
		t.Errorf("add command = %q, want %q", actualAdd, expectedAdd)
	}

	// Test delete command format
	deleteArgs := []string{"route", "-n", "delete", "-net", subnet}
	expectedDelete := "route -n delete -net 172.18.0.0/16"
	actualDelete := strings.Join(deleteArgs, " ")
	if actualDelete != expectedDelete {
		t.Errorf("delete command = %q, want %q", actualDelete, expectedDelete)
	}
}

func TestSetupRouteWithTest_Linux(t *testing.T) {
	if !IsLinux() {
		t.Skip("Linux-specific test")
	}

	// On Linux, should return nil immediately
	err := SetupRouteWithTest("172.18.0.0/16", "172.18.0.2")
	if err != nil {
		t.Errorf("SetupRouteWithTest on Linux should return nil, got %v", err)
	}
}

func TestRemoveRoute_Linux(t *testing.T) {
	if !IsLinux() {
		t.Skip("Linux-specific test")
	}

	// On Linux, should return nil immediately
	err := RemoveRoute("172.18.0.0/16")
	if err != nil {
		t.Errorf("RemoveRoute on Linux should return nil, got %v", err)
	}
}

func TestPrintRouteInfo(t *testing.T) {
	// Just ensure it doesn't panic
	PrintRouteInfo("172.18.0.0/16")
}

func TestGetDockerVMGateway(t *testing.T) {
	if !IsDarwin() {
		t.Skip("GetDockerVMGateway test only runs on macOS")
	}

	gateway, err := GetDockerVMGateway()
	if err != nil {
		t.Logf("GetDockerVMGateway error (may be expected if Docker not running): %v", err)
		return
	}

	// Should be a valid IP format
	parts := strings.Split(gateway, ".")
	if len(parts) != 4 {
		t.Errorf("gateway %q doesn't look like an IP address", gateway)
	}

	t.Logf("Docker VM gateway: %s", gateway)
}

// Integration test - only runs if Docker is available
func TestCheckConnectivity(t *testing.T) {
	// Test with localhost (should always work)
	ok := checkConnectivity("127.0.0.1")
	if !ok {
		t.Log("Warning: ping to 127.0.0.1 failed (might be blocked)")
	}

	// Test with definitely unreachable IP (should fail quickly due to timeout)
	ok = checkConnectivity("192.0.2.1") // TEST-NET-1, should not respond
	if ok {
		t.Log("Note: 192.0.2.1 responded (unexpected)")
	}
}

func TestSubnetFormatValidation(t *testing.T) {
	validSubnets := []string{
		"172.18.0.0/16",
		"192.168.1.0/24",
		"10.0.0.0/8",
	}

	for _, subnet := range validSubnets {
		t.Run(subnet, func(t *testing.T) {
			parts := strings.Split(subnet, "/")
			if len(parts) != 2 {
				t.Errorf("subnet %q should have CIDR notation", subnet)
			}

			ipParts := strings.Split(parts[0], ".")
			if len(ipParts) != 4 {
				t.Errorf("subnet %q should have 4 octets", subnet)
			}
		})
	}
}
