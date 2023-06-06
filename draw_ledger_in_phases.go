package main

func TestDrawLedgerInPhases() {
	probabilityConflict := 0.4
	numOutputsGenesis = 4
	numTransactions := 16
	computeBranch := false
	GrowingLedgerLimit(probabilityConflict, numTransactions, computeBranch)
	DrawDAG("ledger_start")
	GetReality()
	DrawDAG("ledger_after_reality")
	assignWeightsAfterReality()
	DeleteRejectedTransactions()
	DrawDAG("ledger_after_pruning")
}
