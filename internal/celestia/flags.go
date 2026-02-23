package celestia

import "github.com/spf13/viper"

func init() {
	declareStringFlag("project-name", "celestia.project-name", defaultCelestiaConfig.ProjectName, "Docker Compose project name for Celestia services")
	declareStringFlag("runtime-dir", "celestia.runtime-dir", defaultCelestiaConfig.RuntimeDir, "Runtime directory for generated Celestia compose assets")
	declareStringFlag("data-dir", "celestia.data-dir", defaultCelestiaConfig.DataDir, "Data directory for Celestia chain/node/database state")
	declareStringFlag("chain-id", "celestia.chain-id", defaultCelestiaConfig.ChainID, "Celestia local chain ID / p2p network")

	declareBoolFlag("attach-to-l2-network", "celestia.attach-to-l2-network", defaultCelestiaConfig.AttachToL2Network, "Attach op-alt-da container to localnet-l2 network when available")
	declareBoolFlag("celenium-enabled", "celestia.celenium-enabled", defaultCelestiaConfig.CeleniumEnabled, "Enable Celenium services (db/indexer/api/interface)")

	declareIntFlag("celenium-indexer-start-height", "celestia.celenium-indexer-start-height", int(defaultCelestiaConfig.CeleniumIndexerStartHeight), "Initial start height for Celenium indexer")

	declareStringFlag("celenium-indexer-url", "celestia.celenium-indexer.url", defaultCelestiaConfig.CeleniumIndexer.URL, "Repository URL for celenium-indexer")
	declareStringFlag("celenium-indexer-branch", "celestia.celenium-indexer.branch", defaultCelestiaConfig.CeleniumIndexer.Branch, "Repository branch/tag/ref for celenium-indexer")
	declareStringFlag("celenium-indexer-local-path", "celestia.celenium-indexer.local-path", "", "Local path override for celenium-indexer repository")

	declareStringFlag("celenium-interface-url", "celestia.celenium-interface.url", defaultCelestiaConfig.CeleniumInterface.URL, "Repository URL for celenium-interface")
	declareStringFlag("celenium-interface-branch", "celestia.celenium-interface.branch", defaultCelestiaConfig.CeleniumInterface.Branch, "Repository branch/tag/ref for celenium-interface")
	declareStringFlag("celenium-interface-local-path", "celestia.celenium-interface.local-path", "", "Local path override for celenium-interface repository")

	declareStringFlag("celestia-app-image", "celestia.images.celestia-app", defaultCelestiaConfig.Images.CelestiaApp, "Celestia app image")
	declareStringFlag("celestia-node-image", "celestia.images.celestia-node", defaultCelestiaConfig.Images.CelestiaNode, "Celestia bridge node image")
	declareStringFlag("op-alt-da-image", "celestia.images.op-alt-da", defaultCelestiaConfig.Images.OpAltDA, "op-alt-da image")
	declareStringFlag("celenium-db-image", "celestia.images.celenium-db", defaultCelestiaConfig.Images.CeleniumDB, "Celenium database image")
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
