package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func loadEnvCandidates() {
	candidates := orderedEnvCandidates()
	visited := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, exists := visited[absolute]; exists {
			continue
		}
		visited[absolute] = struct{}{}
		loadEnvFile(absolute)
	}
}

func orderedEnvCandidates() []string {
	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
	}
	if executablePath, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executablePath)
		candidates = append(candidates,
			filepath.Join(executableDir, ".env"),
			filepath.Join(filepath.Dir(executableDir), ".env"),
		)
	}
	return candidates
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, parseEnvValue(strings.TrimSpace(value)))
	}
}

func parseEnvValue(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			if unquoted, err := strconv.Unquote(`"` + strings.ReplaceAll(value[1:len(value)-1], `"`, `\"`) + `"`); err == nil {
				return unquoted
			}
			return value[1 : len(value)-1]
		}
	}
	return value
}
