package celestia

const (
	defaultProjectName = "celestia-node"
	defaultRuntimeDir  = ".localnet/celestia"
	defaultDataDir     = ".localnet/celestia/data"
	defaultChainID     = "private"

	defaultCeleniumIndexerURL = "https://github.com/ayaz-ssvlabs/celestia-indexer.git"
	defaultCeleniumIndexerRef = "ssv-indexer-a06ad9f"

	defaultCeleniumInterfaceURL = "https://github.com/ayaz-ssvlabs/celenium-interface.git"
	defaultCeleniumInterfaceRef = "ssv-interface-05162fe"

	defaultCelestiaAppImage  = "ghcr.io/celestiaorg/celestia-app:latest"
	defaultCelestiaNodeImage = "ghcr.io/celestiaorg/celestia-node:latest"
	defaultOpAltDAImage      = "ghcr.io/celestiaorg/op-alt-da:0.12.0"
	defaultCeleniumDBImage   = "timescale/timescaledb-ha:pg15.8-ts2.17.0-all"

	defaultCeleniumIndexerStartHeight = 1
	defaultCeleniumEnabled            = true
	defaultAttachToL2Network          = true

	defaultNamespace = "0000000000000000000000000000000000000000010203040506070809"
)
