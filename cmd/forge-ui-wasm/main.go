package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alexandremahdhaoui/forge-ui/internal/render"
)

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	var cmd render.Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing command: %v\n", err)
		os.Exit(1)
	}

	html, err := render.Execute(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error rendering: %v\n", err)
		os.Exit(1)
	}

	if _, err := fmt.Fprint(os.Stdout, html); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}
}
