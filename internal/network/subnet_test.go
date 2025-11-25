package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSubnetNumber(t *testing.T) {
	tests := []struct {
		name     string
		subnet   string
		expected int
	}{
		{"valid subnet 18", "172.18.0.0/16", 18},
		{"valid subnet 19", "172.19.0.0/16", 19},
		{"valid subnet 31", "172.31.0.0/16", 31},
		{"different prefix", "192.168.0.0/24", 168},
		{"empty string", "", 0},
		{"invalid format", "invalid", 0},
		{"single part", "172", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSubnetNumber(tt.subnet)
			if result != tt.expected {
				t.Errorf("extractSubnetNumber(%q) = %d, want %d", tt.subnet, result, tt.expected)
			}
		})
	}
}

func TestGetDefaultIP(t *testing.T) {
	tests := []struct {
		name     string
		subnet   string
		expected string
	}{
		{"172.18 subnet", "172.18.0.0/16", "172.18.0.2"},
		{"172.19 subnet", "172.19.0.0/16", "172.19.0.2"},
		{"172.31 subnet", "172.31.0.0/16", "172.31.0.2"},
		{"empty string", "", ""},
		{"no CIDR", "172.18.0.0", "172.18.0.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDefaultIP(tt.subnet)
			if result != tt.expected {
				t.Errorf("GetDefaultIP(%q) = %q, want %q", tt.subnet, result, tt.expected)
			}
		})
	}
}

func TestGetContainerIP(t *testing.T) {
	tests := []struct {
		name     string
		subnet   string
		index    int
		expected string
	}{
		{"first container", "172.18.0.0/16", 0, "172.18.0.2"},
		{"second container", "172.18.0.0/16", 1, "172.18.0.3"},
		{"tenth container", "172.18.0.0/16", 8, "172.18.0.10"},
		{"different subnet", "172.19.0.0/16", 0, "172.19.0.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetContainerIP(tt.subnet, tt.index)
			if result != tt.expected {
				t.Errorf("GetContainerIP(%q, %d) = %q, want %q", tt.subnet, tt.index, result, tt.expected)
			}
		})
	}
}

func TestProjectManager_AllocateSubnet(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "bootapp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := &ProjectManager{
		globalPath: filepath.Join(tmpDir, "projects.json"),
		projects:   make(map[string]ProjectInfo),
	}

	// First allocation should get 172.18.0.0/16
	subnet1, err := mgr.allocateSubnet()
	if err != nil {
		t.Fatalf("allocateSubnet() error = %v", err)
	}
	if subnet1 != "172.18.0.0/16" {
		t.Errorf("First subnet = %q, want %q", subnet1, "172.18.0.0/16")
	}

	// Add first project
	mgr.projects["project1"] = ProjectInfo{Subnet: subnet1}

	// Second allocation should get 172.19.0.0/16
	subnet2, err := mgr.allocateSubnet()
	if err != nil {
		t.Fatalf("allocateSubnet() error = %v", err)
	}
	if subnet2 != "172.19.0.0/16" {
		t.Errorf("Second subnet = %q, want %q", subnet2, "172.19.0.0/16")
	}
}

func TestProjectManager_AllocateSubnet_SkipsUsed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootapp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := &ProjectManager{
		globalPath: filepath.Join(tmpDir, "projects.json"),
		projects: map[string]ProjectInfo{
			"existing1": {Subnet: "172.18.0.0/16"},
			"existing2": {Subnet: "172.19.0.0/16"},
			"existing3": {Subnet: "172.21.0.0/16"}, // Skip 20
		},
	}

	// Should allocate 172.20.0.0/16 (skipped one)
	subnet, err := mgr.allocateSubnet()
	if err != nil {
		t.Fatalf("allocateSubnet() error = %v", err)
	}
	if subnet != "172.20.0.0/16" {
		t.Errorf("Subnet = %q, want %q", subnet, "172.20.0.0/16")
	}
}

func TestProjectManager_AllocateSubnet_Exhausted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootapp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Fill all available subnets (18-31)
	projects := make(map[string]ProjectInfo)
	for i := subnetStart; i <= subnetEnd; i++ {
		projects[string(rune('a'+i-subnetStart))] = ProjectInfo{
			Subnet: GetContainerIP("172."+string(rune('0'+i/10))+string(rune('0'+i%10))+".0.0/16", -2),
		}
	}

	// Manually set correct subnets
	projects = make(map[string]ProjectInfo)
	for i := subnetStart; i <= subnetEnd; i++ {
		name := string(rune('a' + i - subnetStart))
		projects[name] = ProjectInfo{
			Subnet: "172." + itoa(i) + ".0.0/16",
		}
	}

	mgr := &ProjectManager{
		globalPath: filepath.Join(tmpDir, "projects.json"),
		projects:   projects,
	}

	_, err = mgr.allocateSubnet()
	if err == nil {
		t.Error("allocateSubnet() should return error when exhausted")
	}
}

// Helper function for int to string
func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

func TestProjectManager_GetOrCreateProject_New(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootapp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := &ProjectManager{
		globalPath: filepath.Join(tmpDir, "projects.json"),
		projects:   make(map[string]ProjectInfo),
	}

	info, _, err := mgr.GetOrCreateProject("myproject", "/path/to/project", []string{"myproject.local"}, nil)
	if err != nil {
		t.Fatalf("GetOrCreateProject() error = %v", err)
	}

	if info.Subnet != "172.18.0.0/16" {
		t.Errorf("Subnet = %q, want %q", info.Subnet, "172.18.0.0/16")
	}
	if info.Path != "/path/to/project" {
		t.Errorf("Path = %q, want %q", info.Path, "/path/to/project")
	}
	if len(info.Domains) != 1 || info.Domains[0] != "myproject.local" {
		t.Errorf("Domains = %v, want [%s]", info.Domains, "myproject.local")
	}

	// Verify saved to projects map
	if _, ok := mgr.projects["myproject"]; !ok {
		t.Error("Project not saved to projects map")
	}
}

func TestProjectManager_GetOrCreateProject_Existing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootapp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := &ProjectManager{
		globalPath: filepath.Join(tmpDir, "projects.json"),
		projects: map[string]ProjectInfo{
			"existing": {
				Path:   "/old/path",
				Subnet: "172.20.0.0/16",
				Domain: "existing.local",
			},
		},
	}

	// Get existing project
	info, _, err := mgr.GetOrCreateProject("existing", "/old/path", []string{"existing.local"}, nil)
	if err != nil {
		t.Fatalf("GetOrCreateProject() error = %v", err)
	}

	// Should return existing subnet, not allocate new one
	if info.Subnet != "172.20.0.0/16" {
		t.Errorf("Subnet = %q, want %q (should keep existing)", info.Subnet, "172.20.0.0/16")
	}
}

func TestProjectManager_GetOrCreateProject_UpdatePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootapp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := &ProjectManager{
		globalPath: filepath.Join(tmpDir, "projects.json"),
		projects: map[string]ProjectInfo{
			"myproject": {
				Path:   "/old/path",
				Subnet: "172.20.0.0/16",
				Domain: "myproject.local",
			},
		},
	}

	// Call with new path
	info, _, err := mgr.GetOrCreateProject("myproject", "/new/path", []string{"myproject.local"}, nil)
	if err != nil {
		t.Fatalf("GetOrCreateProject() error = %v", err)
	}

	// Should update path but keep subnet
	if info.Path != "/new/path" {
		t.Errorf("Path = %q, want %q", info.Path, "/new/path")
	}
	if info.Subnet != "172.20.0.0/16" {
		t.Errorf("Subnet = %q, want %q (should keep existing)", info.Subnet, "172.20.0.0/16")
	}
}

func TestProjectManager_RemoveProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootapp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := &ProjectManager{
		globalPath: filepath.Join(tmpDir, "projects.json"),
		projects: map[string]ProjectInfo{
			"project1": {Subnet: "172.18.0.0/16"},
			"project2": {Subnet: "172.19.0.0/16"},
		},
	}

	err = mgr.RemoveProject("project1")
	if err != nil {
		t.Fatalf("RemoveProject() error = %v", err)
	}

	if _, ok := mgr.projects["project1"]; ok {
		t.Error("project1 should be removed")
	}
	if _, ok := mgr.projects["project2"]; !ok {
		t.Error("project2 should still exist")
	}
}

func TestProjectManager_ListProjects(t *testing.T) {
	mgr := &ProjectManager{
		projects: map[string]ProjectInfo{
			"project1": {Subnet: "172.18.0.0/16", Path: "/path1"},
			"project2": {Subnet: "172.19.0.0/16", Path: "/path2"},
		},
	}

	list := mgr.ListProjects()

	if len(list) != 2 {
		t.Errorf("ListProjects() returned %d projects, want 2", len(list))
	}

	// Verify it's a copy, not the original
	list["project3"] = ProjectInfo{}
	if len(mgr.projects) != 2 {
		t.Error("ListProjects() should return a copy, not the original map")
	}
}

func TestProjectManager_GetProject(t *testing.T) {
	mgr := &ProjectManager{
		projects: map[string]ProjectInfo{
			"existing": {Subnet: "172.18.0.0/16", Path: "/path"},
		},
	}

	// Existing project
	info, ok := mgr.GetProject("existing")
	if !ok {
		t.Error("GetProject() should return true for existing project")
	}
	if info.Subnet != "172.18.0.0/16" {
		t.Errorf("Subnet = %q, want %q", info.Subnet, "172.18.0.0/16")
	}

	// Non-existing project
	_, ok = mgr.GetProject("nonexistent")
	if ok {
		t.Error("GetProject() should return false for non-existing project")
	}
}

func TestProjectManager_SaveAndLoadGlobal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootapp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	globalPath := filepath.Join(tmpDir, "subdir", "projects.json")

	// Save
	mgr1 := &ProjectManager{
		globalPath: globalPath,
		projects: map[string]ProjectInfo{
			"project1": {Path: "/path1", Subnet: "172.18.0.0/16", Domain: "p1.local"},
			"project2": {Path: "/path2", Subnet: "172.19.0.0/16", Domain: "p2.local"},
		},
	}
	err = mgr1.saveGlobal()
	if err != nil {
		t.Fatalf("saveGlobal() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load
	mgr2 := &ProjectManager{
		globalPath: globalPath,
		projects:   make(map[string]ProjectInfo),
	}
	err = mgr2.loadGlobal()
	if err != nil {
		t.Fatalf("loadGlobal() error = %v", err)
	}

	if len(mgr2.projects) != 2 {
		t.Errorf("Loaded %d projects, want 2", len(mgr2.projects))
	}

	p1, ok := mgr2.projects["project1"]
	if !ok {
		t.Fatal("project1 not loaded")
	}
	if p1.Subnet != "172.18.0.0/16" {
		t.Errorf("project1.Subnet = %q, want %q", p1.Subnet, "172.18.0.0/16")
	}
	if p1.Path != "/path1" {
		t.Errorf("project1.Path = %q, want %q", p1.Path, "/path1")
	}
}

func TestNewProjectManager(t *testing.T) {
	// This test uses actual home directory but with a unique subpath
	tmpDir, err := os.MkdirTemp("", "bootapp-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	mgr, err := NewProjectManager()
	if err != nil {
		t.Fatalf("NewProjectManager() error = %v", err)
	}

	if mgr == nil {
		t.Fatal("NewProjectManager() returned nil")
	}

	expectedPath := filepath.Join(tmpDir, ".docker-bootapp", "projects.json")
	if mgr.globalPath != expectedPath {
		t.Errorf("globalPath = %q, want %q", mgr.globalPath, expectedPath)
	}

	if mgr.projects == nil {
		t.Error("projects map should be initialized")
	}
}
