package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/woodywood117/stopwatch"
)

func GetLedgerLimitConflictTime() {
	probabilityConflict := 0.01
	numTransactions := 400000
	upBoundConflicts := 5000
	fmt.Println("**************************************")
	fmt.Println("Parameter of testing getLedgerLimitConflict:")
	fmt.Println("numTransactions = ", numTransactions)
	fmt.Println("upBoundConflicts = ", upBoundConflicts)
	fmt.Println("probabilityConflict = ", probabilityConflict)
	fmt.Println("**************************************")
	GrowingLedgerPruningConflictsLimit(probabilityConflict, numTransactions, upBoundConflicts)
	CleaningStructures()
}

func GrowingLedgerPruningConflictsLimit(probabilityConflict float64, numTransactions int, upBoundConflicts int) {
	numConfirmedTransactions := 0
	exploredSearchLedger = make([]int, numTransactions+1)
	exploredNestedSearchLedger = make([]int, numTransactions+1)
	// Initialization of data structures
	for i := 0; i < len(exploredSearchLedger); i++ {
		exploredSearchLedger[i] = 0
		exploredNestedSearchLedger[i] = 0
	}

	// Create stopwatches
	createLedgerStopwatch := stopwatch.New()
	pruneLedgerStopwatch := stopwatch.New()
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
	//outLedgerPruneLimitConflictWithTimer(createLedgerStopwatch)
	file, _ := os.Create("ledgerGrowAndPrune.txt")
	defer file.Close()
	fmt.Fprintln(file, "0", len(ledgerMap), numConflicts, numConfirmedTransactions)

	// Genesis creation
	createLedgerStopwatch.Start()
	genesis := CreateGenesis(numOutputsGenesis)
	ledgerMap[idGenesis] = &genesis
	createLedgerStopwatch.Pause()
	Analytics(createLedgerStopwatch.Elapsed())
	// Create numTransactions random transactions
	for idTransaction := idGenesis + 1; idTransaction <= numTransactions; idTransaction++ {
		// Measure parameters
		if createLedgerStopwatch.Elapsed() > time.Duration(numAnalytics)*measureEverySec {
			numAnalytics = int(createLedgerStopwatch.Elapsed()/measureEverySec) + 1
			Analytics(createLedgerStopwatch.Elapsed())
			fmt.Fprintln(file, createLedgerStopwatch.Elapsed().Seconds(), len(ledgerMap), numConflicts, numConfirmedTransactions)
			fmt.Println(createLedgerStopwatch.Elapsed().Seconds(), len(ledgerMap), numConflicts, numConfirmedTransactions)
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
		createLedgerStopwatch.Pause()
		//TraverseAndCheck("newTransaction" + strconv.Itoa(idTransaction))
		if numConflicts > upBoundConflicts {
			fmt.Fprintln(file, createLedgerStopwatch.Elapsed().Seconds(), len(ledgerMap), numConflicts, numConfirmedTransactions)
			createLedgerStopwatch.Start()
			pruneLedgerStopwatch.Start()
			//DrawDAG("0")
			curReality := GetReality()
			if len(curReality) < 1 {
				fmt.Println("error")
			}
			fmt.Println("Get Reality ", pruneLedgerStopwatch.Elapsed().Seconds())
			pruneLedgerStopwatch.Reset()
			pruneLedgerStopwatch.Start()
			assignWeightsAfterReality()
			fmt.Println("Assign Weights ", pruneLedgerStopwatch.Elapsed().Seconds())
			pruneLedgerStopwatch.Reset()
			pruneLedgerStopwatch.Start()
			//AssignWeightTransaction(idTransaction)
			//DrawDAG("1")
			//TraverseAndCheck("assign weights")
			DeleteRejectedTransactions()
			fmt.Println("Delete rejected transactions ", pruneLedgerStopwatch.Elapsed().Seconds())
			pruneLedgerStopwatch.Reset()
			numConfirmedTransactions = numConfirmedTransactions + len(ledgerMap) - 1

			//TraverseAndCheck("delete rejected transactions")
			fmt.Println("Number of unspent outputs =", len(unspentLabelsSlice))
			//DrawDAG("2")
			createLedgerStopwatch.Pause()
			numOutputsGenesis = len(unspentLabelsSlice)
			CleaningStructures()
			genesis := CreateGenesis(numOutputsGenesis)
			fmt.Println("unspentOutputs = ", numOutputsGenesis)
			ledgerMap[idGenesis] = &genesis
			//fmt.Fprintln(file, createLedgerStopwatch.Elapsed().Seconds(), len(ledgerMap), numConflicts, numConfirmedTransactions)
		}
	}
	Analytics(createLedgerStopwatch.Elapsed())
}
