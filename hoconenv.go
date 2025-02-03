package hoconenv

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	variables   = make(map[string]string)
	loadedFiles = make(map[string]bool)
	mutex       sync.RWMutex
	prefix      = ""
)

// SetPrefix configures the global prefix for environment variables
func SetPrefix(p string) {
	mutex.Lock()
	defer mutex.Unlock()
	prefix = strings.ToLower(strings.TrimSpace(p)) + "."
}

// Load loads configuration from specified files or default application.* files
func Load(files ...string) error {
	// If no fileName is passed, search for default files
	if len(files) == 0 {
		matches, err := filepath.Glob("application.*")
		if err == nil && len(matches) > 0 {
			for _, match := range matches {
				if err := loadFile(match); err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("no default configuration files found")
	}

	// Load all specified files
	for _, file := range files {
		if err := loadFile(file); err != nil {
			return err
		}
	}

	return nil
}

// GetDefaultValue retrieves the environment variable by key
func GetDefaultValue(key, defaultValue string) string {
	mutex.RLock()
	defer mutex.RUnlock()

	// Only add the prefix if the key doesn't already contain the prefix
	envKey := key
	if !strings.HasPrefix(key, prefix) {
		envKey = prefix + key
	}

	if value, exists := variables[envKey]; exists && value != "" {
		return value
	}

	return defaultValue
}

// loadFile handles the actual file loading logic
func loadFile(filePath string) error {
    mutex.Lock()
    if loadedFiles[filePath] {
        mutex.Unlock()
        return nil // Skip already loaded files
    }
    loadedFiles[filePath] = true
    mutex.Unlock()

    file, err := os.Open(filePath)
    if err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("file does not exist: %s", filePath)
        }

        return fmt.Errorf("failed to open config file %s: %w", filePath, err)
    }

    defer file.Close()

    scanner := bufio.NewScanner(file)
    var keyStack []string
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        line := strings.TrimSpace(scanner.Text())

        // Skip comments and empty lines
        if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
            continue
        }

        if err := parseLine(line, &keyStack, filePath, lineNum); err != nil {
            return err
        }
    }

    if err := scanner.Err(); err != nil {
        return fmt.Errorf("error reading file %s: %w", filePath, err)
    }

    // Apply variables to environment
    return applyVariables()
}

// parseLine handles parsing of individual HOCON lines
func parseLine(line string, keyStack *[]string, filePath string, lineNum int) error {
	if strings.HasPrefix(line, "include ") {
		return handleInclude(line, filePath)
	}

	// Handle nested blocks
	if strings.HasSuffix(line, "{") {
		key := strings.TrimSpace(strings.TrimSuffix(line, "{"))
		*keyStack = append(*keyStack, key)
		return nil
	}

	if line == "}" {
		if len(*keyStack) > 0 {
			*keyStack = (*keyStack)[:len(*keyStack)-1]
		}
		return nil
	}

	// Parse key-value pairs
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid syntax at %s:%d: %s", filePath, lineNum, line)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Handle includes
	if strings.HasPrefix(value, "include") {
		return handleInclude(value, filePath)
	}

	// Process the value
	value = processValue(value)

	// Build the full key
	fullKey := buildFullKey(*keyStack, key)

	// Store the variable
	mutex.Lock()
	variables[fullKey] = value
	mutex.Unlock()

	return nil
}

// processValue handles value processing including quote removal and comment stripping
func processValue(value string) string {
	// Remove quotes
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		value = value[1 : len(value)-1]
	}

	// Remove inline comments
	if idx := strings.Index(value, "#"); idx != -1 {
		value = value[:idx]
	}

	return strings.TrimSpace(value)
}

// buildFullKey constructs the full key path
func buildFullKey(keyStack []string, key string) string {
	if len(keyStack) > 0 {
		return strings.Join(keyStack, ".") + "." + key
	}
	return key
}

// handleInclude processes include directives
func handleInclude(value string, currentFile string) error {
	// Remove "include" keyword and trim spaces
	includeStr := strings.TrimSpace(strings.TrimPrefix(value, "include"))

	// Parse include type and path
	isRequired := true

	if strings.HasPrefix(includeStr, "optional ") {
		isRequired = false
		includeStr = strings.TrimSpace(strings.TrimPrefix(includeStr, "optional"))
	} else if strings.HasPrefix(includeStr, "required ") {
		isRequired = true
		includeStr = strings.TrimSpace(strings.TrimPrefix(includeStr, "required"))
	}

	// Handle quoted strings
	includeStr = strings.Trim(includeStr, "\"'")

	// Handle different include patterns
	switch {
	case strings.HasPrefix(includeStr, "url("):
		// URL includes
		urlStr := strings.TrimPrefix(includeStr, "url(")
		urlStr = strings.TrimSuffix(urlStr, ")")
		urlStr = strings.Trim(urlStr, "\"'")
		return handleURLInclude(urlStr, isRequired)

	case strings.HasPrefix(includeStr, "directory("):
		// Directory includes
		dirStr := strings.TrimPrefix(includeStr, "directory(")
		dirStr = strings.TrimSuffix(dirStr, ")")
		dirStr = strings.Trim(dirStr, "\"'")
		return handleDirectoryInclude(dirStr, isRequired, currentFile)

	case strings.Contains(includeStr, "*"):
		// Glob pattern includes
		return handleGlobInclude(includeStr, isRequired, currentFile)

	default:
		// Regular file include
		return handleFileInclude(includeStr, isRequired, currentFile)
	}
}

// applyVariables applies the stored variables to environment variables
func applyVariables() error {
	mutex.Lock()
	defer mutex.Unlock()

	// Create a new map with prefixed keys
	prefixedVariables := make(map[string]string)
	for key, value := range variables {
		prefixedKey := prefix + strings.ToLower(strings.ReplaceAll(key, ".", "."))
		prefixedVariables[prefixedKey] = value

		if err := os.Setenv(prefixedKey, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", prefixedKey, err)
		}
	}

	// Replace the original variables map with the prefixed version
	variables = prefixedVariables

	return nil
}
