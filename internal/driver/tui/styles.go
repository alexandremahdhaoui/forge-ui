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

//go:build !js || !wasm

package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#059669")
	mutedColor     = lipgloss.Color("#6B7280")
	errorColor     = lipgloss.Color("#DC2626")
	warnColor      = lipgloss.Color("#D97706")

	// Layout styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(mutedColor).
			Padding(0, 1)

	sidebarStyle = lipgloss.NewStyle().
			Width(28).
			BorderStyle(lipgloss.NormalBorder()).
			BorderRight(true).
			BorderForeground(mutedColor).
			Padding(1, 1)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	statusRunning = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	statusSuspended = lipgloss.NewStyle().
			Foreground(warnColor)

	statusError = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F9FAFB")).
			Padding(0, 1)
)

// statusStyle returns the appropriate style for a phase string.
func statusStyle(phase string) lipgloss.Style {
	switch phase {
	case "Running", "passed":
		return statusRunning
	case "Suspended":
		return statusSuspended
	case "Failed", "Error", "failed":
		return statusError
	default:
		return normalStyle
	}
}
