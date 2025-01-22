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

// Config helds the configuration settings for the HOCON loader
type Config struct {
	prefix      string
	loadedFiles map[string]bool
	mutex       sync.RWMutex
	variables   map[string]string
}

// Options represents configuration options for loading HOCON files
type Options struct {
	IgnoreErrors    bool
	OverwriteEnv    bool
	DefaultPrefix   string
	IncludePatterns []string
}

type IncludeType int

const (
	IncludeFile IncludeType = iota
	IncludeURL
	IncludeDirectory
	IncludeRequired
	IncludeOptional
)

// DefaultOptions returns the default configuration options
func DefaultOptions() Options {
	return Options{
		IgnoreErrors:    false,
		OverwriteEnv:    true,
		DefaultPrefix:   "",
		IncludePatterns: []string{".conf", ".hocon"},
	}
}

var defaultConfig = &Config{
	loadedFiles: make(map[string]bool),
	variables:   make(map[string]string),
}

// NewConfig creates a new config instance
func NewConfig() *Config {
	return &Config{
		loadedFiles: make(map[string]bool),
		variables:   make(map[string]string),
	}
}

func SetPrefix(prefix string) {
	defaultConfig.SetPrefix(prefix)
}

// SetPrefix sets the prefix for environment variables for a specific config instance
func (c *Config) SetPrefix(prefix string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.prefix = strings.ToUpper(strings.TrimSpace(prefix)) + "_"
}

// Load loads the HOCON configuration file using default options
func Load(fileName ...string) error {
	return defaultConfig.Load(DefaultOptions(), fileName...)
}

// LoadWithOptions loads the HOCON configuration file with custom options
func LoadWithOptions(opts Options, fileName ...string) error {
	return defaultConfig.LoadWithOptions(opts, fileName...)
}

// Load loads the HOCON configuration file for a specific config instance
func (c *Config) Load(opts Options, fileName ...string) error {
	return c.LoadWithOptions(opts, fileName...)
}

// LoadWithOptions loads the HOCON configuration file with custom options for a specific config instance
func (c *Config) LoadWithOptions(opts Options, fileName ...string) error {
	// Set default prefix if provided
	if opts.DefaultPrefix != "" {
		c.SetPrefix(opts.DefaultPrefix)
	}

	// If no fileName is passed, search for default files
	if len(fileName) == 0 {
		found := false
		for _, pattern := range opts.IncludePatterns {
			matches, err := filepath.Glob("application" + pattern)
			if err == nil && len(matches) > 0 {
				for _, match := range matches {
					if err := c.loadFile(match, opts); err != nil && !opts.IgnoreErrors {
						return err
					}
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("no default configuration files found")
		}
		return nil
	}

	// Load all specified files
	for _, file := range fileName {
		if err := c.loadFile(file, opts); err != nil && !opts.IgnoreErrors {
			return err
		}
	}

	return nil
}

// loadFile handles the actual file loading logic
func (c *Config) loadFile(filePath string, opts Options) error {
	c.mutex.Lock()
	if c.loadedFiles[filePath] {
		c.mutex.Unlock()
		return nil // Skip already loaded files
	}
	c.loadedFiles[filePath] = true
	c.mutex.Unlock()

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

		if err := c.parseLine(line, &keyStack, opts, filePath, lineNum); err != nil && !opts.IgnoreErrors {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// Apply variables to environment
	return c.applyVariables(opts)
}

// parseLine handles parsing of individual HOCON lines
func (c *Config) parseLine(line string, keyStack *[]string, opts Options, filePath string, lineNum int) error {
	if strings.HasPrefix(line, "include ") {
		return c.handleInclude(line, opts, filePath)
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
		return c.handleInclude(value, opts, filePath)
	}

	// Process the value
	value = c.processValue(value)

	// Build the full key
	fullKey := c.buildFullKey(*keyStack, key)

	// Store the variable
	c.mutex.Lock()
	c.variables[fullKey] = value
	c.mutex.Unlock()

	return nil
}

// processValue handles value processing including quote removal and comment stripping
func (c *Config) processValue(value string) string {
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
func (c *Config) buildFullKey(keyStack []string, key string) string {
	if len(keyStack) > 0 {
		return strings.Join(keyStack, ".") + "." + key
	}
	return key
}

// handleInclude processes include directives
func (c *Config) handleInclude(value string, opts Options, currentFile string) error {
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
		return c.handleURLInclude(urlStr, isRequired, opts)

	case strings.HasPrefix(includeStr, "directory("):
		// Directory includes
		dirStr := strings.TrimPrefix(includeStr, "directory(")
		dirStr = strings.TrimSuffix(dirStr, ")")
		dirStr = strings.Trim(dirStr, "\"'")
		return c.handleDirectoryInclude(dirStr, isRequired, opts, currentFile)

	case strings.Contains(includeStr, "*"):
		// Glob pattern includes
		return c.handleGlobInclude(includeStr, isRequired, opts, currentFile)

	default:
		// Regular file include
		return c.handleFileInclude(includeStr, isRequired, opts, currentFile)
	}
}

// handleFileInclude processes a single file include
func (c *Config) handleFileInclude(file string, required bool, opts Options, currentFile string) error {
	if !filepath.IsAbs(file) {
		file = filepath.Join(filepath.Dir(currentFile), file)
	}

	if err := c.loadFile(file, opts); err != nil {
		if required {
			return fmt.Errorf("failed to include required file %s: %w", file, err)
		}
		// Log warning for optional includes
		fmt.Printf("Warning: Optional include file not found: %s\n", file)
	}
	return nil
}

// handleURLInclude processes URL includes (placeholder for future implementation)
func (c *Config) handleURLInclude(urlStr string, required bool, opts Options) error {
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

		if err := c.parseLine(line, &keyStack, opts, urlStr, lineNum); err != nil && !opts.IgnoreErrors {
			return err
		}
	}

	return scanner.Err()
}

// handleDirectoryInclude processes directory includes
func (c *Config) handleDirectoryInclude(dir string, required bool, opts Options, currentFile string) error {
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

		// Check if file matches configured patterns
		matched := false
		for _, pattern := range opts.IncludePatterns {
			if strings.HasSuffix(file.Name(), pattern) {
				matched = true
				break
			}
		}

		if matched {
			filePath := filepath.Join(dir, file.Name())
			if err := c.loadFile(filePath, opts); err != nil && required {
				return fmt.Errorf("failed to include file %s from directory: %w", filePath, err)
			}
		}
	}

	return nil
}

// handleGlobInclude processes glob pattern includes
func (c *Config) handleGlobInclude(pattern string, required bool, opts Options, currentFile string) error {
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
		if err := c.loadFile(match, opts); err != nil && required {
			return fmt.Errorf("failed to include file %s from glob: %w", match, err)
		}
	}

	return nil
}

// applyVariables applies the stored variables to environment variables
func (c *Config) applyVariables(opts Options) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for key, value := range c.variables {
		envKey := c.prefix + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))

		// Check if environment variable already exists
		if !opts.OverwriteEnv {
			if _, exists := os.LookupEnv(envKey); exists {
				continue
			}
		}

		if err := os.Setenv(envKey, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", envKey, err)
		}
	}

	return nil
}

// GetVariable returns the value of a stored variable
func (c *Config) GetVariable(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	value, exists := c.variables[key]
	return value, exists
}

// GetAllVariables returns a copy of all stored variables
func (c *Config) GetAllVariables() map[string]string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	variables := make(map[string]string, len(c.variables))
	for k, v := range c.variables {
		variables[k] = v
	}
	return variables
}

// ClearVariables removes all stored variables
func (c *Config) ClearVariables() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.variables = make(map[string]string)
	c.loadedFiles = make(map[string]bool)
}
