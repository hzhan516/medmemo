// read-version reads productVersion from a Wails project JSON file.
// Usage: go run scripts/read-version.go wails.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type wailsConfig struct {
	Info struct {
		ProductVersion string `json:"productVersion"`
	} `json:"info"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: read-version <wails.json>")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	var cfg wailsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	if cfg.Info.ProductVersion == "" {
		fmt.Fprintln(os.Stderr, "productVersion not found in wails.json")
		os.Exit(1)
	}

	fmt.Print(cfg.Info.ProductVersion)
}
