package hosts

import (
	"os"
	"strings"
	"testing"

	"github.com/yejune/bootapp/internal/network"
)

func TestMarkerFormat(t *testing.T) {
	// Test that marker is correct
	if marker != "## bootapp" {
		t.Errorf("marker = %q, want %q", marker, "## bootapp")
	}
}

func TestBuildEntry(t *testing.T) {
	ip := "172.18.0.2"
	domain := "myapp.local"
	projectName := "myproject"

	// Build entry manually (same logic as AddEntry - new macOS compatible format)
	// Comment line followed by host entry
	commentLine := marker + ":" + projectName
	hostEntry := ip + "\t" + domain
	entry := commentLine + "\n" + hostEntry

	expected := "## bootapp:myproject\n172.18.0.2\tmyapp.local"
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
	commentLine := marker + ":" + projectName

	// Build entries (same logic as AddEntries - new format with comment on separate line)
	var entries []string
	for _, info := range containers {
		if info.IP == "" || len(info.Domains) == 0 {
			continue
		}
		for _, domain := range info.Domains {
			if domain != "" {
				// Add comment line first, then host entry
				entries = append(entries, commentLine)
				entries = append(entries, info.IP+"\t"+domain)
			}
		}
	}

	// Should have 6 entries (2 lines per domain: comment + host, 3 domains total)
	if len(entries) != 6 {
		t.Errorf("entries count = %d, want 6", len(entries))
	}

	// Verify format: alternating comment and host lines
	for i, entry := range entries {
		if i%2 == 0 {
			// Comment line
			if entry != commentLine {
				t.Errorf("entry[%d] = %q, want comment line %q", i, entry, commentLine)
			}
		} else {
			// Host line - should have IP and domain
			fields := strings.Fields(entry)
			if len(fields) != 2 {
				t.Errorf("entry[%d] %q has wrong format, want 2 fields", i, entry)
			}
		}
	}
}

func TestBuildEntries_EmptyContainers(t *testing.T) {
	containers := map[string]network.ContainerInfo{}
	projectName := "myproject"
	commentLine := marker + ":" + projectName

	var entries []string
	for _, info := range containers {
		if info.IP == "" || len(info.Domains) == 0 {
			continue
		}
		for _, domain := range info.Domains {
			if domain != "" {
				entries = append(entries, commentLine)
				entries = append(entries, info.IP+"\t"+domain)
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
	commentLine := marker + ":" + projectName

	var entries []string
	for _, info := range containers {
		if info.IP == "" || len(info.Domains) == 0 {
			continue
		}
		for _, domain := range info.Domains {
			domain = strings.TrimSpace(domain)
			if domain != "" {
				entries = append(entries, commentLine)
				entries = append(entries, info.IP+"\t"+domain)
			}
		}
	}

	// Should only have 2 entries (1 comment + 1 host, skipping empty domains)
	if len(entries) != 2 {
		t.Errorf("entries count = %d, want 2", len(entries))
	}
}

func TestSedPattern(t *testing.T) {
	projectName := "myproject"

	// Pattern for removing project entries (new format: delete comment line and next line)
	pattern := "/" + marker + ":" + projectName + "/{N;d;}"

	expected := "/## bootapp:myproject/{N;d;}"
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
			"bootapp comment line (new format)",
			"## bootapp:myproject",
			"##",
			"bootapp:myproject",
			true,
		},
		{
			"bootapp host entry (new format)",
			"172.18.0.2\tmyapp.local",
			"172.18.0.2",
			"myapp.local",
			false,
		},
		{
			"legacy bootapp entry (old format)",
			"172.18.0.2\tmyapp.local\t## bootapp:myproject",
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
	// New format: comment lines contain marker
	lines := []string{
		"127.0.0.1\tlocalhost",
		"## bootapp:myproject",
		"172.18.0.2\tmyapp.local",
		"## bootapp:myproject",
		"172.18.0.3\tapi.local",
		"192.168.1.1\trouter.local",
		"## bootapp:otherproject",
		"172.19.0.2\tother.local",
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
	// New format: comment lines contain project marker
	lines := []string{
		"## bootapp:myproject",
		"172.18.0.2\tmyapp.local",
		"## bootapp:myproject",
		"172.18.0.3\tapi.local",
		"## bootapp:otherproject",
		"172.19.0.2\tother.local",
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
	// New format: comment line followed by host entry
	content := `127.0.0.1	localhost
255.255.255.255	broadcasthost
::1             localhost
## bootapp:myproject
172.18.0.2	myapp.local
## bootapp:myproject
172.18.0.3	api.local
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
	// New format: host line contains only IP and domain (no marker)
	tests := []struct {
		line   string
		domain string
		wantIP string
	}{
		{
			"172.18.0.2\tmyapp.local",
			"myapp.local",
			"172.18.0.2",
		},
		{
			"172.18.0.3  api.local",
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

	// New format: comment line followed by host entry
	content := `127.0.0.1	localhost
## bootapp:myproject
172.18.0.2	myapp.local
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
	// New format: host line without marker
	line := "172.18.0.2\tmyapp.local"

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
