package blockscout

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed docker-compose.blockscout.yml
var embeddedComposeFS embed.FS

const composeFileName = "docker-compose.blockscout.yml"

func ensureComposeFile(localnetDir string) (string, error) {
	composePath := filepath.Join(localnetDir, composeFileName)

	content, err := embeddedComposeFS.ReadFile(composeFileName)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded %s: %w", composeFileName, err)
	}

	if err := os.MkdirAll(filepath.Dir(composePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create %s directory: %w", localnetDir, err)
	}

	if err := os.WriteFile(composePath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", composeFileName, err)
	}

	return composePath, nil
}
