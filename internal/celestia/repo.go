package celestia

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethera-labs/local-testnet/configs"
)

type preparedRepository struct {
	ContainerPath string
	HostPath      string
}

func prepareRepository(ctx context.Context, rootDir, servicesDir string, repo configs.Repository, defaultURL, defaultRef, name string) (preparedRepository, error) {
	if repo.LocalPath == "" && repo.URL == "" {
		repo.URL = defaultURL
		repo.Branch = defaultRef
	}
	if repo.URL != "" && repo.Branch == "" {
		repo.Branch = defaultRef
	}

	if repo.LocalPath != "" {
		resolvedPath, err := resolvePath(rootDir, repo.LocalPath)
		if err != nil {
			return preparedRepository{}, fmt.Errorf("failed to resolve local path for %s: %w", name, err)
		}
		hostPath, err := toHostPath(resolvedPath)
		if err != nil {
			return preparedRepository{}, fmt.Errorf("failed to resolve host path for %s: %w", name, err)
		}
		return preparedRepository{
			ContainerPath: resolvedPath,
			HostPath:      hostPath,
		}, nil
	}

	repoPath := filepath.Join(servicesDir, name)
	if err := ensureRepositoryAtRef(ctx, repoPath, repo.URL, repo.Branch); err != nil {
		return preparedRepository{}, fmt.Errorf("failed to prepare repository %s: %w", name, err)
	}
	hostPath, err := toHostPath(repoPath)
	if err != nil {
		return preparedRepository{}, fmt.Errorf("failed to resolve host path for cloned repository %s: %w", name, err)
	}

	return preparedRepository{
		ContainerPath: repoPath,
		HostPath:      hostPath,
	}, nil
}

func ensureRepositoryAtRef(ctx context.Context, repoPath, url, ref string) error {
	if url == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}
	if ref == "" {
		return fmt.Errorf("repository ref cannot be empty")
	}

	gitDir := filepath.Join(repoPath, ".git")
	if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
		if err := runGit(ctx, repoPath, "fetch", "--tags", "origin", ref); err != nil {
			return fmt.Errorf("failed to fetch ref %q: %w", ref, err)
		}
		if err := runGit(ctx, repoPath, "checkout", "--detach", "FETCH_HEAD"); err != nil {
			return fmt.Errorf("failed to checkout ref %q: %w", ref, err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return fmt.Errorf("failed to create repository parent directory: %w", err)
	}

	if err := runCmd(ctx, "", "git", "clone", "--depth", "1", "--branch", ref, url, repoPath); err == nil {
		return nil
	}

	_ = os.RemoveAll(repoPath)
	if err := runCmd(ctx, "", "git", "clone", "--depth", "1", url, repoPath); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	if err := runGit(ctx, repoPath, "fetch", "--tags", "origin", ref); err != nil {
		return fmt.Errorf("failed to fetch ref %q after clone: %w", ref, err)
	}
	if err := runGit(ctx, repoPath, "checkout", "--detach", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("failed to checkout ref %q after clone: %w", ref, err)
	}

	return nil
}

func runGit(ctx context.Context, repoPath string, args ...string) error {
	fullArgs := append([]string{"-C", repoPath}, args...)
	return runCmd(ctx, "", "git", fullArgs...)
}

func runCmd(ctx context.Context, dir, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
