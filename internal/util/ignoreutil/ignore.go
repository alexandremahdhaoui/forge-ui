// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ignoreutil

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// FileName is the name of the ignore file read from each scanned directory.
const FileName = ".forge-workspace-ignore"

// Load reads the ignore file from dir and returns the parsed patterns.
// Returns nil if the file does not exist or cannot be read.
// Blank lines and lines starting with # are skipped.
// Trailing whitespace is trimmed from each pattern.
func Load(dir string) []string {
	f, err := os.Open(filepath.Join(dir, FileName))
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip leading slashes — patterns match directory names, not paths.
		line = strings.TrimLeft(line, "/")
		if line == "" {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// IsIgnored returns true if name matches any of the given patterns.
// Patterns use filepath.Match syntax (*, ?, [abc], [a-z]).
// Malformed patterns are silently skipped.
func IsIgnored(name string, patterns []string) bool {
	for _, p := range patterns {
		if matched, err := filepath.Match(p, name); err == nil && matched {
			return true
		}
	}
	return false
}
