package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetProjectName(t *testing.T) {
	tests := []struct {
		name        string
		composePath string
		expected    string
	}{
		{"simple path", "/home/user/myproject/docker-compose.yml", "myproject"},
		{"nested path", "/var/www/apps/webapp/docker-compose.yml", "webapp"},
		{"current dir", "./docker-compose.yml", "."},
		{"compose.yaml", "/projects/api/compose.yaml", "api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectName(tt.composePath)
			if result != tt.expected {
				t.Errorf("GetProjectName(%q) = %q, want %q", tt.composePath, result, tt.expected)
			}
		})
	}
}

func TestSplitDomains(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single domain", "myapp.local", []string{"myapp.local"}},
		{"comma separated", "a.com,b.com,c.com", []string{"a.com", "b.com", "c.com"}},
		{"comma with spaces", "a.com, b.com, c.com", []string{"a.com", "b.com", "c.com"}},
		{"newline separated", "a.com\nb.com\nc.com", []string{"a.com", "b.com", "c.com"}},
		{"mixed separators", "a.com, b.com\nc.com", []string{"a.com", "b.com", "c.com"}},
		{"empty string", "", nil},
		{"only whitespace", "   ", nil},
		{"with extra spaces", "  a.com   b.com  ", []string{"a.com", "b.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitDomains(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitDomains(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, d := range result {
				if d != tt.expected[i] {
					t.Errorf("splitDomains(%q)[%d] = %q, want %q", tt.input, i, d, tt.expected[i])
				}
			}
		})
	}
}

func TestUniqueDomains(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"no duplicates", []string{"a.com", "b.com"}, []string{"a.com", "b.com"}},
		{"with duplicates", []string{"a.com", "b.com", "a.com"}, []string{"a.com", "b.com"}},
		{"all same", []string{"a.com", "a.com", "a.com"}, []string{"a.com"}},
		{"empty", []string{}, nil},
		{"preserves order", []string{"c.com", "a.com", "b.com", "a.com"}, []string{"c.com", "a.com", "b.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueDomains(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("uniqueDomains(%v) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, d := range result {
				if d != tt.expected[i] {
					t.Errorf("uniqueDomains(%v)[%d] = %q, want %q", tt.input, i, d, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractTraefikHosts(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		expected []string
	}{
		{
			"single host",
			"Host(`myapp.local`)",
			[]string{"myapp.local"},
		},
		{
			"multiple hosts with OR",
			"Host(`a.com`) || Host(`b.com`)",
			[]string{"a.com", "b.com"},
		},
		{
			"hosts in same Host()",
			"Host(`a.com`, `b.com`)",
			[]string{"a.com", "b.com"},
		},
		{
			"complex rule",
			"Host(`api.example.com`) && PathPrefix(`/v1`)",
			[]string{"api.example.com"},
		},
		{
			"no hosts",
			"PathPrefix(`/api`)",
			nil,
		},
		{
			"empty",
			"",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTraefikHosts(tt.rule)
			if len(result) != len(tt.expected) {
				t.Errorf("extractTraefikHosts(%q) = %v, want %v", tt.rule, result, tt.expected)
				return
			}
			for i, h := range result {
				if h != tt.expected[i] {
					t.Errorf("extractTraefikHosts(%q)[%d] = %q, want %q", tt.rule, i, h, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractDomainsFromEnvironment_List(t *testing.T) {
	// Test list format (- VAR=value)
	env := []interface{}{
		"SOME_VAR=foo",
		"DOMAIN=myapp.local",
		"OTHER=bar",
	}

	result := extractDomainsFromEnvironment(env)
	if len(result) != 1 || result[0] != "myapp.local" {
		t.Errorf("extractDomainsFromEnvironment() = %v, want [myapp.local]", result)
	}
}

func TestExtractDomainsFromEnvironment_Map(t *testing.T) {
	// Test map format (VAR: value)
	env := map[string]interface{}{
		"SOME_VAR": "foo",
		"DOMAIN":   "myapp.local",
		"OTHER":    "bar",
	}

	result := extractDomainsFromEnvironment(env)
	if len(result) != 1 || result[0] != "myapp.local" {
		t.Errorf("extractDomainsFromEnvironment() = %v, want [myapp.local]", result)
	}
}

func TestExtractDomainsFromEnvironment_MultipleDomains(t *testing.T) {
	env := map[string]interface{}{
		"SSL_DOMAINS": "api.local,www.local",
	}

	result := extractDomainsFromEnvironment(env)
	if len(result) != 2 {
		t.Errorf("extractDomainsFromEnvironment() = %v, want 2 domains", result)
		return
	}
	if result[0] != "api.local" || result[1] != "www.local" {
		t.Errorf("extractDomainsFromEnvironment() = %v, want [api.local, www.local]", result)
	}
}

func TestExtractDomainsFromEnvironment_AllKeys(t *testing.T) {
	tests := []struct {
		key    string
		value  string
		domain string
	}{
		{"DOMAIN", "a.local", "a.local"},
		{"DOMAINS", "b.local", "b.local"},
		{"SSL_DOMAINS", "c.local", "c.local"},
		{"APP_DOMAIN", "d.local", "d.local"},
		{"VIRTUAL_HOST", "e.local", "e.local"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			env := map[string]interface{}{
				tt.key: tt.value,
			}
			result := extractDomainsFromEnvironment(env)
			if len(result) != 1 || result[0] != tt.domain {
				t.Errorf("extractDomainsFromEnvironment() with %s = %v, want [%s]", tt.key, result, tt.domain)
			}
		})
	}
}

func TestExtractDomainsFromLabels_Map(t *testing.T) {
	labels := map[string]interface{}{
		"traefik.http.routers.myapp.rule": "Host(`myapp.local`)",
		"traefik.http.routers.myapp.tls":  "true",
	}

	result := extractDomainsFromLabels(labels)
	if len(result) != 1 || result[0] != "myapp.local" {
		t.Errorf("extractDomainsFromLabels() = %v, want [myapp.local]", result)
	}
}

func TestExtractDomainsFromLabels_List(t *testing.T) {
	labels := []interface{}{
		"traefik.http.routers.myapp.rule=Host(`myapp.local`)",
		"traefik.http.routers.myapp.tls=true",
	}

	result := extractDomainsFromLabels(labels)
	if len(result) != 1 || result[0] != "myapp.local" {
		t.Errorf("extractDomainsFromLabels() = %v, want [myapp.local]", result)
	}
}

func TestParseComposeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	content := `version: "3.8"
services:
  app:
    image: nginx
    environment:
      DOMAIN: myapp.local
  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: secret
`
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	compose, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile() error = %v", err)
	}

	if compose.Version != "3.8" {
		t.Errorf("Version = %q, want %q", compose.Version, "3.8")
	}

	if len(compose.Services) != 2 {
		t.Errorf("Services count = %d, want 2", len(compose.Services))
	}

	if _, ok := compose.Services["app"]; !ok {
		t.Error("app service not found")
	}
	if _, ok := compose.Services["db"]; !ok {
		t.Error("db service not found")
	}
}

func TestParseComposeFile_NotFound(t *testing.T) {
	_, err := ParseComposeFile("/nonexistent/path/docker-compose.yml")
	if err == nil {
		t.Error("ParseComposeFile() should return error for non-existent file")
	}
}

func TestParseComposeFile_InvalidYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	content := `
this is not: valid: yaml
  - wrong indentation
`
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	_, err = ParseComposeFile(composePath)
	if err == nil {
		t.Error("ParseComposeFile() should return error for invalid YAML")
	}
}

func TestExtractDomains(t *testing.T) {
	compose := &ComposeFile{
		Services: map[string]Service{
			"app": {
				Environment: map[string]interface{}{
					"DOMAIN": "myapp.local",
				},
			},
			"api": {
				Environment: map[string]interface{}{
					"SSL_DOMAINS": "api.local,api2.local",
				},
			},
			"db": {
				Environment: map[string]interface{}{
					"MYSQL_PASSWORD": "secret",
				},
			},
		},
	}

	domains := ExtractDomains(compose)

	// Should contain 3 domains (myapp.local, api.local, api2.local)
	if len(domains) != 3 {
		t.Errorf("ExtractDomains() = %v, want 3 domains", domains)
	}

	// Check all domains are present
	domainMap := make(map[string]bool)
	for _, d := range domains {
		domainMap[d] = true
	}

	expected := []string{"myapp.local", "api.local", "api2.local"}
	for _, e := range expected {
		if !domainMap[e] {
			t.Errorf("ExtractDomains() missing %q", e)
		}
	}
}

func TestExtractDomain(t *testing.T) {
	compose := &ComposeFile{
		Services: map[string]Service{
			"app": {
				Environment: map[string]interface{}{
					"DOMAIN": "first.local",
				},
			},
		},
	}

	domain := ExtractDomain(compose)
	if domain != "first.local" {
		t.Errorf("ExtractDomain() = %q, want %q", domain, "first.local")
	}
}

func TestExtractDomain_Empty(t *testing.T) {
	compose := &ComposeFile{
		Services: map[string]Service{
			"db": {
				Environment: map[string]interface{}{
					"MYSQL_PASSWORD": "secret",
				},
			},
		},
	}

	domain := ExtractDomain(compose)
	if domain != "" {
		t.Errorf("ExtractDomain() = %q, want empty string", domain)
	}
}

func TestExtractServiceDomains(t *testing.T) {
	compose := &ComposeFile{
		Services: map[string]Service{
			"app": {
				Environment: map[string]interface{}{
					"DOMAIN": "myapp.local",
				},
			},
			"api": {
				Environment: map[string]interface{}{
					"SSL_DOMAINS": "api.local,api2.local",
				},
			},
			"db": {
				Environment: map[string]interface{}{
					"MYSQL_PASSWORD": "secret",
				},
			},
			"redis": {
				// No environment
			},
		},
	}

	result := ExtractServiceDomains(compose)

	// Should only have app and api (services with domain config)
	if len(result) != 2 {
		t.Errorf("ExtractServiceDomains() returned %d services, want 2", len(result))
	}

	// Check app
	if domains, ok := result["app"]; !ok {
		t.Error("app service not in result")
	} else if len(domains) != 1 || domains[0] != "myapp.local" {
		t.Errorf("app domains = %v, want [myapp.local]", domains)
	}

	// Check api
	if domains, ok := result["api"]; !ok {
		t.Error("api service not in result")
	} else if len(domains) != 2 {
		t.Errorf("api domains = %v, want 2 domains", domains)
	}

	// db and redis should not be in result
	if _, ok := result["db"]; ok {
		t.Error("db should not be in result (no domain config)")
	}
	if _, ok := result["redis"]; ok {
		t.Error("redis should not be in result (no domain config)")
	}
}

func TestFindComposeFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	// Create multiple compose files
	files := []string{"docker-compose.yml", "docker-compose.local.yml", "docker-compose.prod.yml"}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("version: '3'"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	found, err := FindComposeFiles()
	if err != nil {
		t.Fatalf("FindComposeFiles() error = %v", err)
	}

	if len(found) != 3 {
		t.Errorf("FindComposeFiles() found %d files, want 3", len(found))
	}
}

func TestFindComposeFile_MultipleError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	// Create multiple compose files
	os.WriteFile("docker-compose.yml", []byte("version: '3'"), 0644)
	os.WriteFile("docker-compose.local.yml", []byte("version: '3'"), 0644)

	_, err = FindComposeFile()
	if err == nil {
		t.Error("FindComposeFile() should return error for multiple files")
		return
	}

	multiErr, ok := err.(*MultipleFilesError)
	if !ok {
		t.Errorf("Expected MultipleFilesError, got %T", err)
		return
	}

	if len(multiErr.Files) != 2 {
		t.Errorf("MultipleFilesError.Files has %d files, want 2", len(multiErr.Files))
	}
}

func TestFindComposeFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	// Test: no compose file
	_, err = FindComposeFile()
	if err == nil {
		t.Error("FindComposeFile() should return error when no compose file exists")
	}

	// Test: docker-compose.yml exists
	if err := os.WriteFile("docker-compose.yml", []byte("version: '3'"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	path, err := FindComposeFile()
	if err != nil {
		t.Fatalf("FindComposeFile() error = %v", err)
	}
	if filepath.Base(path) != "docker-compose.yml" {
		t.Errorf("FindComposeFile() = %q, want docker-compose.yml", path)
	}

	// Test: compose.yaml takes precedence if docker-compose.yml is removed
	os.Remove("docker-compose.yml")
	if err := os.WriteFile("compose.yaml", []byte("version: '3'"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	path, err = FindComposeFile()
	if err != nil {
		t.Fatalf("FindComposeFile() error = %v", err)
	}
	if filepath.Base(path) != "compose.yaml" {
		t.Errorf("FindComposeFile() = %q, want compose.yaml", path)
	}
}

func TestExtractServiceDomains_WithTraefik(t *testing.T) {
	compose := &ComposeFile{
		Services: map[string]Service{
			"web": {
				Labels: map[string]interface{}{
					"traefik.http.routers.web.rule": "Host(`web.example.com`)",
				},
			},
			"api": {
				Labels: map[string]interface{}{
					"traefik.http.routers.api.rule": "Host(`api.example.com`) || Host(`api2.example.com`)",
				},
			},
		},
	}

	result := ExtractServiceDomains(compose)

	if len(result) != 2 {
		t.Errorf("ExtractServiceDomains() returned %d services, want 2", len(result))
	}

	// Check web
	if domains, ok := result["web"]; !ok {
		t.Error("web service not in result")
	} else if len(domains) != 1 || domains[0] != "web.example.com" {
		t.Errorf("web domains = %v, want [web.example.com]", domains)
	}

	// Check api
	if domains, ok := result["api"]; !ok {
		t.Error("api service not in result")
	} else if len(domains) != 2 {
		t.Errorf("api domains = %v, want 2 domains", domains)
	}
}

func TestExtractServiceDomains_MixedEnvAndLabels(t *testing.T) {
	compose := &ComposeFile{
		Services: map[string]Service{
			"app": {
				Environment: map[string]interface{}{
					"DOMAIN": "env.local",
				},
				Labels: map[string]interface{}{
					"traefik.http.routers.app.rule": "Host(`traefik.local`)",
				},
			},
		},
	}

	result := ExtractServiceDomains(compose)

	if domains, ok := result["app"]; !ok {
		t.Error("app service not in result")
	} else if len(domains) != 2 {
		t.Errorf("app domains = %v, want 2 domains (from env and labels)", domains)
	}
}
