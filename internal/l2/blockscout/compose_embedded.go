package blockscout

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed docker-compose.blockscout.yml
var embeddedDockerFS embed.FS

const dockerFileName = "docker-compose.blockscout.yml"

func ensureDockerFile(localnetDir string) (string, error) {
	dockerPath := filepath.Join(localnetDir, dockerFileName)

	content, err := embeddedDockerFS.ReadFile(dockerFileName)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded %s: %w", dockerFileName, err)
	}

	if err := os.MkdirAll(filepath.Dir(dockerPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create %s directory: %w", localnetDir, err)
	}

	if err := os.WriteFile(dockerPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", dockerFileName, err)
	}

	return dockerPath, nil
}
