package render

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").ParseFS(templateFS, "templates/*.html")
	if err != nil {
		panic(fmt.Sprintf("render: failed to parse templates: %v", err))
	}
}

// Execute processes a Command and returns the rendered HTML content.
func Execute(cmd Command) (string, error) {
	switch cmd.Page {
	case PagePortfolios:
		return renderPage("portfolios", cmd)
	case PagePortfolio:
		return renderPage("portfolio", cmd)
	case PageWorkspace:
		return renderPage("workspace", cmd)
	case PageForge:
		return renderPage("forge", cmd)
	default:
		return "", fmt.Errorf("unknown page: %q", cmd.Page)
	}
}

func renderPage(name string, cmd Command) (string, error) {
	data, err := unmarshalPageData(name, cmd)
	if err != nil {
		return "", fmt.Errorf("unmarshal %s data: %w", name, err)
	}

	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

func unmarshalPageData(page string, cmd Command) (any, error) {
	switch page {
	case "portfolios":
		var data model.PortfoliosPageData
		if err := json.Unmarshal(cmd.Data, &data); err != nil {
			return nil, err
		}
		data.DarkMode = cmd.Theme == "dark"
		if cmd.Sort != "" {
			data.SortMode = cmd.Sort
		}
		if data.SortMode == "" {
			data.SortMode = "time"
		}
		data.HomeURL = "#"
		return data, nil

	case "portfolio":
		var data model.PortfolioPageData
		if err := json.Unmarshal(cmd.Data, &data); err != nil {
			return nil, err
		}
		data.DarkMode = cmd.Theme == "dark"
		if cmd.Sort != "" {
			data.SortMode = cmd.Sort
		}
		if data.SortMode == "" {
			data.SortMode = "time"
		}
		data.HomeURL = "#"
		return data, nil

	case "workspace":
		var data model.WorkspacePageData
		if err := json.Unmarshal(cmd.Data, &data); err != nil {
			return nil, err
		}
		data.DarkMode = cmd.Theme == "dark"
		if cmd.Sort != "" {
			data.SortMode = cmd.Sort
		}
		if data.SortMode == "" {
			data.SortMode = "time"
		}
		data.HomeURL = "#"
		return data, nil

	case "forge":
		var data model.ForgePageData
		if err := json.Unmarshal(cmd.Data, &data); err != nil {
			return nil, err
		}
		data.DarkMode = cmd.Theme == "dark"
		data.HomeURL = "#"
		return data, nil

	default:
		return nil, fmt.Errorf("unknown page: %q", page)
	}
}
