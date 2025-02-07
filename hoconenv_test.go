package hoconenv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Helper functions
func setupTestEnv(t *testing.T) func() {
	tempDir, err := os.MkdirTemp("", "hoconenv-test")
	if err != nil {
		t.Fatal(err)
	}
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir(tempDir)

	return func() {
		os.Chdir(originalWd)
		os.RemoveAll(tempDir)
	}
}

func createTempConfig(t *testing.T, name, content string) {
	dir := filepath.Dir(name)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}
	err := os.WriteFile(name, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertEnvVar(t *testing.T, key, expected string) {
	t.Helper()
	if got := os.Getenv(key); got != expected {
		t.Errorf("env var %s = %s; want %s", key, got, expected)
	}
}

func TestDefaultFile(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	content := `
database {
	url = "postgresql://localhost:5432/db"
	user = "admin"
}	
`

	createTempConfig(t, "application.conf", content)
	err := Load()

	assertNoError(t, err)
	assertEnvVar(t, "database.url", "postgresql://localhost:5432/db")
	assertEnvVar(t, "database.user", "admin")
}

func TestBasicLoading(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	content := `
database {
	url = "postgresql://localhost:5432/db"
	user = "admin"
}
`
	createTempConfig(t, "basic.conf", content)
	err := Load("basic.conf")

	assertNoError(t, err)
	assertEnvVar(t, "database.url", "postgresql://localhost:5432/db")
	assertEnvVar(t, "database.user", "admin")
}

func TestIncludeFile(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mainContent := `
include required "sub.conf"
app.name = "main"
`
	subContent := `
app.version = "1.0"
`
	createTempConfig(t, "main.conf", mainContent)
	createTempConfig(t, "sub.conf", subContent)

	err := Load("main.conf")

	assertNoError(t, err)
	assertEnvVar(t, "app.name", "main")
	assertEnvVar(t, "app.version", "1.0")
}

// func TestIncludeFileDoesNotExist(t *testing.T) {
// 	cleanup := setupTestEnv(t)
// 	defer cleanup()

// 	mainContent := `
// include required "missing.conf"
// app.name = "main"
// `

// 	createTempConfig(t, "main.conf", mainContent)

// 	err := Load("main.conf")

// 	if err == nil {
// 		t.Fatal("expected an error for missing required file, but got nil")
// 	}

// 	expectedErr := "failed to include required file"
// 	if !strings.Contains(err.Error(), expectedErr) {
// 		t.Errorf("expected error to contain %q, got %q", expectedErr, err.Error())
// 	}
// }

func TestIncludeURL(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`remote.config = "from-url"`))
	}))

	defer server.Close()

	content := `
include url("` + server.URL + `")
local.config = "local"	
`

	createTempConfig(t, "url.conf", content)

	err := Load("url.conf")

	assertNoError(t, err)
	assertEnvVar(t, "remote.config", "from-url")
	assertEnvVar(t, "local.config", "local")
}

func TestIncludeDirectory(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	os.Mkdir("configs", 0755)
	createTempConfig(t, "configs/1.conf", "a = 1")
	createTempConfig(t, "configs/2.conf", "b = 2")

	content := `
include directory("configs")
`

	createTempConfig(t, "dir.conf", content)

	err := Load("dir.conf")

	assertNoError(t, err)
	assertEnvVar(t, "a", "1")
	assertEnvVar(t, "b", "2")
}

func TestGlobInclude(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	createTempConfig(t, "conf1.conf", "a = 1")
	createTempConfig(t, "conf2.conf", "b = 2")

	content := `
include "conf*.conf"	
`
	createTempConfig(t, "glob.conf", content)

	err := Load("glob.conf")

	assertNoError(t, err)
	assertEnvVar(t, "a", "1")
	assertEnvVar(t, "b", "2")
}

func TestOptionalInclude(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	content := `
include optional ("nonexistent.conf")
test = "value"	
`

	createTempConfig(t, "optional.conf", content)

	err := Load("optional.conf")

	assertNoError(t, err)
	assertEnvVar(t, "test", "value")
}

func TestPrefixGlobal(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Global prefix
	SetPrefix("PROD")

	content := `
host = "https://idontknow.com"	
`

	createTempConfig(t, "global_prefix.conf", content)

	err := Load("global_prefix.conf")

	assertNoError(t, err)
	assertEnvVar(t, "prod.host", "https://idontknow.com")
}

func TestDefaultValueWithPrefix(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	content := `
host = "localhost"
`

	createTempConfig(t, "default_value_prefix", content)

	SetPrefix("test")

	err := Load("default_value_prefix")
	assertNoError(t, err)

	// Test existing key
	value := GetDefaultValue("test.host", "https://idontknow.com")
	if value != "localhost" {
		t.Errorf("Expected 'localhost', got '%s'", value)
	}

	// Test non-existing key
	value = GetDefaultValue("test.port", "5432")
	if value != "5432" {
		t.Errorf("Expected '5432', got '%s'", value)
	}
}

func TestDefaultValue(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	content := `
database {
	url = "postgresql://localhost:5432/db"
}
`

	createTempConfig(t, "default_value", content)

	err := Load("default_value")
	assertNoError(t, err)

	// Test existing key
	value := GetDefaultValue("database.url", "mysql://localhost:3306/db")
	if value != "postgresql://localhost:5432/db" {
		t.Errorf("Expected 'postgresql://localhost:5432/db', got '%s'", value)
	}

	// Test non-existing key
	value = GetDefaultValue("database.port", "5432")
	if value != "5432" {
		t.Errorf("Expected '5432', got '%s'", value)
	}
}
