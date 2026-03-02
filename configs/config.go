package configs

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var Values Config

type (
	RepositoryName string
	L2ChainName    string
	ImageName      string

	Config struct {
		L1            L1            `mapstructure:"l1"`
		L2            L2            `mapstructure:"l2"`
		Celestia      Celestia      `mapstructure:"celestia"`
		Observability Observability `mapstructure:"observability"`
	}

	L1 struct {
	}

	Celestia struct {
		ProjectName                string         `mapstructure:"project-name"`
		RuntimeDir                 string         `mapstructure:"runtime-dir"`
		DataDir                    string         `mapstructure:"data-dir"`
		AttachToL2Network          bool           `mapstructure:"attach-to-l2-network"`
		ChainID                    string         `mapstructure:"chain-id"`
		CeleniumEnabled            bool           `mapstructure:"celenium-enabled"`
		CeleniumIndexerStartHeight uint64         `mapstructure:"celenium-indexer-start-height"`
		CeleniumIndexer            Repository     `mapstructure:"celenium-indexer"`
		CeleniumInterface          Repository     `mapstructure:"celenium-interface"`
		Images                     CelestiaImages `mapstructure:"images"`
	}

	CelestiaImages struct {
		CelestiaApp  string `mapstructure:"celestia-app"`
		CelestiaNode string `mapstructure:"celestia-node"`
		OpAltDA      string `mapstructure:"op-alt-da"`
		CeleniumDB   string `mapstructure:"celenium-db"`
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
		OpSuccinct            OpSuccinctConfig              `mapstructure:"op-succinct"`
		AltDA                 AltDAConfig                   `mapstructure:"alt-da"`
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

	OpSuccinctConfig struct {
		RollupA OpSuccinctInstanceConfig `mapstructure:"rollup-a"`
		RollupB OpSuccinctInstanceConfig `mapstructure:"rollup-b"`
	}

	OpSuccinctInstanceConfig struct {
		Enabled *bool `mapstructure:"enabled"`
	}

	AltDAConfig struct {
		Enabled                    bool   `mapstructure:"enabled"`
		Provider                   string `mapstructure:"provider"`
		DAServer                   string `mapstructure:"da-server"`
		VerifyOnRead               bool   `mapstructure:"verify-on-read"`
		DAService                  bool   `mapstructure:"da-service"`
		PutTimeout                 string `mapstructure:"put-timeout"`
		GetTimeout                 string `mapstructure:"get-timeout"`
		MaxConcurrentDARequests    uint64 `mapstructure:"max-concurrent-da-requests"`
		DACommitmentType           string `mapstructure:"da-commitment-type"`
		DAChallengeWindow          uint64 `mapstructure:"da-challenge-window"`
		DAResolveWindow            uint64 `mapstructure:"da-resolve-window"`
		DABondSize                 uint64 `mapstructure:"da-bond-size"`
		DAResolverRefundPercentage uint64 `mapstructure:"da-resolver-refund-percentage"`
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
		ID       int    `mapstructure:"id"`
		RPCPort  int    `mapstructure:"rpc-port"`
		L1Sender Wallet `mapstructure:"l1-sender"`
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
	RepositoryNamePublisher       RepositoryName = "publisher"
	RepositoryNameEtheraContracts RepositoryName = "ethera-contracts"
	RepositoryNameOpSuccinct      RepositoryName = "op-succinct"

	ImageNameOpDeployer ImageName = "op-deployer"
	ImageNameOpNode     ImageName = "op-node"
	ImageNameOpProposer ImageName = "op-proposer"
	ImageNameOpBatcher  ImageName = "op-batcher"

	L2ChainNameRollupA L2ChainName = "rollup-a"
	L2ChainNameRollupB L2ChainName = "rollup-b"

	AltDACommitmentTypeKeccak  = "KeccakCommitment"
	AltDACommitmentTypeGeneric = "GenericCommitment"

	AltDAProviderCelestia     = "celestia"
	AltDAProviderOpAltDALocal = "op-alt-da-local"
)

func (c L2) IsOpSuccinctChainEnabled(chain L2ChainName) bool {
	defaultEnabled := false
	if repo, exists := c.Repositories[RepositoryNameOpSuccinct]; exists {
		defaultEnabled = isRepositoryConfigured(repo)
	}

	switch chain {
	case L2ChainNameRollupA:
		return isOpSuccinctEnabled(c.OpSuccinct.RollupA.Enabled, defaultEnabled)
	case L2ChainNameRollupB:
		return isOpSuccinctEnabled(c.OpSuccinct.RollupB.Enabled, defaultEnabled)
	default:
		return false
	}
}

func (c L2) EnabledOpSuccinctChains() []L2ChainName {
	chains := make([]L2ChainName, 0, 2)
	if c.IsOpSuccinctChainEnabled(L2ChainNameRollupA) {
		chains = append(chains, L2ChainNameRollupA)
	}
	if c.IsOpSuccinctChainEnabled(L2ChainNameRollupB) {
		chains = append(chains, L2ChainNameRollupB)
	}
	return chains
}

func (c L2) AnyOpSuccinctChainEnabled() bool {
	return len(c.EnabledOpSuccinctChains()) > 0
}

func (c L2) IsCelestiaAltDAEnabled() bool {
	return c.AltDA.Enabled && c.AltDA.ProviderName() == AltDAProviderCelestia
}

func (c L2) IsLocalOpAltDAEnabled() bool {
	return c.AltDA.Enabled && c.AltDA.ProviderName() == AltDAProviderOpAltDALocal
}

func (c AltDAConfig) ProviderName() string {
	provider := strings.TrimSpace(c.Provider)
	if provider == "" {
		return AltDAProviderCelestia
	}
	return provider
}

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

	requiredRepos := []RepositoryName{RepositoryNameOpGeth, RepositoryNamePublisher}
	for _, name := range requiredRepos {
		repo, exists := c.Repositories[name]
		if !exists {
			errs = append(errs, fmt.Errorf("l2.repositories.%s is required", name))
			continue
		}

		hasLocal := repo.LocalPath != ""
		hasRemote := repo.URL != "" && repo.Branch != ""
		if !hasLocal && !hasRemote {
			errs = append(errs, fmt.Errorf("l2.repositories.%s must set either local-path or url+branch", name))
		}
		if hasLocal && hasRemote {
			errs = append(errs, fmt.Errorf("l2.repositories.%s cannot set both local-path and url+branch (choose one)", name))
		}
	}

	// op-succinct repository is required only when at least one op-succinct chain is enabled.
	if c.AnyOpSuccinctChainEnabled() {
		repo, exists := c.Repositories[RepositoryNameOpSuccinct]
		if !exists || !isRepositoryConfigured(repo) {
			errs = append(errs, fmt.Errorf("l2.repositories.%s must set either local-path or url+branch when any l2.op-succinct.*.enabled is true", RepositoryNameOpSuccinct))
		} else if err := validateRepository(repo, fmt.Sprintf("l2.repositories.%s", RepositoryNameOpSuccinct)); err != nil {
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
		if rollupA.L1Sender.PrivateKey != "" && rollupA.L1Sender.Address == "" {
			errs = append(errs, errors.New("l2.chain-configs.rollup-a.l1-sender.address is required when private-key is set"))
		}
		if rollupA.L1Sender.Address != "" && rollupA.L1Sender.PrivateKey == "" {
			errs = append(errs, errors.New("l2.chain-configs.rollup-a.l1-sender.private-key is required when address is set"))
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
		if rollupB.L1Sender.PrivateKey != "" && rollupB.L1Sender.Address == "" {
			errs = append(errs, errors.New("l2.chain-configs.rollup-b.l1-sender.address is required when private-key is set"))
		}
		if rollupB.L1Sender.Address != "" && rollupB.L1Sender.PrivateKey == "" {
			errs = append(errs, errors.New("l2.chain-configs.rollup-b.l1-sender.private-key is required when address is set"))
		}
	}

	if c.DeploymentTarget == "" {
		errs = append(errs, errors.New("l2.deployment-target is required"))
	} else if c.DeploymentTarget != "live" && c.DeploymentTarget != "calldata" {
		errs = append(errs, errors.New("l2.deployment-target must be either 'live' or 'calldata'"))
	}

	if c.AltDA.Enabled {
		switch c.AltDA.ProviderName() {
		case AltDAProviderCelestia, AltDAProviderOpAltDALocal:
		default:
			errs = append(errs, fmt.Errorf(
				"l2.alt-da.provider must be either %q or %q when l2.alt-da.enabled is true",
				AltDAProviderCelestia,
				AltDAProviderOpAltDALocal,
			))
		}

		if c.AltDA.DAServer == "" {
			errs = append(errs, errors.New("l2.alt-da.da-server is required when l2.alt-da.enabled is true"))
		} else if _, err := url.ParseRequestURI(c.AltDA.DAServer); err != nil {
			errs = append(errs, fmt.Errorf("l2.alt-da.da-server is invalid: %w", err))
		}

		if c.AltDA.DACommitmentType == "" {
			errs = append(errs, errors.New("l2.alt-da.da-commitment-type is required when l2.alt-da.enabled is true"))
		} else if c.AltDA.DACommitmentType != AltDACommitmentTypeKeccak && c.AltDA.DACommitmentType != AltDACommitmentTypeGeneric {
			errs = append(errs, errors.New("l2.alt-da.da-commitment-type must be either 'KeccakCommitment' or 'GenericCommitment'"))
		}

		if c.AltDA.DAChallengeWindow == 0 {
			errs = append(errs, errors.New("l2.alt-da.da-challenge-window must be positive when l2.alt-da.enabled is true"))
		}
		if c.AltDA.DAResolveWindow == 0 {
			errs = append(errs, errors.New("l2.alt-da.da-resolve-window must be positive when l2.alt-da.enabled is true"))
		}
		if c.AltDA.MaxConcurrentDARequests == 0 {
			errs = append(errs, errors.New("l2.alt-da.max-concurrent-da-requests must be positive when l2.alt-da.enabled is true"))
		}

		if c.AltDA.PutTimeout != "" {
			if _, err := time.ParseDuration(c.AltDA.PutTimeout); err != nil {
				errs = append(errs, fmt.Errorf("l2.alt-da.put-timeout is invalid: %w", err))
			}
		}
		if c.AltDA.GetTimeout != "" {
			if _, err := time.ParseDuration(c.AltDA.GetTimeout); err != nil {
				errs = append(errs, fmt.Errorf("l2.alt-da.get-timeout is invalid: %w", err))
			}
		}
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

func (c *Celestia) Validate() error {
	var errs []error

	if c.ProjectName == "" {
		errs = append(errs, errors.New("celestia.project-name is required"))
	}
	if c.RuntimeDir == "" {
		errs = append(errs, errors.New("celestia.runtime-dir is required"))
	}
	if c.DataDir == "" {
		errs = append(errs, errors.New("celestia.data-dir is required"))
	}
	if c.ChainID == "" {
		errs = append(errs, errors.New("celestia.chain-id is required"))
	}
	if c.Images.CelestiaApp == "" {
		errs = append(errs, errors.New("celestia.images.celestia-app is required"))
	}
	if c.Images.CelestiaNode == "" {
		errs = append(errs, errors.New("celestia.images.celestia-node is required"))
	}
	if c.Images.OpAltDA == "" {
		errs = append(errs, errors.New("celestia.images.op-alt-da is required"))
	}

	if c.CeleniumEnabled {
		if c.Images.CeleniumDB == "" {
			errs = append(errs, errors.New("celestia.images.celenium-db is required when celestia.celenium-enabled is true"))
		}
		if c.CeleniumIndexerStartHeight == 0 {
			errs = append(errs, errors.New("celestia.celenium-indexer-start-height must be positive when celestia.celenium-enabled is true"))
		}
		if err := validateRepository(c.CeleniumIndexer, "celestia.celenium-indexer"); err != nil {
			errs = append(errs, err)
		}
		if err := validateRepository(c.CeleniumInterface, "celestia.celenium-interface"); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("Celestia configuration validation failed: %w", errors.Join(errs...))
	}

	return nil
}

func validateRepository(repo Repository, key string) error {
	hasLocal := repo.LocalPath != ""
	hasRemote := repo.URL != "" && repo.Branch != ""
	if !hasLocal && !hasRemote {
		return fmt.Errorf("%s must set either local-path or url+branch", key)
	}
	if hasLocal && hasRemote {
		return fmt.Errorf("%s cannot set both local-path and url+branch (choose one)", key)
	}
	return nil
}

func isRepositoryConfigured(repo Repository) bool {
	return repo.LocalPath != "" || repo.URL != "" || repo.Branch != ""
}

func isOpSuccinctEnabled(enabled *bool, defaultEnabled bool) bool {
	if enabled == nil {
		// Backward compatibility: when unset, inherit from repository configuration.
		return defaultEnabled
	}
	return *enabled
}
