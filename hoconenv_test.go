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

	cfg := NewConfig()
	err := cfg.Load(DefaultOptions(), "basic.conf")

	assertNoError(t, err)
	assertEnvVar(t, "DATABASE_URL", "postgresql://localhost:5432/db")
	assertEnvVar(t, "DATABASE_USER", "admin")
}

func TestIncludeFile(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mainContent := `
include "sub.conf"
app.name = "main"
`
	subContent := `
app.version = "1.0"
`
	createTempConfig(t, "main.conf", mainContent)
	createTempConfig(t, "sub.conf", subContent)

	cfg := NewConfig()
	err := cfg.Load(DefaultOptions(), "main.conf")

	assertNoError(t, err)
	assertEnvVar(t, "APP_NAME", "main")
	assertEnvVar(t, "APP_VERSION", "1.0")
}

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

	cfg := NewConfig()
	err := cfg.Load(DefaultOptions(), "url.conf")

	assertNoError(t, err)
	assertEnvVar(t, "REMOTE_CONFIG", "from-url")
	assertEnvVar(t, "LOCAL_CONFIG", "local")
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

	cfg := NewConfig()
	err := cfg.Load(DefaultOptions(), "dir.conf")

	assertNoError(t, err)
	assertEnvVar(t, "A", "1")
	assertEnvVar(t, "B", "2")
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

	cfg := NewConfig()
	err := cfg.Load(DefaultOptions(), "glob.conf")

	assertNoError(t, err)
	assertEnvVar(t, "A", "1")
	assertEnvVar(t, "B", "2")
}

func TestOptionalInclude(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	content := `
include optional("nonexistent.conf")
test = "value"	
`

	createTempConfig(t, "optional.conf", content)

	cfg := NewConfig()
	err := cfg.Load(DefaultOptions(), "optional.conf")

	assertNoError(t, err)
	assertEnvVar(t, "TEST", "value")
}

func TestPrefixOption(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	content := `
test = "value"	
`
	createTempConfig(t, "prefix.conf", content)

	opts := DefaultOptions()
	opts.DefaultPrefix = "PRE"

	cfg := NewConfig()
	err := cfg.Load(opts, "prefix.conf")

	assertNoError(t, err)
	assertEnvVar(t, "PRE_TEST", "value")
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
	assertEnvVar(t, "PROD_HOST", "https://idontknow.com")
}

func TestPrefixLotOptions(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	local := `
host = "localhost"	
`

	staging := `
host = "https://stg/idontknow.com"	
`

	prod := `
host = "https://prod/idontknow.com"	
`

	createTempConfig(t, "local.conf", local)
	createTempConfig(t, "staging.conf", staging)
	createTempConfig(t, "prod.conf", prod)

	localOpts := DefaultOptions()
	localOpts.DefaultPrefix = "LOCAL"

	stgOpts := DefaultOptions()
	stgOpts.DefaultPrefix = "STG"

	prodOpts := DefaultOptions()
	prodOpts.DefaultPrefix = "PROD"

	cfg := NewConfig()

	err := cfg.Load(localOpts, "local.conf")
	assertNoError(t, err)
	assertEnvVar(t, "LOCAL_HOST", "localhost")

	err = cfg.Load(stgOpts, "staging.conf")
	assertNoError(t, err)
	assertEnvVar(t, "STG_HOST", "https://stg/idontknow.com")

	err = cfg.Load(prodOpts, "prod.conf")
	assertNoError(t, err)
	assertEnvVar(t, "PROD_HOST", "https://prod/idontknow.com")
}