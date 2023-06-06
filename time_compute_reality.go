package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/woodywood117/stopwatch"
)

func GetRealityTime() {
	numGetReality := 10

	upBoundConflicts := []int{10000, 20000, 40000}
	probabilityConflict := 0.1
	numTransactionsStart := 5000000
	globalSeed = 0
	fmt.Println("**************************************")
	fmt.Println("Parameter of testing getReality:")
	fmt.Println("upBoundConflicts = ", upBoundConflicts)
	fmt.Println("probabilityConflict = ", probabilityConflict)
	fmt.Println("**************************************")
	file, _ := os.Create("getRealityTime.txt")
	fmt.Fprintln(file, len(upBoundConflicts))
	getRealityStopwatch := stopwatch.New()
	getLedgerStopwatch := stopwatch.New()
	for j := range upBoundConflicts {
		numConflict := upBoundConflicts[j]
		fmt.Fprintln(file, numConflict, numGetReality)
		for i := 0; i < numGetReality; i++ {
			getLedgerStopwatch.Start()
			GrowingLedgerLimitConflict(probabilityConflict, numTransactionsStart, numConflict)
			getLedgerStopwatch.Pause()
			//DrawDAG("1")
			getRealityStopwatch.Start()
			reality := GetReality()
			fmt.Fprintln(file, getRealityStopwatch.Elapsed().Seconds(), "\t", len(reality))
			fmt.Println(len(ledgerMap), numConflicts, len(reality), "Reality = ", getRealityStopwatch.Elapsed().Seconds(), "Ledger = ", getLedgerStopwatch.Elapsed().Seconds())
			getRealityStopwatch.Reset()
			getLedgerStopwatch.Reset()
			CleaningStructures()
		}
	}
}

func GrowingLedgerLimitConflict(probabilityConflict float64, numTransactionsStart int, upBoundConflicts int) {
	exploredSearchLedger = make([]int, numTransactionsStart+1)
	exploredNestedSearchLedger = make([]int, numTransactionsStart+1)
	// Initialization of data structures
	for i := 0; i < len(exploredSearchLedger); i++ {
		exploredSearchLedger[i] = 0
		exploredNestedSearchLedger[i] = 0
	}

	// Create stopwatches
	createLedgerStopwatch := stopwatch.New()
	createInputOutputStopwatch := stopwatch.New()
	//getBranchStopwatch := stopwatch.New()
	// numAnalytics how many times we measure analytics

	// randomize seed
	if globalSeed != 0 {
		rand.Seed(globalSeed)
	} else {
		rand.Seed(time.Now().UnixNano())
	}

	fmt.Println("start")

	// Genesis creation
	createLedgerStopwatch.Start()
	genesis := CreateGenesis(numOutputsGenesis)
	ledgerMap[idGenesis] = &genesis
	createLedgerStopwatch.Pause()
	// Create numTransactionsStart random transactions
	for idTransaction := idGenesis + 1; idTransaction <= numTransactionsStart; idTransaction++ {

		// Initialize input and output maps
		curInputLabels := make(map[string]int)
		curOutputLabels := make(map[string]int)
		//GenerateLabels(curInputLabels, curOutputLabels, probabilityConflict)
		createInputOutputStopwatch.Start()
		GenerateLabels(curInputLabels, curOutputLabels, probabilityConflict)
		createInputOutputStopwatch.Pause()
		// create transaction
		createLedgerStopwatch.Start()
		newLedgerNode := AddTransactionLedger(idTransaction, curInputLabels, curOutputLabels)
		createLedgerStopwatch.Pause()
		// add to the global Ledger DAG slice
		ledgerMap[idTransaction] = &newLedgerNode

		if numConflicts > upBoundConflicts {
			fmt.Println("Create labels time ", createInputOutputStopwatch.Elapsed().Seconds())
			fmt.Println("Add transaction time ", createLedgerStopwatch.Elapsed().Seconds())
			return
		}
	}
}
