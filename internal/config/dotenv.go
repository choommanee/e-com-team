package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadDotenv reads a .env file (if present) and sets any variables that are not
// already defined in the process environment. Existing env vars always win, so
// real deployment configuration is never overridden by a local .env.
//
// Format: KEY=VALUE per line; blank lines and lines starting with # are
// ignored; surrounding quotes on the value are stripped.
func LoadDotenv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // no .env is fine
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
