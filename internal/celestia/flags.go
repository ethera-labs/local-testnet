package celestia

import "github.com/spf13/viper"

func init() {
	declareStringFlag("project-name", "celestia.project-name", defaultProjectName, "Docker Compose project name for Celestia services")
	declareStringFlag("runtime-dir", "celestia.runtime-dir", defaultRuntimeDir, "Runtime directory for generated Celestia compose assets")
	declareStringFlag("data-dir", "celestia.data-dir", defaultDataDir, "Data directory for Celestia chain/node/database state")
	declareStringFlag("chain-id", "celestia.chain-id", defaultChainID, "Celestia local chain ID / p2p network")

	declareBoolFlag("attach-to-l2-network", "celestia.attach-to-l2-network", defaultAttachToL2Network, "Attach op-alt-da container to localnet-l2 network when available")
	declareBoolFlag("celenium-enabled", "celestia.celenium-enabled", defaultCeleniumEnabled, "Enable Celenium services (db/indexer/api/interface)")

	declareIntFlag("celenium-indexer-start-height", "celestia.celenium-indexer-start-height", defaultCeleniumIndexerStartHeight, "Initial start height for Celenium indexer")

	declareStringFlag("celenium-indexer-url", "celestia.celenium-indexer.url", defaultCeleniumIndexerURL, "Repository URL for celenium-indexer")
	declareStringFlag("celenium-indexer-branch", "celestia.celenium-indexer.branch", defaultCeleniumIndexerRef, "Repository branch/tag/ref for celenium-indexer")
	declareStringFlag("celenium-indexer-local-path", "celestia.celenium-indexer.local-path", "", "Local path override for celenium-indexer repository")

	declareStringFlag("celenium-interface-url", "celestia.celenium-interface.url", defaultCeleniumInterfaceURL, "Repository URL for celenium-interface")
	declareStringFlag("celenium-interface-branch", "celestia.celenium-interface.branch", defaultCeleniumInterfaceRef, "Repository branch/tag/ref for celenium-interface")
	declareStringFlag("celenium-interface-local-path", "celestia.celenium-interface.local-path", "", "Local path override for celenium-interface repository")

	declareStringFlag("celestia-app-image", "celestia.images.celestia-app", defaultCelestiaAppImage, "Celestia app image")
	declareStringFlag("celestia-node-image", "celestia.images.celestia-node", defaultCelestiaNodeImage, "Celestia bridge node image")
	declareStringFlag("op-alt-da-image", "celestia.images.op-alt-da", defaultOpAltDAImage, "op-alt-da image")
	declareStringFlag("celenium-db-image", "celestia.images.celenium-db", defaultCeleniumDBImage, "Celenium database image")
}

func declareStringFlag(name, key, defaultValue, description string) {
	CMD.Flags().String(name, defaultValue, description)
	if err := viper.BindPFlag(key, CMD.Flags().Lookup(name)); err != nil {
		panic(err)
	}
}

func declareBoolFlag(name, key string, defaultValue bool, description string) {
	CMD.Flags().Bool(name, defaultValue, description)
	if err := viper.BindPFlag(key, CMD.Flags().Lookup(name)); err != nil {
		panic(err)
	}
}

func declareIntFlag(name, key string, defaultValue int, description string) {
	CMD.Flags().Int(name, defaultValue, description)
	if err := viper.BindPFlag(key, CMD.Flags().Lookup(name)); err != nil {
		panic(err)
	}
}
