package hoconenv

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
