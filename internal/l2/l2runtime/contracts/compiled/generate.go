//go:build ignore

// This program regenerates compiled/contracts.json from the local contracts
// checkout via `forge inspect`. Run from the repo root:
//
//	go run ./internal/l2/l2runtime/contracts/compiled/generate.go
//
// Requires a usable `forge` on $PATH and configs/config.yaml pointing at a
// valid ethera-contracts local-path.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const contractsSubPath = "contracts/L2"

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: generate.go <path-to-local-testnet-root>")
	}
	rootDir := os.Args[1]
	contractsDir := filepath.Join(rootDir, contractsSubPath)

	targets := []string{
		"UniversalBridgeMailbox",
		"ComposeL2ToL2Bridge",
		"CetFactory",
		"ComposeETHLiquidity",
		"USDCMintable",
	}

	out := make(map[string]map[string]any, len(targets))
	for _, name := range targets {
		abiBytes, err := forgeInspect(contractsDir, name, "abi", "--json")
		if err != nil {
			log.Fatalf("forge inspect %s abi: %v", name, err)
		}
		bytecodeBytes, err := forgeInspect(contractsDir, name, "bytecode")
		if err != nil {
			log.Fatalf("forge inspect %s bytecode: %v", name, err)
		}

		out[name] = map[string]any{
			"abi":      json.RawMessage(abiBytes),
			"bytecode": strings.TrimSpace(string(bytecodeBytes)),
		}
		fmt.Fprintf(os.Stderr, "  %s  abi=%d bytecode=%d\n", name, len(abiBytes), len(bytecodeBytes))
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	if _, err := os.Stdout.Write(data); err != nil {
		log.Fatalf("write: %v", err)
	}
	os.Stdout.Write([]byte("\n"))
}

func forgeInspect(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("forge", append([]string{"inspect"}, args...)...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
