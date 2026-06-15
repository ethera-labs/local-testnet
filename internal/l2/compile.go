package l2

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/git"
	"github.com/ethera-labs/local-testnet/internal/l2/l2runtime/contracts"
	"github.com/spf13/cobra"
)

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile L2 contracts from contracts repository",
	Long:  "Compiles Solidity contracts for L2 deployment and generates contracts.json (and entrypoint.json when the account-abstraction repository is configured).",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("running contract compilation command")
		ctx := cmd.Context()
		rootDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		cloner := git.NewCloner()
		servicesDir := filepath.Join(rootDir, localnetDirName, servicesDirName)
		outputDir := filepath.Join(rootDir, localnetDirName, compiledContractsDirName)

		if err := compileEtheraContracts(ctx, cloner, servicesDir, outputDir); err != nil {
			return err
		}

		if err := compileEntryPoint(ctx, cloner, servicesDir, outputDir); err != nil {
			return err
		}

		slog.Info("contract compilation completed successfully")

		return nil
	},
}

// compileEtheraContracts clones the ethera-contracts repository configured
// under `l2.repositories.ethera-contracts` and compiles every contract listed
// in [contracts.Contracts] into `compiled/contracts.json`. The repository is
// required; the command fails when it is not configured.
func compileEtheraContracts(ctx context.Context, cloner *git.Cloner, servicesDir, outputDir string) error {
	repo, ok := findRepository(configs.RepositoryNameEtheraContracts)
	if !ok {
		return fmt.Errorf("could not find: '%s' repository in the configuration", configs.RepositoryNameEtheraContracts)
	}

	slog.With("name", repo.Name).Info("cloning ethera-contracts repository")
	if err := cloner.Clone(ctx, servicesDir, repo); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	compiler := contracts.NewCompiler(
		filepath.Join(servicesDir, "ethera-contracts", "L2"),
		outputDir,
	)

	contractsToCompile := contractNames(contracts.Contracts)
	slog.Info("starting Ethera contract compilation", "contracts", contractsToCompile)
	if err := compiler.Compile(ctx, contractsToCompile); err != nil {
		return fmt.Errorf("ethera contract compilation failed: %w", err)
	}
	return nil
}

// compileEntryPoint clones the account-abstraction repository configured under
// `l2.repositories.account-abstraction` and compiles every contract listed in
// [contracts.EntryPointContracts] into `compiled/entrypoint.json`, including
// `deployedBytecode` so the bundler can apply `EntryPointSimulations` as an
// `eth_call` state override.
//
// The repository is optional: when it is not configured the function logs and
// returns nil so stacks that do not run the ERC-4337 bundler are unaffected.
func compileEntryPoint(ctx context.Context, cloner *git.Cloner, servicesDir, outputDir string) error {
	repo, ok := findRepository(configs.RepositoryNameAccountAbstraction)
	if !ok {
		slog.With("name", configs.RepositoryNameAccountAbstraction).
			Info("account-abstraction repository not configured; skipping EntryPoint compilation")
		return nil
	}

	slog.With("name", repo.Name).Info("cloning account-abstraction repository")
	if err := cloner.Clone(ctx, servicesDir, repo); err != nil {
		return fmt.Errorf("failed to clone account-abstraction: %w", err)
	}

	compiler := contracts.NewCompiler(
		filepath.Join(servicesDir, "account-abstraction"),
		outputDir,
	)

	contractsToCompile := contractNames(contracts.EntryPointContracts)
	slog.Info("starting EntryPoint compilation", "contracts", contractsToCompile)
	if err := compiler.CompileEntryPoint(ctx, contractsToCompile); err != nil {
		return fmt.Errorf("entrypoint compilation failed: %w", err)
	}
	return nil
}

// findRepository looks up a repository entry from the parsed L2 configuration
// and adapts it to the [git.Repository] shape expected by [git.Cloner]. The
// boolean is false when the entry is absent so callers can treat the
// repository as optional.
func findRepository(name configs.RepositoryName) (git.Repository, bool) {
	repo, ok := configs.Values.L2.Repositories[name]
	if !ok {
		return git.Repository{}, false
	}
	return git.Repository{
		Name: string(name),
		URL:  repo.URL,
		Ref:  repo.Branch,
	}, true
}

// contractNames flattens a contract-name allowlist into a deterministically
// ordered slice for `forge inspect`. Sorting keeps regenerated artefacts
// stable across runs.
func contractNames(set map[contracts.ContractName]struct{}) []string {
	names := make([]string, 0, len(set))
	for name := range maps.Keys(set) {
		names = append(names, string(name))
	}
	slices.Sort(names)
	return names
}
