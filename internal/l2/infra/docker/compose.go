package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DockerBuild builds docker compose services.
func DockerBuild(ctx context.Context, dockerFilePath string, env map[string]string, services ...string) error {
	args := append([]string{"build", "--parallel"}, services...)
	return dockerRun(ctx, dockerFilePath, env, args...)
}

// DockerBuildMultiFile builds docker compose services using multiple compose files.
func DockerBuildMultiFile(ctx context.Context, dockerFilePaths []string, env map[string]string, services ...string) error {
	args := append([]string{"build", "--parallel"}, services...)
	return dockerRunMultiFile(ctx, dockerFilePaths, env, args...)
}

// DockerUp starts docker compose services in detached mode.
func DockerUp(ctx context.Context, dockerFilePath string, env map[string]string, services ...string) error {
	args := append([]string{"up", "-d", "--no-build"}, services...)
	return dockerRun(ctx, dockerFilePath, env, args...)
}

// DockerRestart restarts docker compose services.
func DockerRestart(ctx context.Context, dockerFilePath string, env map[string]string, services ...string) error {
	args := append([]string{"up", "-d", "--force-recreate", "--no-build"}, services...)
	return dockerRun(ctx, dockerFilePath, env, args...)
}

// DockerRestartMultiFile restarts docker compose services using multiple compose files.
func DockerRestartMultiFile(ctx context.Context, dockerFilePaths []string, env map[string]string, services ...string) error {
	args := append([]string{"up", "-d", "--force-recreate", "--no-build"}, services...)
	return dockerRunMultiFile(ctx, dockerFilePaths, env, args...)
}

// DockerDown stops docker compose services.
func DockerDown(ctx context.Context, dockerFilePath string, env map[string]string, removeVolumes bool) error {
	args := []string{"down"}
	if removeVolumes {
		args = append(args, "-v")
	}
	return dockerRun(ctx, dockerFilePath, env, args...)
}

// DockerUpMultiFile starts docker compose services using multiple compose files.
func DockerUpMultiFile(ctx context.Context, dockerFilePaths []string, env map[string]string, services ...string) error {
	args := append([]string{"up", "-d", "--no-build"}, services...)
	return dockerRunMultiFile(ctx, dockerFilePaths, env, args...)
}

// dockerRunMultiFile executes a docker compose command with multiple compose files.
func dockerRunMultiFile(ctx context.Context, dockerFilePaths []string, env map[string]string, args ...string) error {
	if len(dockerFilePaths) == 0 {
		return fmt.Errorf("no docker files provided")
	}

	fullArgs := []string{"compose"}
	for _, path := range dockerFilePaths {
		fullArgs = append(fullArgs, "-f", path)
	}
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", fullArgs...)
	cmd.Dir = filepath.Dir(dockerFilePaths[0])

	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose %s failed: %w", strings.Join(args, " "), err)
	}

	return nil
}

// dockerRun executes a docker compose command with environment variables.
func dockerRun(ctx context.Context, dockerFilePath string, env map[string]string, args ...string) error {
	fullArgs := append([]string{"compose", "-f", dockerFilePath}, args...)
	cmd := exec.CommandContext(ctx, "docker", fullArgs...)
	cmd.Dir = filepath.Dir(dockerFilePath)

	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose %s failed: %w", strings.Join(args, " "), err)
	}

	return nil
}
