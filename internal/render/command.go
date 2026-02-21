package render

import "encoding/json"

// Command represents a rendering request from the browser.
// It is received as JSON on stdin by the WASM module.
type Command struct {
	Action string          `json:"action"` // "render"
	Page   string          `json:"page"`   // "portfolios", "portfolio", "workspace", "forge"
	Theme  string          `json:"theme"`  // "light", "dark"
	Sort   string          `json:"sort"`   // "name", "time"
	Data   json.RawMessage `json:"data"`
}

// Page constants.
const (
	PagePortfolios = "portfolios"
	PagePortfolio  = "portfolio"
	PageWorkspace  = "workspace"
	PageForge      = "forge"
)
