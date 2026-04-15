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
func ComposeBuild(ctx context.Context, composeFilePath string, env map[string]string, services ...string) error {
	args := composeBuildArgs(services)
	return composeRun(ctx, composeFilePath, env, args...)
}

// ComposeBuildMultiFile builds docker compose services using multiple compose files.
func ComposeBuildMultiFile(ctx context.Context, composeFilePaths []string, env map[string]string, services ...string) error {
	args := composeBuildArgs(services)
	return composeRunMultiFile(ctx, composeFilePaths, env, args...)
}

// ComposeUp starts docker compose services in detached mode.
func ComposeUp(ctx context.Context, composeFilePath string, env map[string]string, services ...string) error {
	args := composeUpArgs(services, false, false)
	return composeRun(ctx, composeFilePath, env, args...)
}

// ComposeRestart restarts docker compose services.
func ComposeRestart(ctx context.Context, composeFilePath string, env map[string]string, services ...string) error {
	args := composeUpArgs(services, true, false)
	return composeRun(ctx, composeFilePath, env, args...)
}

// ComposeRestartMultiFile restarts docker compose services using multiple compose files.
func ComposeRestartMultiFile(ctx context.Context, composeFilePaths []string, env map[string]string, services ...string) error {
	args := composeUpArgs(services, true, false)
	return composeRunMultiFile(ctx, composeFilePaths, env, args...)
}

// ComposeRestartNoDepsMultiFile restarts docker compose services using multiple compose files
// without recreating dependencies.
func ComposeRestartNoDepsMultiFile(ctx context.Context, composeFilePaths []string, env map[string]string, services ...string) error {
	args := composeUpArgs(services, true, true)
	return composeRunMultiFile(ctx, composeFilePaths, env, args...)
}

// ComposeDown stops docker compose services.
func ComposeDown(ctx context.Context, composeFilePath string, env map[string]string, removeVolumes bool) error {
	args := []string{"down"}
	if removeVolumes {
		args = append(args, "-v")
	}
	return composeRun(ctx, composeFilePath, env, args...)
}

// ComposeUpMultiFile starts docker compose services using multiple compose files.
func ComposeUpMultiFile(ctx context.Context, composeFilePaths []string, env map[string]string, services ...string) error {
	args := composeUpArgs(services, false, false)
	return composeRunMultiFile(ctx, composeFilePaths, env, args...)
}

func composeUpArgs(services []string, forceRecreate, noDeps bool) []string {
	args := []string{"up", "-d", "--no-build"}
	if forceRecreate {
		args = append(args, "--force-recreate")
	}
	if noDeps {
		args = append(args, "--no-deps")
	}
	return append(args, services...)
}

// composeRunMultiFile executes a docker compose command with multiple compose files.
func composeRunMultiFile(ctx context.Context, composeFilePaths []string, env map[string]string, args ...string) error {
	if len(composeFilePaths) == 0 {
		return fmt.Errorf("no compose files provided")
	}

	fullArgs := []string{"compose"}
	for _, path := range composeFilePaths {
		fullArgs = append(fullArgs, "-f", path)
	}
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", fullArgs...)
	cmd.Dir = filepath.Dir(composeFilePaths[0])

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

// composeRun executes a docker compose command with environment variables.
func composeRun(ctx context.Context, composeFilePath string, env map[string]string, args ...string) error {
	fullArgs := append([]string{"compose", "-f", composeFilePath}, args...)
	cmd := exec.CommandContext(ctx, "docker", fullArgs...)
	cmd.Dir = filepath.Dir(composeFilePath)

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
