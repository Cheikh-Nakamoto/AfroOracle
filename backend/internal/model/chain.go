package model

// chainPublish currently returns a deterministic simulated hash. Credentials
// alone are not evidence that a transaction was sent or included on Polygon;
// the label stays simulated until the real publish call waits for a receipt.
func chainPublish(result ConsensusResult, report any) (tx, chain string) {
	return reportHash(report), "simulated"
}

// ChainPublish is the controller-facing entry point.
func ChainPublish(result ConsensusResult, report any) (tx, chain string) {
	return chainPublish(result, report)
}
