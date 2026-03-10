package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

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
