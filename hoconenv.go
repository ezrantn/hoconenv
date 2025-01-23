package hoconenv

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type includeType int

const (
	includeFile includeType = iota
	includeURL
	includeDirectory
	includeRequired
	includeOptional
)

var (
	variables   = make(map[string]string)
	loadedFiles = make(map[string]bool)
	mutex       sync.RWMutex
	prefix      string
)

func SetPrefix(p string) {
	mutex.Lock()
	defer mutex.Unlock()
	prefix = strings.ToLower(strings.TrimSpace(p)) + "."
}

// Load loads configuration from specified files or default application.* files.
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

// parseLine handles parsing of individual HOCsON lines
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

	// Handle quoted strings
	includeStr = strings.Trim(includeStr, "\"'")

	// Parse include type and path
	isRequired := true
	if strings.HasPrefix(includeStr, "optional(") && strings.HasSuffix(includeStr, ")") {
		isRequired = false
		includeStr = strings.TrimPrefix(includeStr, "optional(")
		includeStr = strings.TrimSuffix(includeStr, ")")
		includeStr = strings.Trim(includeStr, "\"'")
	}

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

// handleFileInclude processes a single file include
func handleFileInclude(file string, required bool, currentFile string) error {
	if !filepath.IsAbs(file) {
		file = filepath.Join(filepath.Dir(currentFile), file)
	}

	if err := loadFile(file); err != nil {
		if required {
			return fmt.Errorf("failed to include required file %s: %w", file, err)
		}
		// Log warning for optional includes
		fmt.Printf("Warning: Optional include file not found: %s\n", file)
	}
	return nil
}

// handleURLInclude processes URL includes (placeholder for future implementation)
func handleURLInclude(urlStr string, required bool) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		if required {
			return fmt.Errorf("invalid URL %s: %w", urlStr, err)
		}

		return nil
	}

	// Validate scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		if required {
			return fmt.Errorf("unsupported URL scheme %s, only http and https are supported", parsedURL.Scheme)
		}
		return nil
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(urlStr)
	if err != nil {
		if required {
			return fmt.Errorf("failed to fetch URL %s: %w", urlStr, err)
		}

		return nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if required {
			return fmt.Errorf("failed to fetch URL %s: status code %d", urlStr, resp.StatusCode)
		}

		return nil
	}

	scanner := bufio.NewScanner(resp.Body)
	var keyStack []string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		if err := parseLine(line, &keyStack, urlStr, lineNum); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// handleDirectoryInclude processes directory includes
func handleDirectoryInclude(dir string, required bool, currentFile string) error {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(filepath.Dir(currentFile), dir)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		if required {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}
		fmt.Printf("Warning: Optional include directory not found: %s\n", dir)
		return nil
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dir, file.Name())
		if err := loadFile(filePath); err != nil {
			if required {
				return fmt.Errorf("failed to include file %s from directory: %w", filePath, err)
			}

			fmt.Printf("Warning: Failed to include optional file %s: %v\n", filePath, err)
		}
	}

	return nil
}

// handleGlobInclude processes glob pattern includes
func handleGlobInclude(pattern string, required bool, currentFile string) error {
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(filepath.Dir(currentFile), pattern)
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		if required {
			return fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}
		fmt.Printf("Warning: Invalid optional glob pattern: %s\n", pattern)
		return nil
	}

	if len(matches) == 0 && required {
		return fmt.Errorf("no files found matching required pattern: %s", pattern)
	}

	for _, match := range matches {
		if err := loadFile(match); err != nil && required {
			return fmt.Errorf("failed to include file %s from glob: %w", match, err)
		}
	}

	return nil
}

// applyVariables applies the stored variables to environment variables
func applyVariables() error {
	mutex.RLock()
	defer mutex.RUnlock()

	for key, value := range variables {
		envKey := prefix + strings.ToLower(strings.ReplaceAll(key, ".", "."))

		if err := os.Setenv(envKey, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", envKey, err)
		}
	}

	return nil
}
