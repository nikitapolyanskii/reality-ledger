package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/woodywood117/stopwatch"
)

func GetLedgerLimitTransactionTime() {
	probabilityConflict := []float64{0.05, 0.05, 0.1, 0.5}
	numTransactionsStart := 2000000
	computeBranch := []bool{false, true}
	// Test 1: ever growing ledger
	file, _ := os.Create("ledgerGrow.txt")
	fmt.Fprintln(file, len(probabilityConflict))
	defer file.Close()

	for i := range probabilityConflict {
		for j := range computeBranch {
			GrowingLedgerLimit(probabilityConflict[i], numTransactionsStart, computeBranch[j])
			//reality := GetReality()
			//reality[0] = 4
			fmt.Fprintln(file, probabilityConflict[i], "\t", computeBranch[j], "\t", len(allAnalytics))
			for s := range allAnalytics {
				fmt.Fprintln(file, allAnalytics[s].timestamp, "\t", allAnalytics[s].numTransactions, "\t", allAnalytics[s].numConflicts)
			}
			allAnalytics = nil
			CleaningStructures()
		}

	}
}

func GrowingLedgerLimit(probabilityConflict float64, numTransactionsLimit int, doBranch bool) {
	exploredSearchLedger = make([]int, numTransactionsLimit+1)
	exploredNestedSearchLedger = make([]int, numTransactionsLimit+1)
	// Initialization of data structures
	for i := 0; i < len(exploredSearchLedger); i++ {
		exploredSearchLedger[i] = 0
		exploredNestedSearchLedger[i] = 0
	}

	// Create stopwatches
	createLedgerStopwatch := stopwatch.New()
	//getBranchStopwatch := stopwatch.New()
	// numAnalytics how many times we measure analytics
	numAnalytics := 0

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
	Analytics(createLedgerStopwatch.Elapsed())
	// Create numTransactionsLimit random transactions
	for idTransaction := idGenesis + 1; idTransaction <= numTransactionsLimit; idTransaction++ {
		// Measure parameters
		if createLedgerStopwatch.Elapsed() > time.Duration(numAnalytics)*measureEverySec {
			numAnalytics = numAnalytics + 1
			Analytics(createLedgerStopwatch.Elapsed())

			fmt.Println(allAnalytics[len(allAnalytics)-1])

			//			TraverseAndCheck("newTransaction" + strconv.Itoa(idTransaction))
			//fmt.Println("Create time = ", createLedgerStopwatch.Elapsed().Seconds())
		}
		// Initialize input and output maps
		curInputLabels := make(map[string]int)
		curOutputLabels := make(map[string]int)
		GenerateLabels(curInputLabels, curOutputLabels, probabilityConflict)

		// create transaction
		createLedgerStopwatch.Start()
		newLedgerNode := AddTransactionLedger(idTransaction, curInputLabels, curOutputLabels)

		// add to the global Ledger DAG slice
		ledgerMap[idTransaction] = &newLedgerNode
		if doBranch {
			_ = GetBranch(idTransaction)
		}
		createLedgerStopwatch.Pause()

	}
	Analytics(createLedgerStopwatch.Elapsed())
}
