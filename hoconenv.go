package hoconenv

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load loads the HOCON configuration file from the current directory or a specified path.
func Load(fileName ...string) error {
	// If no fileName is passed, search for a default file in the current directory
	var filePath string
	if len(fileName) == 0 {
		filePath = "application.conf" // Default file name
	} else {
		filePath = fileName[0] // Use the first argument as the file path
	}

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("config file %s does not exist", filePath)
	}

	// Open the config file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	config := make(map[string]string)
	scanner := bufio.NewScanner(file)
	var currentKey string
	keyStack := []string{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || line == "" {
			continue
		}

		// Handle nested keys
		if strings.HasSuffix(line, "{") {
			key := strings.TrimSpace(strings.TrimSuffix(line, "{"))
			keyStack = append(keyStack, key)
			currentKey = strings.Join(keyStack, ".")
			continue
		}

		// Handle closing of nested blocks
		if line == "}" {
			if len(keyStack) > 0 {
				keyStack = keyStack[:len(keyStack)-1]
			}
			currentKey = strings.Join(keyStack, ".")
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid syntax in line: %s", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes from values
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = value[1 : len(value)-1]
		}

		// Remove inline comments
		if idx := strings.Index(value, "#"); idx != -1 {
			value = value[:idx]
		}

		// Trim any remaining spaces
		value = strings.TrimSpace(value)

		// Combine with parent key if nested
		if currentKey != "" {
			key = currentKey + "." + key
		}

		// Convert to environment variable format
		envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		config[envKey] = value

		// Set environment variable
		os.Setenv(envKey, value)

		// Check for additional file references (optional)
		if strings.HasPrefix(value, "include ") {
			includeFile := strings.TrimSpace(value[len("include "):])
			if err := Load(includeFile); err != nil {
				return fmt.Errorf("failed to load included file %s: %w", includeFile, err)
			}
		}
	}

	return scanner.Err()
}
