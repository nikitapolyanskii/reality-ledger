package main

import "testing"

func TestTimeComputeReality(t *testing.T) {
	GetInputOutputDistribution()
	GetRealityTime()
}

func TestTimeConfirmedTransactionLimitConflict(t *testing.T) {
	GetInputOutputDistribution()
	GetLedgerLimitConflictTime()
}

func TestTimeLedgerGrow(t *testing.T) {
	GetInputOutputDistribution()
	GetLedgerLimitTransactionTime()
}

func TestDrawLedger(t *testing.T) {
	GetInputOutputDistribution()
	TestDrawLedgerInPhases()
}
