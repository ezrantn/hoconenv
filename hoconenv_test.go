package hoconenv

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Unsetenv("APP_NAME")
	os.Unsetenv("APP_DATABASE_HOST")
	os.Unsetenv("APP_DATABASE_PASSWORD")
	os.Exit(code)
}

func TestLoadValidHocon(t *testing.T) {
	hoconContent := `
app {
	name = MyApp
	version = "1.0"
	database {
		host = localhost
		port = 5432
		user = admin
		password = "secret"
	}
}`
	fileName := "test_config.conf"
	err := os.WriteFile(fileName, []byte(hoconContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	defer os.Remove(fileName)

	// Load the HOCON file
	err = Load(fileName)
	if err != nil {
		t.Fatalf("Failed to load HOCON: %v", err)
	}

	// Check environment variables
	tests := map[string]string{
		"APP_NAME":              "MyApp",
		"APP_VERSION":           "1.0",
		"APP_DATABASE_HOST":     "localhost",
		"APP_DATABASE_PORT":     "5432",
		"APP_DATABASE_USER":     "admin",
		"APP_DATABASE_PASSWORD": "secret",
	}

	for key, expected := range tests {
		value := os.Getenv(key)
		if value != expected {
			t.Errorf("Expected %s to be %s, got %s", key, expected, value)
		}
	}
}

func TestLoadEmptyFile(t *testing.T) {
	fileName := "empty_config.conf"
	err := os.WriteFile(fileName, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	defer os.Remove(fileName)

	initialEnv := os.Environ()

	// Load the empty file
	err = Load(fileName)
	if err != nil {
		t.Fatalf("Failed to load empty HOCON file: %v", err)
	}

	finalEnv := os.Environ()
	// Ensure no unexpected environment variables are set
	if len(finalEnv) != len(initialEnv) {
		t.Errorf("Expected no environment variables to be set for an empty file, got %v", finalEnv)
	}
}

func TestLoadInvalidSyntax(t *testing.T) {
	// Create a HOCON file with invalid syntax
	hoconContent := `
app {
	name = MyApp
	database {
    	host: localhost
    	port = 5432
    }
}`
	fileName := "invalid_config.conf"
	err := os.WriteFile(fileName, []byte(hoconContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(fileName)

	// Attempt to load the invalid file
	err = Load(fileName)
	if err == nil {
		t.Error("Expected an error for invalid HOCON syntax, got nil")
	}
}

func TestLoadCommentsAndEmptyLines(t *testing.T) {
	hoconContent := `
# This is a comment
app {
    name = MyApp

    // Another comment
    database {
        host = localhost # Inline comment
        port = 5432
    }
}`
	fileName := "commented_config.conf"
	err := os.WriteFile(fileName, []byte(hoconContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	defer os.Remove(fileName)

	// Load the file
	err = Load(fileName)
	if err != nil {
		t.Fatalf("Failed to load HOCON with comments: %v", err)
	}

	// Check environment variables
	tests := map[string]string{
		"APP_NAME":          "MyApp",
		"APP_DATABASE_HOST": "localhost",
		"APP_DATABASE_PORT": "5432",
	}

	for key, expected := range tests {
		value := os.Getenv(key)
		if value != expected {
			t.Errorf("Expected %s to be %s, got %s", key, expected, value)
		}
	}
}

func TestSetPrefix(t *testing.T) {
	hoconContent := `
app {
	name = MyApp
	version = "1.0"
	database {
		host = localhost
		port = 5432
		user = admin
		password = "secret"
	}
}`
	fileName := "test_set_prefix.conf"
	err := os.WriteFile(fileName, []byte(hoconContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	defer os.Remove(fileName)

	// Set a prefix "TEST"
	SetPrefix("TEST")

	// Load the HOCON file
	err = Load(fileName)
	if err != nil {
		t.Fatalf("Failed to load HOCON: %v", err)
	}

	// Check environment variables with prefix "TEST"
	tests := map[string]string{
		"TEST_APP_NAME":              "MyApp",
		"TEST_APP_VERSION":           "1.0",
		"TEST_APP_DATABASE_HOST":     "localhost",
		"TEST_APP_DATABASE_PORT":     "5432",
		"TEST_APP_DATABASE_USER":     "admin",
		"TEST_APP_DATABASE_PASSWORD": "secret",
	}

	for key, expected := range tests {
		value := os.Getenv(key)
		if value != expected {
			t.Errorf("Expected %s to be %s, got %s", key, expected, value)
		}
	}
}
