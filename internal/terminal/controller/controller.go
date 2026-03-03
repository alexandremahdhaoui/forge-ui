package controller

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
)

// SessionController manages the lifecycle of a terminal session.
// Start blocks until the session ends or Stop is called.
type SessionController interface {
	Start(cfg types.TerminalConfig, termIO adapter.TerminalIO) error
	Stop() error
}
