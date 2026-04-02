package l2

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/git"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

type recordingCloner struct {
	repos []git.Repository
}

func (c *recordingCloner) CloneAll(_ context.Context, _ string, repos []git.Repository) error {
	c.repos = append([]git.Repository(nil), repos...)
	return nil
}

func TestCloneRepositoriesSkipsDisabledFeatureRepositories(t *testing.T) {
	t.Parallel()

	cloner := &recordingCloner{}
	service := &Service{
		rootDir: filepath.Join(t.TempDir(), "workspace"),
		cloner:  cloner,
		logger:  logger.Named("test"),
	}

	cfg := configs.L2{
		Repositories: map[configs.RepositoryName]configs.Repository{
			configs.RepositoryNameOpGeth: {
				URL:    "https://example.com/op-geth.git",
				Branch: "main",
			},
			configs.RepositoryNamePublisher: {
				URL:    "https://example.com/publisher.git",
				Branch: "main",
			},
			configs.RepositoryNameEtheraContracts: {
				URL:    "https://example.com/ethera-contracts.git",
				Branch: "main",
			},
			configs.RepositoryNameOpRbuilder: {
				URL:    "https://example.com/op-rbuilder.git",
				Branch: "stage",
			},
			configs.RepositoryNameSidecar: {
				URL:    "https://example.com/sidecar.git",
				Branch: "main",
			},
		},
	}

	if err := service.cloneRepositories(context.Background(), cfg); err != nil {
		t.Fatalf("expected cloneRepositories to succeed, got: %v", err)
	}

	got := repositoryNames(cloner.repos)
	want := map[string]bool{
		string(configs.RepositoryNameOpGeth):          true,
		string(configs.RepositoryNamePublisher):       true,
		string(configs.RepositoryNameEtheraContracts): true,
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected repository count: got %d, want %d (%v)", len(got), len(want), got)
	}
	for name := range want {
		if !got[name] {
			t.Fatalf("expected repository %q to be cloned, got %v", name, got)
		}
	}
}

func TestCloneRepositoriesIncludesEnabledFeatureRepositories(t *testing.T) {
	t.Parallel()

	cloner := &recordingCloner{}
	service := &Service{
		rootDir: filepath.Join(t.TempDir(), "workspace"),
		cloner:  cloner,
		logger:  logger.Named("test"),
	}

	cfg := configs.L2{
		Flashblocks: configs.FlashblocksConfig{Enabled: true},
		Sidecar:     configs.SidecarConfig{Enabled: true},
		Repositories: map[configs.RepositoryName]configs.Repository{
			configs.RepositoryNameOpGeth: {
				URL:    "https://example.com/op-geth.git",
				Branch: "main",
			},
			configs.RepositoryNamePublisher: {
				URL:    "https://example.com/publisher.git",
				Branch: "main",
			},
			configs.RepositoryNameEtheraContracts: {
				URL:    "https://example.com/ethera-contracts.git",
				Branch: "main",
			},
			configs.RepositoryNameOpRbuilder: {
				URL:    "https://example.com/op-rbuilder.git",
				Branch: "stage",
			},
			configs.RepositoryNameSidecar: {
				URL:    "https://example.com/sidecar.git",
				Branch: "main",
			},
		},
	}

	if err := service.cloneRepositories(context.Background(), cfg); err != nil {
		t.Fatalf("expected cloneRepositories to succeed, got: %v", err)
	}

	got := repositoryNames(cloner.repos)
	for _, name := range []configs.RepositoryName{
		configs.RepositoryNameOpGeth,
		configs.RepositoryNamePublisher,
		configs.RepositoryNameEtheraContracts,
		configs.RepositoryNameOpRbuilder,
		configs.RepositoryNameSidecar,
	} {
		if !got[string(name)] {
			t.Fatalf("expected repository %q to be cloned, got %v", name, got)
		}
	}
}

func repositoryNames(repos []git.Repository) map[string]bool {
	names := make(map[string]bool, len(repos))
	for _, repo := range repos {
		names[repo.Name] = true
	}
	return names
}
