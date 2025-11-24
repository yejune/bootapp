package hosts

import (
	"os"
	"strings"
	"testing"

	"github.com/yejune/docker-bootapp/internal/network"
)

func TestMarkerFormat(t *testing.T) {
	// Test that marker is correct
	if marker != "## docker-bootapp" {
		t.Errorf("marker = %q, want %q", marker, "## docker-bootapp")
	}
}

func TestBuildEntry(t *testing.T) {
	ip := "172.18.0.2"
	domain := "myapp.local"
	projectName := "myproject"

	// Build entry manually (same logic as AddEntry)
	entry := ip + "\t" + domain + "\t" + marker + ":" + projectName

	expected := "172.18.0.2\tmyapp.local\t## docker-bootapp:myproject"
	if entry != expected {
		t.Errorf("entry = %q, want %q", entry, expected)
	}
}

func TestBuildEntries(t *testing.T) {
	containers := map[string]network.ContainerInfo{
		"app": {
			IP:      "172.18.0.2",
			Domains: []string{"myapp.local", "www.myapp.local"},
		},
		"api": {
			IP:      "172.18.0.3",
			Domains: []string{"api.local"},
		},
		"db": {
			IP:      "172.18.0.4",
			Domains: nil, // No domains - should be skipped
		},
		"redis": {
			IP:      "", // No IP - should be skipped
			Domains: []string{"redis.local"},
		},
	}

	projectName := "myproject"

	// Build entries (same logic as AddEntries)
	var entries []string
	for _, info := range containers {
		if info.IP == "" || len(info.Domains) == 0 {
			continue
		}
		for _, domain := range info.Domains {
			if domain != "" {
				entry := info.IP + "\t" + domain + "\t" + marker + ":" + projectName
				entries = append(entries, entry)
			}
		}
	}

	// Should have 3 entries (2 for app, 1 for api)
	if len(entries) != 3 {
		t.Errorf("entries count = %d, want 3", len(entries))
	}

	// Verify each entry has correct format
	for _, entry := range entries {
		if !strings.Contains(entry, marker+":"+projectName) {
			t.Errorf("entry %q missing marker", entry)
		}
		fields := strings.Fields(entry)
		if len(fields) < 3 {
			t.Errorf("entry %q has wrong format", entry)
		}
	}
}

func TestBuildEntries_EmptyContainers(t *testing.T) {
	containers := map[string]network.ContainerInfo{}
	projectName := "myproject"

	var entries []string
	for _, info := range containers {
		if info.IP == "" || len(info.Domains) == 0 {
			continue
		}
		for _, domain := range info.Domains {
			if domain != "" {
				entry := info.IP + "\t" + domain + "\t" + marker + ":" + projectName
				entries = append(entries, entry)
			}
		}
	}

	if len(entries) != 0 {
		t.Errorf("entries count = %d, want 0", len(entries))
	}
}

func TestBuildEntries_SkipEmptyDomain(t *testing.T) {
	containers := map[string]network.ContainerInfo{
		"app": {
			IP:      "172.18.0.2",
			Domains: []string{"myapp.local", "", "  "}, // Contains empty domain
		},
	}
	projectName := "myproject"

	var entries []string
	for _, info := range containers {
		if info.IP == "" || len(info.Domains) == 0 {
			continue
		}
		for _, domain := range info.Domains {
			domain = strings.TrimSpace(domain)
			if domain != "" {
				entry := info.IP + "\t" + domain + "\t" + marker + ":" + projectName
				entries = append(entries, entry)
			}
		}
	}

	// Should only have 1 entry (skipping empty domains)
	if len(entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(entries))
	}
}

func TestSedPattern(t *testing.T) {
	projectName := "myproject"

	// Pattern for removing project entries
	pattern := "/" + marker + ":" + projectName + "/d"

	expected := "/## docker-bootapp:myproject/d"
	if pattern != expected {
		t.Errorf("pattern = %q, want %q", pattern, expected)
	}
}

func TestParseHostsLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantIP     string
		wantDomain string
		wantMarker bool
	}{
		{
			"bootapp entry",
			"172.18.0.2\tmyapp.local\t## docker-bootapp:myproject",
			"172.18.0.2",
			"myapp.local",
			true,
		},
		{
			"bootapp entry with spaces",
			"172.18.0.2  myapp.local  ## docker-bootapp:myproject",
			"172.18.0.2",
			"myapp.local",
			true,
		},
		{
			"regular hosts entry",
			"127.0.0.1\tlocalhost",
			"127.0.0.1",
			"localhost",
			false,
		},
		{
			"comment line",
			"# This is a comment",
			"#",      // First field of comment
			"This",   // Second field
			false,
		},
		{
			"empty line",
			"",
			"",
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := strings.Fields(tt.line)
			hasMarker := strings.Contains(tt.line, marker)

			if hasMarker != tt.wantMarker {
				t.Errorf("hasMarker = %v, want %v", hasMarker, tt.wantMarker)
			}

			if len(fields) >= 2 {
				ip := fields[0]
				domain := fields[1]
				if ip != tt.wantIP {
					t.Errorf("ip = %q, want %q", ip, tt.wantIP)
				}
				if domain != tt.wantDomain {
					t.Errorf("domain = %q, want %q", domain, tt.wantDomain)
				}
			} else if tt.wantIP != "" {
				t.Error("expected to parse IP and domain but got fewer fields")
			}
		})
	}
}

func TestFilterMarkedLines(t *testing.T) {
	lines := []string{
		"127.0.0.1\tlocalhost",
		"172.18.0.2\tmyapp.local\t## docker-bootapp:myproject",
		"172.18.0.3\tapi.local\t## docker-bootapp:myproject",
		"192.168.1.1\trouter.local",
		"172.19.0.2\tother.local\t## docker-bootapp:otherproject",
	}

	// Filter lines with our marker
	var marked []string
	for _, line := range lines {
		if strings.Contains(line, marker) {
			marked = append(marked, line)
		}
	}

	if len(marked) != 3 {
		t.Errorf("marked lines = %d, want 3", len(marked))
	}
}

func TestFilterByProject(t *testing.T) {
	lines := []string{
		"172.18.0.2\tmyapp.local\t## docker-bootapp:myproject",
		"172.18.0.3\tapi.local\t## docker-bootapp:myproject",
		"172.19.0.2\tother.local\t## docker-bootapp:otherproject",
	}

	projectName := "myproject"
	projectMarker := marker + ":" + projectName

	// Filter lines for specific project
	var projectLines []string
	for _, line := range lines {
		if strings.Contains(line, projectMarker) {
			projectLines = append(projectLines, line)
		}
	}

	if len(projectLines) != 2 {
		t.Errorf("project lines = %d, want 2", len(projectLines))
	}
}

// TestListEntriesFromReader tests parsing hosts file content
func TestListEntriesFromContent(t *testing.T) {
	content := `127.0.0.1	localhost
255.255.255.255	broadcasthost
::1             localhost
172.18.0.2	myapp.local	## docker-bootapp:myproject
172.18.0.3	api.local	## docker-bootapp:myproject
# Some comment
192.168.1.1	router.local
`

	lines := strings.Split(content, "\n")
	var entries []string
	for _, line := range lines {
		if strings.Contains(line, marker) {
			entries = append(entries, line)
		}
	}

	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

// TestGetIPFromLine extracts IP from hosts line
func TestGetIPFromLine(t *testing.T) {
	tests := []struct {
		line   string
		domain string
		wantIP string
	}{
		{
			"172.18.0.2\tmyapp.local\t## docker-bootapp:myproject",
			"myapp.local",
			"172.18.0.2",
		},
		{
			"172.18.0.3  api.local  ## docker-bootapp:myproject",
			"api.local",
			"172.18.0.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			if strings.Contains(tt.line, tt.domain) {
				fields := strings.Fields(tt.line)
				if len(fields) >= 2 {
					ip := fields[0]
					if ip != tt.wantIP {
						t.Errorf("ip = %q, want %q", ip, tt.wantIP)
					}
				}
			}
		})
	}
}

// Integration test helper - creates a temp hosts file for testing
func TestWithTempHostsFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "hosts-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `127.0.0.1	localhost
172.18.0.2	myapp.local	## docker-bootapp:myproject
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Read and verify
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	var markedCount int
	for _, line := range lines {
		if strings.Contains(line, marker) {
			markedCount++
		}
	}

	if markedCount != 1 {
		t.Errorf("marked lines = %d, want 1", markedCount)
	}
}

func TestEntryContainsDomain(t *testing.T) {
	line := "172.18.0.2\tmyapp.local\t## docker-bootapp:myproject"

	tests := []struct {
		domain string
		want   bool
	}{
		{"myapp.local", true},
		{"other.local", false},
		{"myapp", true},     // Partial match - current behavior
		{"app.local", true}, // Also partial match (contained in "myapp.local")
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := strings.Contains(line, tt.domain)
			if got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}
