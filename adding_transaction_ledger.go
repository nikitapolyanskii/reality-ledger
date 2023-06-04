package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
)

// random number of inputs/outputs of transactions
var numInputsMax = 2  // max number of inputs of transactions, i.e. it could be 1 or 2
var numOutputsMax = 3 // max number of outputs of transactions, i.e. it could be 1, 2, or 3

// fixed number of outputs of the genesis transaction
var numOutputsGenesis = 16

// inputOutputDistribution is a slice of DistribEntry2D structs that describes the distribution of random inputs and outputs of transactions
var inputOutputDistribution []DistribEntry2D

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

// OutputInfo represents information about an output created by a transaction.
// It stores the transaction ID, along with the indices of the slices where the output is stored.
type OutputInfo struct {
	ID                int // ID of the transaction creating the output
	indexSlice        int // Index of the slice outputLabelsSliceOwnerID containing this output
	indexUnspentSlice int // Index of the slice unspentLabelsSlice containing this output
	indexSpentSlice   int // Index of the slice unconfirmedSpentLabelsSlice containing this output
}

type DistribEntry2D struct {
	weight  float64
	inputs  int
	outputs int
}

func getInputOutputDistribution() {

	cumWeight := 0.0
	curWeight := 1.0 / float64(numInputsMax*numOutputsMax)
	for in := 1; in <= numInputsMax; in++ {
		for out := 1; out <= numOutputsMax; out++ {
			cumWeight = cumWeight + curWeight
			triple := DistribEntry2D{weight: cumWeight, inputs: in, outputs: out}
			inputOutputDistribution = append(inputOutputDistribution, triple)
		}
	}

}

// CreateGenesis creates the genesis transaction
func CreateGenesis(numOutputs int) Transaction {
	// Initialize the output labels map
	outputLabels := make(map[string]int)

	// Generate random output labels and populate the data structures
	for i := 0; i < numOutputs; i++ {
		randString := hash(strconv.Itoa(rand.Int()))
		outputLabels[randString] = i
		outputLabelsSliceOwnerID = append(outputLabelsSliceOwnerID, CreateOutputID(idGenesis, randString))
		unspentLabelsSlice = append(unspentLabelsSlice, CreateOutputID(idGenesis, randString))
		outputLabelsMapOwnerID[randString] = &OutputInfo{
			ID:                idGenesis,
			indexSlice:        len(outputLabelsSliceOwnerID) - 1,
			indexUnspentSlice: len(unspentLabelsSlice) - 1,
			indexSpentSlice:   -1,
		}
	}

	// Create and return the genesis transaction
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

// CreateTransaction creates a new Transaction with the specified ID, input labels, and output labels.
// It updates relevant data structures and returns the created Transaction.
func CreateTransaction(id int, inputLabels map[string]int, outputLabels map[string]int) Transaction {
	// Update some data structures

	// Update outputLabelsSliceOwnerID, unspentLabelsSlice, and outputLabelsMapOwnerID
	keys := make([]string, 0)
	for key := range outputLabels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, addLabel := range keys {
		// Create an output ID and add it to outputLabelsSliceOwnerID
		outputLabelsSliceOwnerID = append(outputLabelsSliceOwnerID, CreateOutputID(id, addLabel))

		// Add the output ID to unspentLabelsSlice
		unspentLabelsSlice = append(unspentLabelsSlice, CreateOutputID(id, addLabel))

		// Update outputLabelsMapOwnerID to map the output label to the output ID
		outputLabelsMapOwnerID[addLabel] = &OutputInfo{
			ID:                id,
			indexSlice:        len(outputLabelsSliceOwnerID) - 1,
			indexUnspentSlice: len(unspentLabelsSlice) - 1,
			indexSpentSlice:   -1,
		}
	}

	// Update unspentLabelsSlice and outputLabelsMapOwnerID for input labels
	keys = make([]string, 0)
	for key := range inputLabels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	totalDeleteUnspent := 0
	for _, removeLabel := range keys {
		curIndexUnspentSlice := outputLabelsMapOwnerID[removeLabel].indexUnspentSlice
		if curIndexUnspentSlice != -1 {
			totalDeleteUnspent = totalDeleteUnspent + 1

			// Move the last unspent label to the current index
			unspentLabelsSlice[curIndexUnspentSlice] = unspentLabelsSlice[len(unspentLabelsSlice)-totalDeleteUnspent]

			// Update the copy label and its index
			copyLabel := unspentLabelsSlice[curIndexUnspentSlice].OutputLabel
			if curIndexUnspentSlice == len(unspentLabelsSlice)-totalDeleteUnspent {
				outputLabelsMapOwnerID[removeLabel].indexUnspentSlice = -1
			} else {
				outputLabelsMapOwnerID[removeLabel].indexUnspentSlice = -1
				outputLabelsMapOwnerID[copyLabel].indexUnspentSlice = curIndexUnspentSlice
			}
		}

		curIndexSpentSlice := outputLabelsMapOwnerID[removeLabel].indexSpentSlice
		addLabel := removeLabel
		if curIndexSpentSlice == -1 {
			ownerID := outputLabelsMapOwnerID[addLabel].ID

			// Add the label to unconfirmedSpentLabelsSlice
			unconfirmedSpentLabelsSlice = append(unconfirmedSpentLabelsSlice, CreateOutputID(ownerID, addLabel))

			// Update outputLabelsMapOwnerID to map the label to the unconfirmed spent index
			outputLabelsMapOwnerID[addLabel].indexSpentSlice = len(unconfirmedSpentLabelsSlice) - 1
		}
	}

	// Update conflicts and dependencies
	isConflict := false
	curDirectConflicts := make(map[int]int)
	curInputConflictLabels := make(map[string]int)
	curParents := make(map[int]int)
	curParentsConflicts := make(map[int]int)
	for curInputLabel := range inputLabels {
		curParents[outputLabelsMapOwnerID[curInputLabel].ID] = 1

		// Ensure outputLabelsMapConsumerIDs has an entry for the input label
		if len(outputLabelsMapConsumerIDs[curInputLabel]) < 1 {
			outputLabelsMapConsumerIDs[curInputLabel] = map[int]int{}
		}

		// Add the current transaction ID as a consumer of the input label
		outputLabelsMapConsumerIDs[curInputLabel][id] = 1

		// Check for conflicts
		if len(outputLabelsMapConsumerIDs[curInputLabel]) > 1 {
			curInputConflictLabels[curInputLabel] = 1
			isConflict = true

			// Process conflicts with other transactions
			for txIDConflictWithCurrentOne := range outputLabelsMapConsumerIDs[curInputLabel] {
				if txIDConflictWithCurrentOne == id {
					continue
				}

				// Add the transaction ID as a direct conflict
				curDirectConflicts[txIDConflictWithCurrentOne] = 1

				// Update conflicts and dependencies in ledgerMap
				ledgerMap[txIDConflictWithCurrentOne].DirectConflicts[id] = 1
				ledgerMap[txIDConflictWithCurrentOne].InputConflictLabels[curInputLabel] = 1

				// Handle conflicts recursively
				if !ledgerMap[txIDConflictWithCurrentOne].IsConflict {
					downUpVisited := make([]int, 0)
					allVisited := make([]int, 0)

					// Traverse down till conflicts and mark the past conflicts (BFS)
					conflictMapToDelete := ledgerMap[txIDConflictWithCurrentOne].ChildrenConflicts
					Stack := make([]int, 0)
					Stack = append(Stack, txIDConflictWithCurrentOne)
					for len(Stack) > 0 {
						curVertex := Stack[len(Stack)-1]
						allVisited = append(allVisited, curVertex)
						downUpVisited = append(downUpVisited, curVertex)
						Stack = Stack[:len(Stack)-1]
						if exploredSearchLedger[curVertex] != txIDConflictWithCurrentOne {
							exploredSearchLedger[curVertex] = txIDConflictWithCurrentOne

							// Update conflicts in the current vertex
							if curVertex != txIDConflictWithCurrentOne {
								ledgerMap[curVertex].ChildrenConflicts[txIDConflictWithCurrentOne] = 1
								for conflictToDelete := range conflictMapToDelete {
									delete(ledgerMap[curVertex].ChildrenConflicts, conflictToDelete)
								}
							}

							// Continue BFS if the current vertex is not a conflict
							if !ledgerMap[curVertex].IsConflict {
								for nextVertex := range ledgerMap[curVertex].Parents {
									Stack = append(Stack, nextVertex)
								}
							}
						}
					}

					// Traverse up till conflicts and mark the future conflicts (BFS)
					conflictMapToDelete = ledgerMap[txIDConflictWithCurrentOne].ParentsConflicts
					Stack = make([]int, 0)
					Stack = append(Stack, txIDConflictWithCurrentOne)
					for len(Stack) > 0 {
						curVertex := Stack[len(Stack)-1]
						downUpVisited = append(downUpVisited, curVertex)
						allVisited = append(allVisited, curVertex)
						Stack = Stack[:len(Stack)-1]
						if exploredSearchLedger[curVertex] != -txIDConflictWithCurrentOne {
							exploredSearchLedger[curVertex] = -txIDConflictWithCurrentOne

							// Update conflicts in the current vertex
							if curVertex != txIDConflictWithCurrentOne {
								ledgerMap[curVertex].ParentsConflicts[txIDConflictWithCurrentOne] = 1
								for conflictToDelete := range conflictMapToDelete {
									delete(ledgerMap[curVertex].ParentsConflicts, conflictToDelete)
								}
							}

							// Continue BFS if the current vertex is not a conflict
							if !ledgerMap[curVertex].IsConflict {
								for nextVertex := range ledgerMap[curVertex].Children {
									Stack = append(Stack, nextVertex)
								}
							}
						}
					}

					// Clean exploredSearchLedger
					for t := range allVisited {
						exploredSearchLedger[allVisited[t]] = 0
					}

					// Update conflicts and dependencies in the relevant transactions
					for _, t := range downUpVisited {
						if ledgerMap[t].IsConflict {
							for s := range ledgerMap[t].ParentsConflicts {
								ledgerMap[s].ChildrenConflicts[t] = 1
							}
							for s := range ledgerMap[t].ChildrenConflicts {
								if _, oook := ledgerMap[s]; !oook {
									fmt.Println(len(ledgerMap))
									delete(ledgerMap[t].ChildrenConflicts, s)
								} else {
									ledgerMap[s].ParentsConflicts[t] = 1
								}
							}
						}
					}

					// Mark the conflicting transaction as a conflict
					if ledgerMap[txIDConflictWithCurrentOne].IsConflict == false {
						ledgerMap[txIDConflictWithCurrentOne].IsConflict = true
						numConflicts = numConflicts + 1
					}
				}
			}
		}
	}

	// Process dependencies
	Stack := make([]int, 0)
	for parent := range curParents {
		Stack = append(Stack, parent)
		ledgerMap[parent].Children[id] = 1
		if ledgerMap[parent].IsConflict {
			curParentsConflicts[parent] = 1
		} else {
			for k, v := range ledgerMap[parent].ParentsConflicts {
				curParentsConflicts[k] = v
			}
		}
	}

	// Traverse the past of the current transaction
	allVisited := make([]int, 0)
	if isConflict {
		numConflicts = numConflicts + 1
		for len(Stack) > 0 {
			curVertex := Stack[len(Stack)-1]
			allVisited = append(allVisited, curVertex)
			Stack = Stack[:len(Stack)-1]
			if exploredSearchLedger[curVertex] != id {
				exploredSearchLedger[curVertex] = id
				ledgerMap[curVertex].ChildrenConflicts[id] = 1
				if !ledgerMap[curVertex].IsConflict {
					for nextVertex := range ledgerMap[curVertex].Parents {
						Stack = append(Stack, nextVertex)
					}
				}
			}
		}
	}

	// Clean exploredIDs
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}

	// Return the result
	return Transaction{
		ID:                  id,
		InputLabels:         inputLabels,
		OutputLabels:        outputLabels,
		Parents:             curParents,
		Children:            map[int]int{},
		ChildrenConflicts:   map[int]int{},
		ParentsConflicts:    curParentsConflicts,
		IsConflict:          isConflict,
		DirectConflicts:     curDirectConflicts,
		InputConflictLabels: curInputConflictLabels,
	}
}
