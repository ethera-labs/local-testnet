package l2

import (
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
	Long:  "Compiles Solidity contracts for L2 deployment and generates contracts.json with ABIs and bytecodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("running contract compilation command")
		ctx := cmd.Context()
		rootDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		repositoryName := configs.RepositoryNameEtheraContracts
		slog.With("name", repositoryName).Info("fetching repository settings from configuration before cloning")
		var (
			contractsRepo git.Repository
			found         bool
		)
		for name, repo := range configs.Values.L2.Repositories {
			if name == repositoryName {
				contractsRepo = git.Repository{
					Name: string(name),
					URL:  repo.URL,
					Ref:  repo.Branch,
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("could not find: '%s' repository in the configuration", repositoryName)
		}

		slog.Info("cloning repositories")
		cloner := git.NewCloner()
		servicesDir := filepath.Join(rootDir, localnetDirName, servicesDirName)
		if err := cloner.Clone(ctx, servicesDir, contractsRepo); err != nil {
			return fmt.Errorf("failed to clone repository: '%w'", err)
		}

		compiler := contracts.NewCompiler(
			filepath.Join(servicesDir, "ethera-contracts", "L2"),
			filepath.Join(rootDir, localnetDirName, compiledContractsDirName),
		)

		var contractToCompile []string
		for _, contractName := range slices.Collect(maps.Keys(contracts.Contracts)) {
			contractToCompile = append(contractToCompile, string(contractName))
		}

		slog.Info("starting contract compilation", "contracts", contractToCompile)
		if err := compiler.Compile(ctx, contractToCompile); err != nil {
			return fmt.Errorf("contract compilation failed: %w", err)
		}

		slog.Info("contract compilation completed successfully")

		return nil
	},
}
