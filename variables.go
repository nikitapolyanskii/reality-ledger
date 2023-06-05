package main

import (
	"time"
)

// main global parameters of DAG

var globalSeed int64 = 100

// Analytics
var measureEverySec = time.Duration(100) * time.Millisecond
var allAnalytics []AllParameters

var idGenesis = 1

// main global variables
var ledgerMap = map[int]*Transaction{}                    // all transactions and parent-child dependencies are stored there. to modify structure Transaction use pointers
var outputLabelsSliceOwnerID []StringInt                  // array that stores pairs outputLabel + id of transaction; Used for taking random UTXOs
var unspentLabelsSlice []StringInt                        // array that stores pairs unspent outputLabel + id of transaction; Used for taking random UTXOs
var unconfirmedSpentLabelsSlice []StringInt               // array that stores pairs spent existing outputLabel + id of transaction; Used for taking random UTXOs // This are still unconfirmed
var confirmedSpentlabelsSlice []StringInt                 // array that stores pairs spent outputLabel + id of transaction // This are already confirmed
var outputLabelsMapOwnerID = map[string]*OutputInfo{}     // map that contains UTXO label and id of transaction created whis UTXO
var outputLabelsMapConsumerIDs = map[string]map[int]int{} // map that contains UTXO and map of all tx ids consuming this UTXO
var exploredSearchLedger = []int{}                        // map showing explorations when traversing graph
var exploredNestedSearchLedger = []int{}
var confirmedTransactions = map[int]int{} // map containing all confirmed Transactions
var numConflicts = 0                      // number of conflicts

var numBadAttemptsInputLabel = 1
var threshold float64 = 0.66
