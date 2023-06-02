package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/woodywood117/stopwatch"
	"golang.org/x/exp/maps"
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
var outputLabelsMapOwnerID = map[string]*IntIntIntInt{}   // map that contains UTXO label and id of transaction created whis UTXO
var outputLabelsMapConsumerIDs = map[string]map[int]int{} // map that contains UTXO and map of all tx ids consuming this UTXO
var exploredSearchLedger = []int{}                        // map showing explorations when traversing graph
var exploredNestedSearchLedger = []int{}
var confirmedTransactions = map[int]int{} // map containing all confirmed Transactions
var numConflicts = 0                      // number of conflicts
var inputOutputDistribution []FloatIntInt

// random number of inputs/outputs
var ifUniform = 1
var numInputsMax = 2
var numOutputsMax = 3
var file_distribution = "./distribution_stardust.txt"
var numOutputsGenesis = 16
var numBadAttemptsInputLabel = 1
var threshold float64 = 0.66

// Transaction represents a UTXO-based transaction
type Transaction struct {
	ID                  int
	InputLabels         map[string]int
	OutputLabels        map[string]int
	Parents             map[int]int
	Children            map[int]int
	ChildrenConflicts   map[int]int
	ParentsConflicts    map[int]int
	InputConflictLabels map[string]int //subset of InputLabels that is share with other txs
	IsConflict          bool
	DirectConflicts     map[int]int // tx ids of txs directly conflicting with a given one
	Weight              float64
}

type AllParameters struct {
	timestamp       float64
	numConflicts    int
	numTransactions int
}

type IntIntIntInt struct {
	ID                int // id of transaction creating the output
	indexSlice        int // index of slice outputLabelsSliceOwnerID containing this output
	indexUnspentSlice int // index of slice unspentLabelsSlice containing this output
	indexSpentSlice   int // index of slice unconfirmedSpentLabelsSlice containing this output
}
type StringInt struct {
	ID          int
	OutputLabel string
}

func GetBranch(id int) map[int]int {
	branch := map[int]int{}
	Stack := make([]int, 0)
	if ledgerMap[id].IsConflict {
		branch[id] = 1
	}
	for nextVertex := range ledgerMap[id].ParentsConflicts {
		Stack = append(Stack, nextVertex)
	}
	allVisited := make([]int, 0)

	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != id {
			exploredSearchLedger[curVertex] = id
			branch[curVertex] = 1
			for nextVertex := range ledgerMap[curVertex].ParentsConflicts {
				Stack = append(Stack, nextVertex)
			}
		}
	}
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
	return branch
}

func hash(str string) string {
	sum := sha256.Sum256([]byte(str))
	return fmt.Sprintf("%x", sum)
}

func AssignWeightTransaction(id int) {
	allVisited := make([]int, 0)
	//Missing: Traverse down till conflicts and mark the past conflicts BFS
	Stack := make([]int, 0)
	Stack = append(Stack, id)
	outputLabelsToDelete := map[string]int{}
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != id {
			exploredSearchLedger[curVertex] = id
			ledgerMap[curVertex].Weight = 1
			//delete all inputs from potential spentOutputs
			for curLabel := range ledgerMap[curVertex].InputLabels {
				outputLabelsToDelete[curLabel] = curVertex
			}

			for nextVertex := range ledgerMap[curVertex].Parents {
				Stack = append(Stack, nextVertex)
			}
		}
	}
	numDel := 0
	for curLabel := range outputLabelsToDelete {
		numDel = numDel + 1
		confirmedSpentlabelsSlice = append(confirmedSpentlabelsSlice, CreateOutputID(outputLabelsToDelete[curLabel], curLabel))
		curIndex := outputLabelsMapOwnerID[curLabel].indexSpentSlice
		if curIndex != -1 {
			unconfirmedSpentLabelsSlice[curIndex] = unconfirmedSpentLabelsSlice[len(unconfirmedSpentLabelsSlice)-1]
			movedLabel := unconfirmedSpentLabelsSlice[curIndex].OutputLabel
			unconfirmedSpentLabelsSlice = unconfirmedSpentLabelsSlice[:len(unconfirmedSpentLabelsSlice)-1]
			if curIndex < len(unconfirmedSpentLabelsSlice) {
				outputLabelsMapOwnerID[movedLabel].indexSpentSlice = curIndex
			} else {
				outputLabelsMapOwnerID[movedLabel].indexSpentSlice = -1
			}
			outputLabelsMapOwnerID[curLabel].indexSpentSlice = -1
		}
	}
	//unconfirmedSpentLabelsSlice = unconfirmedSpentLabelsSlice[:len(unconfirmedSpentLabelsSlice)-numDel]

	// clean exploredSearchLedger
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
}

func futureLabelsUntilConflict(newLabel string, Labels map[string]int) {
	Labels[newLabel] = 1
	allVisited := make([]int, 0)
	//Missing: Traverse down till conflicts and mark the past conflicts BFS
	Stack := make([]int, 0)
	id := 1
	for consumerID := range outputLabelsMapConsumerIDs[newLabel] {
		Stack = append(Stack, consumerID)
	}
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != id {
			exploredSearchLedger[curVertex] = id
			if !ledgerMap[curVertex].IsConflict {
				for curLabel := range ledgerMap[curVertex].OutputLabels {
					Labels[curLabel] = 1
				}

				for nextVertex := range ledgerMap[curVertex].Children {
					Stack = append(Stack, nextVertex)
				}
			}
		}
	}
	// clean exploredIdBft
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
}
func PastLabelOnlyConflicts(newLabel string, Labels map[string]int) {
	id := 1
	Labels[newLabel] = 1
	allVisited := make([]int, 0)
	//Missing: Traverse down till conflicts and mark the past conflicts BFS
	Stack := make([]int, 0)
	Stack = append(Stack, outputLabelsMapOwnerID[newLabel].ID)
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != id {
			exploredSearchLedger[curVertex] = id
			for curLabel := range ledgerMap[curVertex].InputConflictLabels {
				Labels[curLabel] = 1
			}
			for nextVertex := range ledgerMap[curVertex].ParentsConflicts {
				Stack = append(Stack, nextVertex)
			}
		}
	}
	// clean exploredIdBft
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
}

// compute bad labels in the past cone
func PastLabels(id int, Labels map[string]int) {
	/* allVisited := make([]int, 0)
	//Missing: Traverse down till conflicts and mark the past conflicts BFS
	Stack := make([]int, 0)
	Stack = append(Stack, id)
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != id {
			exploredSearchLedger[curVertex] = id
			for curLabel := range ledgerMap[curVertex].InputConflictLabels {
				Labels[curLabel] = 1
			}
			for nextVertex := range ledgerMap[curVertex].ParentsConflicts {
				Stack = append(Stack, nextVertex)
			}
		}
	}
	// clean exploredIdBft
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	} */

	allVisited := make([]int, 0)
	//Missing: Traverse down till conflicts and mark the past conflicts BFS
	Stack := make([]int, 0)
	Stack = append(Stack, id)
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != id {
			exploredSearchLedger[curVertex] = id
			for curLabel := range ledgerMap[curVertex].InputLabels {
				Labels[curLabel] = 1
			}
			for nextVertex := range ledgerMap[curVertex].Parents {
				Stack = append(Stack, nextVertex)
			}
		}
	}
	// clean exploredIdBft
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
}

func CreateOutputID(id int, outputLabel string) StringInt {
	return StringInt{
		ID:          id,
		OutputLabel: outputLabel,
	}
}

// CreateGenesis creates the genesis transaction
func CreateGenesis(numOutputs int) Transaction {
	outputLabels := make(map[string]int)
	for i := 0; i < numOutputs; i++ {
		randString := hash(strconv.Itoa(rand.Int()))
		outputLabels[randString] = i
		outputLabelsSliceOwnerID = append(outputLabelsSliceOwnerID, CreateOutputID(idGenesis, randString))
		unspentLabelsSlice = append(unspentLabelsSlice, CreateOutputID(idGenesis, randString))
		outputLabelsMapOwnerID[randString] = &IntIntIntInt{ID: idGenesis, indexSlice: len(outputLabelsSliceOwnerID) - 1, indexUnspentSlice: len(unspentLabelsSlice) - 1, indexSpentSlice: -1}
		// outputLabelsMapConsumerIDs[randString] = map[int]int{} Questionable initialization
	}
	return Transaction{
		ID:                  idGenesis,
		InputLabels:         map[string]int{},
		OutputLabels:        outputLabels,
		Parents:             map[int]int{},
		Children:            map[int]int{},
		ChildrenConflicts:   map[int]int{},
		ParentsConflicts:    map[int]int{},
		IsConflict:          true,
		DirectConflicts:     map[int]int{},
		InputConflictLabels: map[string]int{},
		Weight:              1,
	}
}
func getRandNumInputOutput() (int, int) {
	curRealValue := rand.Float64()
	curIndex := 0
	for {
		if inputOutputDistribution[curIndex].weight < curRealValue {
			curIndex = curIndex + 1
		} else {
			break
		}
	}
	return inputOutputDistribution[curIndex].inputs, inputOutputDistribution[curIndex].outputs
}

func CreateLabelsNew(curInputLabels map[string]int, curOutputLabels map[string]int, probabilityConflict float64) {
	// number of inputs and outputs are random
	numInputLabels, numOutputLabels := getRandNumInputOutput()
	// generate input labels
	curNumInputLabels := 0
	// pick randomly exisiting spent output
	randValue := rand.Float64()
	allLabels := map[string]int{}
	if len(unconfirmedSpentLabelsSlice) > 0 && randValue < probabilityConflict {
		firstLabelNum := rand.Intn(len(unconfirmedSpentLabelsSlice))
		curLabel := unconfirmedSpentLabelsSlice[firstLabelNum].OutputLabel
		//curParent := outputLabelsMapOwnerID[curLabel].ID
		futureLabelsUntilConflict(curLabel, allLabels)
		//PastLabels(curParent, allPastLabels)
		curNumInputLabels = curNumInputLabels + 1
		curInputLabels[curLabel] = 1
	}
	badAttempts := 0
	for curNumInputLabels < numInputLabels {
		if badAttempts > numBadAttemptsInputLabel {
			numInputLabels = numInputLabels - 1
			badAttempts = 0
		}
		labelNum := rand.Intn(len(unspentLabelsSlice))
		curInputLabel := unspentLabelsSlice[labelNum]
		curPastLabelsPast := map[string]int{}
		PastLabelOnlyConflicts(curInputLabel.OutputLabel, curPastLabelsPast)
		//curPastLabels[curInputLabel.OutputLabel] = 1
		//PastLabels(curParent, curPastLabels)
		curOK := true
		for curLabel := range curPastLabelsPast {
			_, ok := allLabels[curLabel]
			if ok {
				badAttempts = badAttempts + 1
				curOK = false
				break
			}
		}
		if !curOK {
			continue
		}
		curNumInputLabels = curNumInputLabels + 1
		for curLabel := range curPastLabelsPast {
			allLabels[curLabel] = 1
		}

		curInputLabels[curInputLabel.OutputLabel] = 1
	}
	// generate output labels
	for j := 0; j < numOutputLabels; j++ {
		curOutputLabel := hash(strconv.Itoa(rand.Int()))
		curOutputLabels[curOutputLabel] = j
	}
}

func CreateLabels(curInputLabels map[string]int, curOutputLabels map[string]int, probabilityConflict float64) {
	// number of inputs and outputs are random
	numInputLabels, numOutputLabels := getRandNumInputOutput()
	// generate input labels
	curNumInputLabels := 0
	allPastLabels := map[string]int{}
	// pick randomly exisiting spent output
	randValue := rand.Float64()
	if len(unconfirmedSpentLabelsSlice) > 0 && randValue < probabilityConflict {
		firstLabelNum := rand.Intn(len(unconfirmedSpentLabelsSlice))
		curLabel := unconfirmedSpentLabelsSlice[firstLabelNum].OutputLabel
		curParent := outputLabelsMapOwnerID[curLabel].ID
		allPastLabels[curLabel] = 1
		PastLabels(curParent, allPastLabels)
		curNumInputLabels = curNumInputLabels + 1
		curInputLabels[curLabel] = 1
	}
	badAttempts := 0
	for curNumInputLabels < numInputLabels {
		if badAttempts > numBadAttemptsInputLabel {
			numInputLabels = numInputLabels - 1
			badAttempts = 0
		}
		labelNum := rand.Intn(len(unspentLabelsSlice))
		curInputLabel := unspentLabelsSlice[labelNum]
		curParent := outputLabelsMapOwnerID[curInputLabel.OutputLabel].ID
		curPastLabels := map[string]int{}
		curPastLabels[curInputLabel.OutputLabel] = 1
		PastLabels(curParent, curPastLabels)
		curOK := true
		for curLabel := range curPastLabels {
			_, ok := allPastLabels[curLabel]
			if ok {
				badAttempts = badAttempts + 1
				curOK = false
				break
			}
		}
		if !curOK {
			continue
		}
		curNumInputLabels = curNumInputLabels + 1
		for curLabel := range curPastLabels {
			allPastLabels[curLabel] = 1
		}
		curInputLabels[curInputLabel.OutputLabel] = 1
	}
	// generate output labels
	for j := 0; j < numOutputLabels; j++ {
		curOutputLabel := hash(strconv.Itoa(rand.Int()))
		curOutputLabels[curOutputLabel] = j
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
func EverGrowingLedger(probabilityConflict float64, numTransactionsStart int, doBranch bool) {
	exploredSearchLedger = make([]int, numTransactionsStart+1)
	exploredNestedSearchLedger = make([]int, numTransactionsStart+1)
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
	// Create numTransactionsStart random transactions
	for idTransaction := idGenesis + 1; idTransaction <= numTransactionsStart; idTransaction++ {
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
		CreateLabels(curInputLabels, curOutputLabels, probabilityConflict)

		// create transaction
		createLedgerStopwatch.Start()
		newLedgerNode := CreateTransaction(idTransaction, curInputLabels, curOutputLabels)

		// add to the global Ledger DAG slice
		ledgerMap[idTransaction] = &newLedgerNode
		if doBranch {
			_ = GetBranch(idTransaction)
		}
		createLedgerStopwatch.Pause()

	}
	Analytics(createLedgerStopwatch.Elapsed())
}

func GrowingLedgerPruningConflictsLimit(probabilityConflict float64, numTransactionsStart int, upBoundConflicts int) {
	numConfirmedTransactions := 0
	exploredSearchLedger = make([]int, numTransactionsStart+1)
	exploredNestedSearchLedger = make([]int, numTransactionsStart+1)
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
	// Create numTransactionsStart random transactions
	for idTransaction := idGenesis + 1; idTransaction <= numTransactionsStart; idTransaction++ {
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
		CreateLabels(curInputLabels, curOutputLabels, probabilityConflict)

		// create transaction
		createLedgerStopwatch.Start()
		newLedgerNode := CreateTransaction(idTransaction, curInputLabels, curOutputLabels)

		// add to the global Ledger DAG slice
		ledgerMap[idTransaction] = &newLedgerNode
		createLedgerStopwatch.Pause()
		//TraverseAndCheck("newTransaction" + strconv.Itoa(idTransaction))
		if numConflicts > upBoundConflicts {
			fmt.Fprintln(file, createLedgerStopwatch.Elapsed().Seconds(), len(ledgerMap), numConflicts, numConfirmedTransactions)
			createLedgerStopwatch.Start()
			pruneLedgerStopwatch.Start()
			curReality := getReality()
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
			DeleteRejectedTransactions(ledgerMap, outputLabelsMapConsumerIDs, threshold, idGenesis)
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
		//CreateLabels(curInputLabels, curOutputLabels, probabilityConflict)
		createInputOutputStopwatch.Start()
		CreateLabels(curInputLabels, curOutputLabels, probabilityConflict)
		createInputOutputStopwatch.Pause()
		// create transaction
		createLedgerStopwatch.Start()
		newLedgerNode := CreateTransaction(idTransaction, curInputLabels, curOutputLabels)
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

func Analytics(curTimestamp time.Duration) {
	curNumTransactions := len(ledgerMap)
	curNumConflicts := numConflicts
	/* Stack := make([]int, 0)
	Stack = append(Stack, idGenesis)
	// traverse the whole ledger DAG from teh genesis
	allVisited := make([]int, 0)
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != idGenesis {
			exploredSearchLedger[curVertex] = idGenesis
			curNumConflicts = curNumConflicts + 1
			for nextVertex := range ledgerMap[curVertex].ChildrenConflicts {
				Stack = append(Stack, nextVertex) // Probably do not go over already explored
			}
		}
	}

	// Clean exploredIDs
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	} */

	allAnalytics = append(allAnalytics, AllParameters{numConflicts: curNumConflicts, numTransactions: curNumTransactions, timestamp: curTimestamp.Seconds()})
}

func CleaningStructures() {
	maps.Clear(ledgerMap)
	outputLabelsSliceOwnerID = nil    // array that stores pairs outputLabel + id of transaction; Used for taking random UTXOs
	unspentLabelsSlice = nil          // array that stores pairs unspent outputLabel + id of transaction; Used for taking random UTXOs
	unconfirmedSpentLabelsSlice = nil // array that stores pairs spent existing outputLabel + id of transaction; Used for taking random UTXOs // This are still unconfirmed
	confirmedSpentlabelsSlice = nil   // array that stores pairs spent outputLabel + id of transaction // This are already confirmed
	maps.Clear(outputLabelsMapOwnerID)
	for key := range outputLabelsMapConsumerIDs {
		maps.Clear(outputLabelsMapConsumerIDs[key])
	}
	maps.Clear(outputLabelsMapConsumerIDs)
	//exploredSearchLedger = nil
	//exploredNestedSearchLedger = nil
	maps.Clear(confirmedTransactions) // map containing all confirmed Transactions
	numConflicts = 0
}

type conflictForSort struct {
	HashName string
	weight   float64
}

func GetTimeBranches(indexParameter int) {
	getTimeSlice := make([]int64, 0)
	Stack := make([]int, 0)
	Stack = append(Stack, idGenesis)
	// traverse the whole ledger DAG from teh genesis
	allVisited := make([]int, 0)
	numTx := float64(len(ledgerMap))
	probGetBranch := min(10.0, numTx) / numTx
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != idGenesis {
			exploredSearchLedger[curVertex] = idGenesis
			if probGetBranch < rand.Float64() {
				start := time.Now()
				A := GetBranch(curVertex)
				if len(A) < 1 {
					fmt.Println("error")
				}
				elapsed := time.Since(start)
				getTimeSlice = append(getTimeSlice, elapsed.Nanoseconds())
			}
			for nextVertex := range ledgerMap[curVertex].Children {
				Stack = append(Stack, nextVertex) // Probably do not go over already explored
			}
		}
	}

	// Clean exploredIDs
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
	file, err := os.Create("outBranch_" + strconv.Itoa(indexParameter) + ".txt")

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer file.Close()

	for t := range getTimeSlice {
		fmt.Fprintln(file, getTimeSlice[t])
	}

}
func assignWeightsAfterReality() {
	allVisited := make([]int, 0)
	//Missing: Traverse down till conflicts and mark the past conflicts BFS
	Stack := make([]int, 0)
	Stack = append(Stack, idGenesis)
	outputLabelsToDelete := map[string]int{}
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != idGenesis {
			exploredSearchLedger[curVertex] = idGenesis
			ledgerMap[curVertex].Weight = 1
			//delete all inputs from potential spentOutputs
			for curLabel := range ledgerMap[curVertex].InputLabels {
				outputLabelsToDelete[curLabel] = curVertex
			}

			for nextVertex := range ledgerMap[curVertex].Children {
				if ledgerMap[nextVertex].Weight > -0.5 {
					pastFine := true
					for curPar := range ledgerMap[nextVertex].ParentsConflicts {
						if ledgerMap[curPar].Weight < -0.5 {
							pastFine = false
						}
					}
					if pastFine {
						Stack = append(Stack, nextVertex)
					}
				}
			}
		}
	}
	numDel := 0
	for curLabel := range outputLabelsToDelete {
		numDel = numDel + 1
		confirmedSpentlabelsSlice = append(confirmedSpentlabelsSlice, CreateOutputID(outputLabelsToDelete[curLabel], curLabel))
		curIndex := outputLabelsMapOwnerID[curLabel].indexSpentSlice
		if curIndex != -1 {
			unconfirmedSpentLabelsSlice[curIndex] = unconfirmedSpentLabelsSlice[len(unconfirmedSpentLabelsSlice)-1]
			movedLabel := unconfirmedSpentLabelsSlice[curIndex].OutputLabel
			unconfirmedSpentLabelsSlice = unconfirmedSpentLabelsSlice[:len(unconfirmedSpentLabelsSlice)-1]
			if curIndex < len(unconfirmedSpentLabelsSlice) {
				outputLabelsMapOwnerID[movedLabel].indexSpentSlice = curIndex
			} else {
				outputLabelsMapOwnerID[movedLabel].indexSpentSlice = -1
			}
			outputLabelsMapOwnerID[curLabel].indexSpentSlice = -1
		}
	}
	//unconfirmedSpentLabelsSlice = unconfirmedSpentLabelsSlice[:len(unconfirmedSpentLabelsSlice)-numDel]

	// clean exploredIdBft
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
}
func LeaveMaximalElements(NewElements map[int]int, curParent int, Stack map[int]conflictForSort) {
	for id := range NewElements {
		allVisited := make([]int, 0)
		localStack := make([]int, 0)
		localStack = append(localStack, id)
		deleteTrue := false
		for len(localStack) > 0 {
			curVertex := localStack[len(localStack)-1]
			allVisited = append(allVisited, curVertex)
			localStack = localStack[:len(localStack)-1]
			if exploredSearchLedger[curVertex] != id {
				exploredSearchLedger[curVertex] = id
				_, ok := Stack[curVertex]
				if ok {
					deleteTrue = true
					break
				}
				if curVertex != id {
					_, ok := NewElements[curVertex]
					if ok {
						deleteTrue = true
						break
					}
				}

				for nextVertex := range ledgerMap[curVertex].ParentsConflicts {
					if nextVertex != curParent {
						localStack = append(localStack, nextVertex)
					}
				}

			}
		}
		for t := range allVisited {
			exploredSearchLedger[allVisited[t]] = 0
		}

		if deleteTrue {
			delete(NewElements, id)
			continue
		}
	}
}

func getReality() map[int]int {
	reality := map[int]int{}
	Stack := map[int]conflictForSort{}
	Stack[idGenesis] = conflictForSort{weight: ledgerMap[idGenesis].Weight, HashName: hash(strconv.Itoa(idGenesis))}
	for len(Stack) > 0 {
		maxWeight := -1.1
		maxWeightIds := map[int]conflictForSort{}
		for curID := range Stack {
			if maxWeight+0.00001 < Stack[curID].weight {
				maxWeight = Stack[curID].weight
				maxWeightIds = map[int]conflictForSort{}
				maxWeightIds[curID] = Stack[curID]
			}
			if 0.00001 > Abs(Stack[curID].weight-maxWeight) {
				maxWeightIds[curID] = Stack[curID]
			}
		}
		var maxHash string
		var winner int
		for winner = range maxWeightIds {
			maxHash = maxWeightIds[winner].HashName
			break
		}
		for curID := range maxWeightIds {
			if maxHash < maxWeightIds[curID].HashName {
				maxHash = maxWeightIds[curID].HashName
				winner = curID
			}
		}

		NewElements := map[int]int{}
		for nextVertex := range ledgerMap[winner].ChildrenConflicts {
			if ledgerMap[nextVertex].Weight >= -0.00001 {
				NewElements[nextVertex] = 1
			}
		}
		ledgerMap[winner].Weight = ledgerMap[winner].Weight + 1
		reality[winner] = 1
		delete(Stack, winner)
		// Remove transactions conflicting with winner from Stack and assign weights
		allVisited := make([]int, 0)
		localStack := make([]int, 0)
		for nextVertex := range ledgerMap[winner].DirectConflicts {
			localStack = append(localStack, nextVertex)
		}
		for len(localStack) > 0 {
			curVertex := localStack[len(localStack)-1]
			allVisited = append(allVisited, curVertex)
			localStack = localStack[:len(localStack)-1]
			if exploredSearchLedger[curVertex] != winner && ledgerMap[curVertex].Weight > -0.5 {
				exploredSearchLedger[curVertex] = winner
				ledgerMap[curVertex].Weight = -1.0
				delete(Stack, curVertex)
				for nextVertex := range ledgerMap[curVertex].ChildrenConflicts {
					localStack = append(localStack, nextVertex)
				}
			}
		}

		for t := range allVisited {
			exploredSearchLedger[allVisited[t]] = 0
		}
		// Leave only maximal elements from NewElements
		LeaveMaximalElements(NewElements, winner, Stack)
		for nextVertex := range NewElements {
			Stack[nextVertex] = conflictForSort{weight: ledgerMap[nextVertex].Weight, HashName: hash(strconv.Itoa(nextVertex))}
		}
	}
	return reality
}

func Abs(f float64) float64 {
	return -min(f, -f)
}

func min(a float64, b float64) float64 {
	if a > b {
		return b
	} else {
		return a
	}
}

func TraverseAndCheck(st string) {
	Stack := make([]int, 0)
	Stack = append(Stack, idGenesis)
	// traverse the whole ledger DAG from teh genesis
	allVisited := make([]int, 0)
	actualNumberOfConflicts := 0
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != idGenesis {
			exploredSearchLedger[curVertex] = idGenesis
			// all checks
			// Children conflict and back
			for aConflictFuture := range ledgerMap[curVertex].ChildrenConflicts {
				_, ok := ledgerMap[aConflictFuture]
				if !ok {
					panic("Fail : children conflict" + st)
				}
				if ledgerMap[curVertex].IsConflict {
					_, ok = ledgerMap[aConflictFuture].ParentsConflicts[curVertex]
					if !ok {
						panic("Fail : children conflict exist but not wise versa" + st)
					}
				}
			}
			// Parent conflict and back
			for aConflictPast := range ledgerMap[curVertex].ParentsConflicts {
				_, ok := ledgerMap[aConflictPast]
				if !ok {
					panic("Fail : parents conflict" + st)
				}
				if ledgerMap[curVertex].IsConflict {
					_, ok = ledgerMap[aConflictPast].ChildrenConflicts[curVertex]
					if !ok {
						panic("Fail : parents conflict exist but not wise versa" + st)
					}
				}
			}
			// Parent check
			for curParent := range ledgerMap[curVertex].Parents {
				_, ok := ledgerMap[curParent]
				if !ok {
					panic("Fail : parents" + st)
				}
			}
			//
			if ledgerMap[curVertex].IsConflict {
				actualNumberOfConflicts = actualNumberOfConflicts + 1
			}

			for nextVertex := range ledgerMap[curVertex].Children {
				Stack = append(Stack, nextVertex)
			}
		}
	}
	if actualNumberOfConflicts != numConflicts+1 {
		fmt.Println("actualNumberOfConflicts = ", actualNumberOfConflicts, "numConflicts = ", numConflicts)
		panic("numConflicts mismathes")
	}
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
	for curOutput := range outputLabelsMapConsumerIDs {
		A := outputLabelsMapConsumerIDs[curOutput]
		for curOwner := range A {
			_, ok := ledgerMap[curOwner]
			if !ok {
				panic("Fail : no owner for an output in outputLabelsMapConsumerIDs" + st)
			}
		}
	}
	for curOutput := range outputLabelsMapOwnerID {
		A := outputLabelsMapOwnerID[curOutput]
		indSlice := A.indexSlice
		indSpentSlice := A.indexSpentSlice
		indUnspentSlice := A.indexUnspentSlice
		id := A.ID
		curId := outputLabelsSliceOwnerID[indSlice].ID
		if id != curId {
			panic("ids are different" + st)
		}
		output := outputLabelsSliceOwnerID[indSlice].OutputLabel
		if curOutput != output {
			panic("outputs are different" + st)
		}
		if indSpentSlice >= 0 {
			curId := unconfirmedSpentLabelsSlice[indSpentSlice].ID
			if id != curId {
				panic("ids are different" + st)
			}
			if curOutput != unconfirmedSpentLabelsSlice[indSpentSlice].OutputLabel {
				panic("outputs are different" + st)
			}
		}
		if indUnspentSlice >= 0 {
			curId := unspentLabelsSlice[indUnspentSlice].ID
			if id != curId {
				panic("ids are different" + st)
			}
			if curOutput != unspentLabelsSlice[indUnspentSlice].OutputLabel {
				panic("outputs are different" + st)
			}
		}
	}

}

type FloatIntInt struct {
	weight  float64
	inputs  int
	outputs int
}

func ReadThreeInts(r io.Reader) ([][]int, []FloatIntInt) {
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanWords)
	result := make([][]int, 256)
	for i := range result {
		result[i] = make([]int, 256)
	}
	totalCount := 0
	for scanner.Scan() {
		inputs, _ := strconv.Atoi(scanner.Text())
		scanner.Scan()
		outputs, _ := strconv.Atoi(scanner.Text())
		scanner.Scan()
		counts, _ := strconv.Atoi(scanner.Text())
		if inputs != 0 && outputs != 0 && counts != 0 {
			result[inputs][outputs] = counts
			totalCount = totalCount + counts
		}
	}
	cumWeight := 0.0
	distribution := make([]FloatIntInt, 0)
	for i := range result {
		for j := range result[i] {
			curWeight := float64(result[i][j]) / float64(totalCount)
			cumWeight = cumWeight + curWeight
			triple := FloatIntInt{weight: cumWeight, inputs: i, outputs: j}
			if curWeight > 0.000000001 {
				distribution = append(distribution, triple)
			}

		}
	}
	return result, distribution
}
func getInputOutputDistribution() {
	if ifUniform != 1 {
		dat, err := os.Open(file_distribution)
		check(err)
		_, inputOutputDistribution = ReadThreeInts(bufio.NewReader(dat))
		if len(inputOutputDistribution) < 1 {
			panic("1")
		}
	} else {
		cumWeight := 0.0
		curWeight := 1.0 / float64(numInputsMax*numOutputsMax)
		for in := 1; in <= numInputsMax; in++ {
			for out := 1; out <= numOutputsMax; out++ {
				cumWeight = cumWeight + curWeight
				triple := FloatIntInt{weight: cumWeight, inputs: in, outputs: out}
				inputOutputDistribution = append(inputOutputDistribution, triple)
			}
		}
	}
}

func outLedgerPruneLimitConflict() {
	file, _ := os.Create("ledgerGrowAndPrune.txt")
	defer file.Close()
	fmt.Fprintln(file, "0", len(ledgerMap), numConflicts)
	numMillisecond := 0.1
	tickNumber := 0
	for range time.Tick(time.Millisecond * 100) {
		tickNumber = tickNumber + 1
		fmt.Fprintln(file, numMillisecond*float64(tickNumber), len(ledgerMap), numConflicts)
		fmt.Println(len(ledgerMap), numConflicts)
	}
}

func outLedgerPruneLimitConflictWithTimer(timer *stopwatch.Stopwatch) {
	file, _ := os.Create("ledgerGrowAndPrune.txt")
	defer file.Close()
	fmt.Fprintln(file, "0", len(ledgerMap), numConflicts)
	tickNumber := 0
	for {
		if timer.Elapsed() > time.Duration(tickNumber)*measureEverySec {
			tickNumber = int(time.Duration(tickNumber)*measureEverySec) + 1
			fmt.Fprintln(file, timer.Elapsed().Seconds(), len(ledgerMap), numConflicts)
			fmt.Println(timer.Elapsed().Seconds(), len(ledgerMap), numConflicts)
		}
	}
}

func main() {

	ifUniform = 1
	getInputOutputDistribution()
	// create ledgerMap and draw the DAG
	// EverGrowingLedger(0.2, 50, false)
	// DrawDAG("test2", ledgerMap, outputLabelsMapConsumerIDs, threshold, idGenesis)

	/*
		probabilityConflict := []float64{0.05, 0.05, 0.1, 0.5}
		numTransactionsStart := 2000000
		computeBranch := []bool{false, true}
		// Test 1: ever growing ledger
		file, _ := os.Create("ledgerGrow.txt")
		fmt.Fprintln(file, len(probabilityConflict))
		defer file.Close()

		for i := range probabilityConflict {
			for j := range computeBranch {
				EverGrowingLedger(probabilityConflict[i], numTransactionsStart, computeBranch[j])

				fmt.Fprintln(file, probabilityConflict[i], "\t", computeBranch[j], "\t", len(allAnalytics))
				for s := range allAnalytics {
					fmt.Fprintln(file, allAnalytics[s].timestamp, "\t", allAnalytics[s].numTransactions, "\t", allAnalytics[s].numConflicts)
				}
				allAnalytics = nil
				CleaningStructures()
			}

		} */

	//writeInFile()
	//createLineChart(probabilityConflict)
	// Test 2: growing ledger + pruning conflicts at limit
	// probabilityConflict := 0.01
	// numTransactionsStart := 40000000
	// upBoundConflicts := 5000
	// GrowingLedgerPruningConflictsLimit(probabilityConflict, numTransactionsStart, upBoundConflicts)
	// CleaningStructures()
	// Test 2.5 GetReality for different upperlimit
	/* numGetReality := 1000

	upBoundConflicts := []int{10000, 20000, 40000}
	probabilityConflict := 0.1
	numTransactionsStart := 500000
	globalSeed = 0
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
			reality := getReality()
			fmt.Fprintln(file, getRealityStopwatch.Elapsed().Seconds(), "\t", len(reality))
			fmt.Println(len(ledgerMap), numConflicts, len(reality), "Reality = ", getRealityStopwatch.Elapsed().Seconds(), "Ledger = ", getLedgerStopwatch.Elapsed().Seconds())
			getRealityStopwatch.Reset()
			getLedgerStopwatch.Reset()
			CleaningStructures()
		}
	} */

	// Test 3: growing ledger +  resolving conflicts completely
	/* probabilityConflict := 0.1
	numTransactionsStart := 10000000
	pruneDelaySeconds := 10.0
	GrowingLedgerPruningConflictsTimely(probabilityConflict, numTransactionsStart, pruneDelaySeconds) */

	// Test 4: get branch for all transactions / histogram

	/* for i := range probabilityConflict {
		EverGrowingLedger(probabilityConflict[i], numTransactionsStart, i, false)

		//DrawDAG("1")
		TraverseAndCheck()
		curReality := getReality()
		assignWeightsAfterReality()
		DeleteRejectedTransactions(ledgerMap,outputLabelsMapConsumerIDs,threshold,idGenesis)
		TraverseAndCheck()
		//DrawDAG("2")
		if ledgerMap[1].IsConflict {
			fmt.Println("fine")
		}
		if len(curReality) < 1 {
			fmt.Println("bad")
		}
		CleaningStructures()
	} */

}
