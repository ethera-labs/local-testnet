package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethera-labs/local-testnet/internal/logger"
)

// Repository represents a git repository to clone
type Repository struct {
	Name string
	URL  string
	Ref  string // branch, tag, or commit
}

// Cloner handles git repository operations
type Cloner struct {
	logger *slog.Logger
}

// NewCloner creates a new git cloner
func NewCloner() *Cloner {
	return &Cloner{
		logger: logger.Named("cloner"),
	}
}

// CloneAll clones multiple repositories in parallel
func (c *Cloner) CloneAll(ctx context.Context, destDir string, repos []Repository) error {
	c.logger.Info("cloning repositories", "count", len(repos), "destination", destDir)

	for _, repo := range repos {
		if err := c.Clone(ctx, destDir, repo); err != nil {
			return fmt.Errorf("failed to clone %s: %w", repo.Name, err)
		}
	}

	c.logger.Info("all repositories cloned successfully")

	return nil
}

// Clone clones a single repository
func (c *Cloner) Clone(ctx context.Context, destDir string, repo Repository) error {
	logger := c.logger.With("name", repo.Name).With("url", repo.URL)
	repoPath := filepath.Join(destDir, repo.Name)

	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		logger.Info("repository already cloned, skipping")
		return nil
	}

	c.logger.Info("cloning repository")

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", repo.Ref, repo.URL, repoPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	c.logger.Info("repository cloned successfully")

	return nil
}

// CheckoutBranch checks out the requested branch in the local repository at
// repoPath. It is a no-op if HEAD is already on branch. It refuses to switch
// when there are uncommitted tracked changes (untracked files are tolerated,
// since the orchestrator routinely leaves generated artifacts in the working
// tree). If the branch only exists on origin, CheckoutBranch fetches and
// creates a local tracking branch.
func (c *Cloner) CheckoutBranch(ctx context.Context, repoPath, branch string) error {
	logger := c.logger.With("path", repoPath).With("branch", branch)

	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return fmt.Errorf("not a git repository at %s: %w", repoPath, err)
	}

	current, err := runGit(ctx, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	current = strings.TrimSpace(current)

	if current == branch {
		logger.Info("local repository already on requested branch")
		return nil
	}

	dirty, err := runGit(ctx, repoPath, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return fmt.Errorf("check working tree: %w", err)
	}
	if strings.TrimSpace(dirty) != "" {
		return fmt.Errorf(
			"local repository at %s has uncommitted changes; refusing to switch from %q to %q. Commit, stash, or switch manually",
			repoPath, current, branch,
		)
	}

	logger.With("from", current).Info("switching local repository branch")

	if _, err := runGit(ctx, repoPath, "checkout", branch); err == nil {
		return nil
	}

	if _, ferr := runGit(ctx, repoPath, "fetch", "origin", branch); ferr != nil {
		return fmt.Errorf("checkout %q failed and fetch origin %q failed: %w", branch, branch, ferr)
	}
	if _, err := runGit(ctx, repoPath, "checkout", "-B", branch, "--track", "origin/"+branch); err != nil {
		return fmt.Errorf("checkout %q failed even after fetch: %w", branch, err)
	}

	return nil
}

// runGit runs `git -C dir <args...>` and returns combined stdout+stderr.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	out, err := exec.CommandContext(ctx, "git", full...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
