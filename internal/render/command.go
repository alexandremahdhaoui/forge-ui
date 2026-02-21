package render

import (
	"net/url"
	"strings"
)

// Input represents the parsed input from stdin.
// Format: "/route?key=value&key=value"
// Examples:
//
//	"/portfolios"
//	"/portfolios?sort=name&theme=dark"
//	"/portfolios/infrastructure"
//	"/portfolios/infrastructure/workspaces/platform"
//	"/portfolios/infrastructure/workspaces/platform/repos/forge"
type Input struct {
	Route string
	Sort  string // "name" or "time"; defaults to "time"
	Theme string // "light" or "dark"; defaults to "light"
}

// ParseInput parses a raw stdin string into an Input.
// The input format is a URL path with optional query parameters.
func ParseInput(raw string) Input {
	raw = strings.TrimSpace(raw)

	// Strip leading "#" if present (hash routing).
	raw = strings.TrimPrefix(raw, "#")

	// Ensure leading slash.
	if raw == "" || raw[0] != '/' {
		raw = "/" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return Input{Route: "/portfolios", Sort: "time", Theme: "light"}
	}

	sort := u.Query().Get("sort")
	if sort == "" {
		sort = "time"
	}

	theme := u.Query().Get("theme")
	if theme == "" {
		theme = "light"
	}

	return Input{
		Route: u.Path,
		Sort:  sort,
		Theme: theme,
	}
}
