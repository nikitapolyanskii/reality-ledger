package main

import (
	"fmt"
	"sort"
)

// CreateTransaction creates a new transaction with input and output labels
func CreateTransaction(id int, inputLabels map[string]int, outputLabels map[string]int) Transaction {
	// Update some data structures
	keys := make([]string, 0)
	for key := range outputLabels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, addLabel := range keys {
		outputLabelsSliceOwnerID = append(outputLabelsSliceOwnerID, CreateOutputID(id, addLabel))
		unspentLabelsSlice = append(unspentLabelsSlice, CreateOutputID(id, addLabel))
		outputLabelsMapOwnerID[addLabel] = &IntIntIntInt{ID: id, indexSlice: len(outputLabelsSliceOwnerID) - 1, indexUnspentSlice: len(unspentLabelsSlice) - 1, indexSpentSlice: -1}
	}
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
			unspentLabelsSlice[curIndexUnspentSlice] = unspentLabelsSlice[len(unspentLabelsSlice)-totalDeleteUnspent]
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
			unconfirmedSpentLabelsSlice = append(unconfirmedSpentLabelsSlice, CreateOutputID(ownerID, addLabel))
			outputLabelsMapOwnerID[addLabel].indexSpentSlice = len(unconfirmedSpentLabelsSlice) - 1
		}
	}
	unspentLabelsSlice = unspentLabelsSlice[:len(unspentLabelsSlice)-totalDeleteUnspent]
	isConflict := false
	curDirectConflicts := make(map[int]int)
	curInputConflictLabels := make(map[string]int)
	curParents := make(map[int]int)
	curParentsConflicts := make(map[int]int)
	for curInputLabel := range inputLabels {
		curParents[outputLabelsMapOwnerID[curInputLabel].ID] = 1
		if len(outputLabelsMapConsumerIDs[curInputLabel]) < 1 {
			outputLabelsMapConsumerIDs[curInputLabel] = map[int]int{}
		}
		outputLabelsMapConsumerIDs[curInputLabel][id] = 1
		if len(outputLabelsMapConsumerIDs[curInputLabel]) > 1 {
			curInputConflictLabels[curInputLabel] = 1
			isConflict = true
			for txIDConflictWithCurrentOne := range outputLabelsMapConsumerIDs[curInputLabel] {
				if txIDConflictWithCurrentOne == id {
					continue
				}
				// going over tx ids conflicting with this tx
				curDirectConflicts[txIDConflictWithCurrentOne] = 1
				ledgerMap[txIDConflictWithCurrentOne].DirectConflicts[id] = 1
				ledgerMap[txIDConflictWithCurrentOne].InputConflictLabels[curInputLabel] = 1
				if !ledgerMap[txIDConflictWithCurrentOne].IsConflict {
					downUpVisited := make([]int, 0)
					allVisited := make([]int, 0)
					//Missing: Traverse down till conflicts and mark the past conflicts BFS
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
							if curVertex != txIDConflictWithCurrentOne {
								ledgerMap[curVertex].ChildrenConflicts[txIDConflictWithCurrentOne] = 1
								for conflictToDelete := range conflictMapToDelete {
									delete(ledgerMap[curVertex].ChildrenConflicts, conflictToDelete)
								}
							}
							if !ledgerMap[curVertex].IsConflict {
								for nextVertex := range ledgerMap[curVertex].Parents {
									Stack = append(Stack, nextVertex) // Probably do not go over already explored
								}
							}
						}
					}

					//Missing: Traverse up till conflicts and mark the future conflicts
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
							if curVertex != txIDConflictWithCurrentOne {
								ledgerMap[curVertex].ParentsConflicts[txIDConflictWithCurrentOne] = 1
								for conflictToDelete := range conflictMapToDelete {
									delete(ledgerMap[curVertex].ParentsConflicts, conflictToDelete)
								}
							}
							if !ledgerMap[curVertex].IsConflict {
								for nextVertex := range ledgerMap[curVertex].Children {
									Stack = append(Stack, nextVertex) // Probably do not go over already explored
								}
							}
						}
					}
					// clean exploredIdBft
					for t := range allVisited {
						exploredSearchLedger[allVisited[t]] = 0
					}
					for _, t := range downUpVisited {
						if ledgerMap[t].IsConflict {
							for s := range ledgerMap[t].ParentsConflicts {
								ledgerMap[s].ChildrenConflicts[t] = 1
							}
							for s := range ledgerMap[t].ChildrenConflicts {
								_, oook := ledgerMap[s]
								if !oook {
									fmt.Println(len(ledgerMap))
									delete(ledgerMap[t].ChildrenConflicts, s)
								} else {
									// It seems ledgerMap[t].ChildrenConflicts not cleaned well
									ledgerMap[s].ParentsConflicts[t] = 1
								}
							}
						}
					}
					// Mark as a confict
					if ledgerMap[txIDConflictWithCurrentOne].IsConflict == false {
						ledgerMap[txIDConflictWithCurrentOne].IsConflict = true
						numConflicts = numConflicts + 1
					}
				}
				//ledgerDAGSlice[txIDConflictWithCurrentOne].InputConflictLabels[curInputLabel] = 1
			}
		}
	}
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
	// traverse the past of the current tx to
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
						Stack = append(Stack, nextVertex) // Probably do not go over already explored
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
