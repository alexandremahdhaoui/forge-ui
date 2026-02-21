package main

import (
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

	html, err := render.Execute(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error rendering: %v\n", err)
		os.Exit(1)
	}

	if _, err := fmt.Fprint(os.Stdout, html); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}
}
