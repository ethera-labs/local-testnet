package docker

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed docker-compose.yml docker-compose.flashblocks.yml docker-compose.sidecar.yml docker-compose.frontend.yml
var embeddedDockerFS embed.FS

const (
	dockerFileName            = "docker-compose.yml"
	dockerFlashblocksFileName = "docker-compose.flashblocks.yml"
	dockerSidecarFileName     = "docker-compose.sidecar.yml"
	dockerFrontendFileName    = "docker-compose.frontend.yml"
)

// EnsureDockerFile ensures the docker-compose.yml file exists in the specified directory
// and returns its path. It always writes the embedded content to ensure the file is up-to-date.
// This allows the compose file to be used from anywhere (including when running
// the binary from a different directory).
func EnsureDockerFile(localnetDir string) (string, error) {
	dockerPath := filepath.Join(localnetDir, dockerFileName)

	content, err := getDockerFileContent()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(dockerPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create %s directory: %w", localnetDir, err)
	}

	if err := os.WriteFile(dockerPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", dockerFileName, err)
	}

	return dockerPath, nil
}

// getDockerFileContent returns the embedded docker-compose.yml content.
func getDockerFileContent() (string, error) {
	content, err := embeddedDockerFS.ReadFile(dockerFileName)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded %s: %w", dockerFileName, err)
	}
	return string(content), nil
}

// EnsureFlashblocksDockerFile ensures the docker-compose.flashblocks.yml file exists
// in the specified directory and returns its path.
func EnsureFlashblocksDockerFile(localnetDir string) (string, error) {
	dockerPath := filepath.Join(localnetDir, dockerFlashblocksFileName)

	content, err := getFlashblocksDockerContent()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(dockerPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create %s directory: %w", localnetDir, err)
	}

	if err := os.WriteFile(dockerPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", dockerFlashblocksFileName, err)
	}

	return dockerPath, nil
}

// getFlashblocksDockerContent returns the embedded docker-compose.flashblocks.yml content.
func getFlashblocksDockerContent() (string, error) {
	content, err := embeddedDockerFS.ReadFile(dockerFlashblocksFileName)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded %s: %w", dockerFlashblocksFileName, err)
	}
	return string(content), nil
}

// EnsureSidecarDockerFile ensures the docker-compose.sidecar.yml file exists
// in the specified directory and returns its path.
func EnsureSidecarDockerFile(localnetDir string) (string, error) {
	dockerPath := filepath.Join(localnetDir, dockerSidecarFileName)

	content, err := getSidecarDockerContent()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(dockerPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create %s directory: %w", localnetDir, err)
	}

	if err := os.WriteFile(dockerPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", dockerSidecarFileName, err)
	}

	return dockerPath, nil
}

// getSidecarDockerContent returns the embedded docker-compose.sidecar.yml content.
func getSidecarDockerContent() (string, error) {
	content, err := embeddedDockerFS.ReadFile(dockerSidecarFileName)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded %s: %w", dockerSidecarFileName, err)
	}
	return string(content), nil
}

// EnsureFrontendDockerFile ensures the docker-compose.frontend.yml file exists.
func EnsureFrontendDockerFile(localnetDir string) (string, error) {
	dockerPath := filepath.Join(localnetDir, dockerFrontendFileName)
	content, err := embeddedDockerFS.ReadFile(dockerFrontendFileName)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded %s: %w", dockerFrontendFileName, err)
	}
	if err := os.MkdirAll(filepath.Dir(dockerPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create %s directory: %w", localnetDir, err)
	}
	if err := os.WriteFile(dockerPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", dockerFrontendFileName, err)
	}
	return dockerPath, nil
}
