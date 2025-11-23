package route

import (
	"testing"
)

func TestCheckRouteExists(t *testing.T) {
	// Test with current subnet
	exists := checkRouteExists("192.168.156.0/24")
	t.Logf("Route exists for 192.168.156.0/24: %v", exists)
}

func TestCheckConnectivity(t *testing.T) {
	// Test connectivity to container
	ok := checkConnectivity("192.168.156.4")
	t.Logf("Connectivity to 192.168.156.4: %v", ok)
}

func TestSetupRouteLogic(t *testing.T) {
	subnet := "192.168.156.0/24"
	testIP := "192.168.156.4"

	routeExists := checkRouteExists(subnet)
	connectivity := checkConnectivity(testIP)

	t.Logf("Route exists: %v", routeExists)
	t.Logf("Connectivity: %v", connectivity)

	if routeExists && connectivity {
		t.Log("RESULT: Skip route setup (existing route works)")
	} else if routeExists && !connectivity {
		t.Log("RESULT: Replace route (existing route broken)")
	} else {
		t.Log("RESULT: Add new route")
	}
}
