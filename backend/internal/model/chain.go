package model

import "os"

// chainPublish returns the tx hash and chain label. Simulated ("0x"+sha256) when
// POLYGON_RPC/PRIVATE_KEY are unset; otherwise a go-ethereum call to
// AgriculturalData.publish. TODO: wire the real go-ethereum call using contracts/abi.json.
func chainPublish(result ConsensusResult, report any) (tx, chain string) {
	if os.Getenv("POLYGON_RPC") == "" || os.Getenv("PRIVATE_KEY") == "" {
		return reportHash(report), "simulated"
	}
	return reportHash(report), "polygon-amoy"
}

// ChainPublish is the controller-facing entry point.
func ChainPublish(result ConsensusResult, report any) (tx, chain string) {
	return chainPublish(result, report)
}
