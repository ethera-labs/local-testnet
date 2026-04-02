package configs

import (
	"errors"
	"fmt"
	"strings"
)

var Values Config

type (
	RepositoryName string
	L2ChainName    string
	ImageName      string

	Config struct {
		L1            L1            `mapstructure:"l1"`
		L2            L2            `mapstructure:"l2"`
		Observability Observability `mapstructure:"observability"`
	}

	L1 struct {
	}

	L2 struct {
		L1ChainID             int                           `mapstructure:"l1-chain-id"`
		L1ElURL               string                        `mapstructure:"l1-el-url"`
		L1ClURL               string                        `mapstructure:"l1-cl-url"`
		EtheraNetworkName     string                        `mapstructure:"ethera-labs-name"`
		Wallet                Wallet                        `mapstructure:"wallet"`
		CoordinatorPrivateKey string                        `mapstructure:"coordinator-private-key"`
		Repositories          map[RepositoryName]Repository `mapstructure:"repositories"`
		ChainConfigs          map[L2ChainName]Chain         `mapstructure:"chain-configs"`
		Images                map[ImageName]Image           `mapstructure:"images"`
		DeploymentTarget      string                        `mapstructure:"deployment-target"`
		GenesisBalanceWei     string                        `mapstructure:"genesis-balance-wei"`
		Dispute               DisputeConfig                 `mapstructure:"dispute"`
		Blockscout            BlockscoutConfig              `mapstructure:"blockscout"`
		Flashblocks           FlashblocksConfig             `mapstructure:"flashblocks"`
		Sidecar               SidecarConfig                 `mapstructure:"sidecar"`
		Frontend              FrontendConfig                `mapstructure:"frontend"`
	}

	FrontendConfig struct {
		Enabled bool `mapstructure:"enabled"`
		Port    int  `mapstructure:"port"`
	}

	BlockscoutConfig struct {
		Enabled bool `mapstructure:"enabled"`
	}

	FlashblocksConfig struct {
		Enabled             bool   `mapstructure:"enabled"`
		OpRbuilderImageTag  string `mapstructure:"op-rbuilder-image-tag"`
		RollupBoostImageTag string `mapstructure:"rollup-boost-image-tag"`
		RollupARPCPort      int    `mapstructure:"rollup-a-rpc-port"`
		RollupBRPCPort      int    `mapstructure:"rollup-b-rpc-port"`
	}

	SidecarConfig struct {
		Enabled        bool `mapstructure:"enabled"`
		RollupAAPIPort int  `mapstructure:"rollup-a-api-port"`
		RollupBAPIPort int  `mapstructure:"rollup-b-api-port"`
	}

	DisputeConfig struct {
		NetworkName                     string `mapstructure:"network-name"`
		ExplorerURL                     string `mapstructure:"explorer-url"`
		ExplorerAPIURL                  string `mapstructure:"explorer-api-url"`
		VerifierAddress                 string `mapstructure:"verifier-address"`
		OwnerAddress                    string `mapstructure:"owner-address"`
		ProposerAddress                 string `mapstructure:"proposer-address"`
		AggregationVkey                 string `mapstructure:"aggregation-vkey"`
		GuardianAddress                 string `mapstructure:"guardian-address"`
		ProofMaturityDelaySeconds       int    `mapstructure:"proof-maturity-delay-seconds"`
		DisputeGameFinalityDelaySeconds int    `mapstructure:"dispute-game-finality-delay-seconds"`
		DisputeGameInitBond             string `mapstructure:"dispute-game-init-bond"`
	}

	Chain struct {
		ID      int `mapstructure:"id"`
		RPCPort int `mapstructure:"rpc-port"`
	}

	Repository struct {
		URL       string `mapstructure:"url"`
		Branch    string `mapstructure:"branch"`
		LocalPath string `mapstructure:"local-path"`
	}

	Image struct {
		Tag string `mapstructure:"tag"`
	}

	Wallet struct {
		PrivateKey string `mapstructure:"private-key"`
		Address    string `mapstructure:"address"`
	}

	Observability struct {
	}
)

const (
	RepositoryNameOpGeth          RepositoryName = "op-geth"
	RepositoryNameOpRbuilder      RepositoryName = "op-rbuilder"
	RepositoryNamePublisher       RepositoryName = "publisher"
	RepositoryNameSidecar         RepositoryName = "sidecar"
	RepositoryNameEtheraContracts RepositoryName = "ethera-contracts"

	ImageNameOpDeployer ImageName = "op-deployer"
	ImageNameOpNode     ImageName = "op-node"
	ImageNameOpProposer ImageName = "op-proposer"
	ImageNameOpBatcher  ImageName = "op-batcher"

	L2ChainNameRollupA L2ChainName = "rollup-a"
	L2ChainNameRollupB L2ChainName = "rollup-b"
)

func (c *L2) Validate() error {
	var errs []error

	if c.L1ChainID == 0 {
		errs = append(errs, errors.New("l2.l1-chain-id is required"))
	}
	if c.L1ElURL == "" {
		errs = append(errs, errors.New("l2.l1-el-url is required"))
	}
	if c.L1ClURL == "" {
		errs = append(errs, errors.New("l2.l1-cl-url is required"))
	}
	if c.CoordinatorPrivateKey == "" {
		errs = append(errs, errors.New("l2.coordinator-private-key is required"))
	}
	if c.Wallet.PrivateKey == "" {
		errs = append(errs, errors.New("l2.wallet.private-key is required"))
	}
	if c.Wallet.Address == "" {
		errs = append(errs, errors.New("l2.wallet.address is required"))
	}
	if normalizedCoordinatorKey, normalizedWalletKey := normalizePrivateKey(c.CoordinatorPrivateKey), normalizePrivateKey(c.Wallet.PrivateKey); normalizedCoordinatorKey != "" && normalizedCoordinatorKey == normalizedWalletKey {
		errs = append(errs, errors.New("l2.coordinator-private-key must differ from l2.wallet.private-key to avoid nonce collisions"))
	}

	requiredRepos := []RepositoryName{
		RepositoryNameOpGeth,
		RepositoryNamePublisher,
		RepositoryNameEtheraContracts,
	}
	if c.Flashblocks.Enabled {
		requiredRepos = append(requiredRepos, RepositoryNameOpRbuilder)
	}
	if c.Sidecar.Enabled {
		requiredRepos = append(requiredRepos, RepositoryNameSidecar)
	}

	for _, name := range requiredRepos {
		repo, exists := c.Repositories[name]
		if !exists {
			errs = append(errs, fmt.Errorf("l2.repositories.%s is required", name))
			continue
		}
		if err := validateRepositoryConfig(name, repo); err != nil {
			errs = append(errs, err)
		}
	}

	requiredImages := []ImageName{ImageNameOpDeployer, ImageNameOpNode, ImageNameOpProposer, ImageNameOpBatcher}
	for _, name := range requiredImages {
		img, exists := c.Images[name]
		if !exists {
			errs = append(errs, fmt.Errorf("l2.images.%s is required", name))
			continue
		}
		if img.Tag == "" {
			errs = append(errs, fmt.Errorf("l2.images.%s.tag is required", name))
		}
	}

	rollupA, hasRollupA := c.ChainConfigs[L2ChainNameRollupA]
	rollupB, hasRollupB := c.ChainConfigs[L2ChainNameRollupB]

	if !hasRollupA {
		errs = append(errs, errors.New("l2.chain-configs.rollup-a is required"))
	} else {
		if rollupA.ID == 0 {
			errs = append(errs, errors.New("l2.chain-configs.rollup-a.id is required"))
		}
		if rollupA.RPCPort == 0 {
			errs = append(errs, errors.New("l2.chain-configs.rollup-a.rpc-port is required"))
		}
	}

	if !hasRollupB {
		errs = append(errs, errors.New("l2.chain-configs.rollup-b is required"))
	} else {
		if rollupB.ID == 0 {
			errs = append(errs, errors.New("l2.chain-configs.rollup-b.id is required"))
		}
		if rollupB.RPCPort == 0 {
			errs = append(errs, errors.New("l2.chain-configs.rollup-b.rpc-port is required"))
		}
	}

	if c.DeploymentTarget == "" {
		errs = append(errs, errors.New("l2.deployment-target is required"))
	} else if c.DeploymentTarget != "live" && c.DeploymentTarget != "calldata" {
		errs = append(errs, errors.New("l2.deployment-target must be either 'live' or 'calldata'"))
	}

	// Validate dispute config
	if c.Dispute.NetworkName == "" {
		errs = append(errs, errors.New("l2.dispute.network-name is required"))
	}
	if c.Dispute.VerifierAddress == "" {
		errs = append(errs, errors.New("l2.dispute.verifier-address is required"))
	}
	if c.Dispute.OwnerAddress == "" {
		errs = append(errs, errors.New("l2.dispute.owner-address is required"))
	}
	if c.Dispute.ProposerAddress == "" {
		errs = append(errs, errors.New("l2.dispute.proposer-address is required"))
	}
	if c.Dispute.AggregationVkey == "" {
		errs = append(errs, errors.New("l2.dispute.aggregation-vkey is required"))
	}
	if c.Dispute.GuardianAddress == "" {
		errs = append(errs, errors.New("l2.dispute.guardian-address is required"))
	}
	if c.Dispute.ProofMaturityDelaySeconds <= 0 {
		errs = append(errs, errors.New("l2.dispute.proof-maturity-delay-seconds must be positive"))
	}
	if c.Dispute.DisputeGameFinalityDelaySeconds <= 0 {
		errs = append(errs, errors.New("l2.dispute.dispute-game-finality-delay-seconds must be positive"))
	}
	if c.Dispute.DisputeGameInitBond == "" {
		errs = append(errs, errors.New("l2.dispute.dispute-game-init-bond is required"))
	}

	if c.EtheraNetworkName == "" {
		errs = append(errs, errors.New("l2.ethera-labs-name is required"))
	}
	if c.Sidecar.Enabled && !c.Flashblocks.Enabled {
		errs = append(errs, errors.New("l2.sidecar.enabled requires l2.flashblocks.enabled"))
	}
	if c.Frontend.Enabled && (!c.Flashblocks.Enabled || !c.Sidecar.Enabled) {
		errs = append(errs, errors.New("l2.frontend.enabled requires l2.flashblocks.enabled and l2.sidecar.enabled"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("L2 configuration validation failed: %w", errors.Join(errs...))
	}

	return nil
}

func normalizePrivateKey(key string) string {
	trimmed := strings.TrimSpace(key)
	trimmed = strings.ToLower(trimmed)
	return strings.TrimPrefix(trimmed, "0x")
}

func validateRepositoryConfig(name RepositoryName, repo Repository) error {
	hasLocal := repo.LocalPath != ""
	hasRemote := repo.URL != "" && repo.Branch != ""

	if !hasLocal && !hasRemote {
		return fmt.Errorf("l2.repositories.%s must set either local-path or url+branch", name)
	}
	if hasLocal && hasRemote {
		return fmt.Errorf("l2.repositories.%s cannot set both local-path and url+branch (choose one)", name)
	}

	return nil
}
