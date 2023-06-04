package main

import "strconv"

type conflictForSort struct {
	HashName string
	weight   float64
}

// LeaveMaximalElements removes non-maximal elements from the NewElements map based on conflict resolution.
// It takes the following parameters:
// - NewElements: A map containing the elements to be checked and modified.
// - curParent: The ID of the current parent transaction.
// - stack: A map representing the conflict stack.
func LeaveMaximalElements(NewElements map[int]int, curParent int, stack map[int]conflictForSort) {
	for id := range NewElements {
		allVisited := make([]int, 0)
		localStack := make([]int, 0)
		localStack = append(localStack, id)
		deleteTrue := false

		// Traverse down the conflict chain
		for len(localStack) > 0 {
			curVertex := localStack[len(localStack)-1]
			allVisited = append(allVisited, curVertex)
			localStack = localStack[:len(localStack)-1]

			// Check if the current vertex is explored or conflicts with stack
			if exploredSearchLedger[curVertex] != id {
				exploredSearchLedger[curVertex] = id

				// Check if the current vertex conflicts with stack or NewElements
				_, ok := stack[curVertex]
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

				// Traverse to the next vertices in the conflict chain
				for nextVertex := range ledgerMap[curVertex].ParentsConflicts {
					if nextVertex != curParent {
						localStack = append(localStack, nextVertex)
					}
				}
			}
		}

		// Clean up exploredSearchLedger
		for t := range allVisited {
			exploredSearchLedger[allVisited[t]] = 0
		}

		// Delete the vertex if it conflicts with stack or NewElements
		if deleteTrue {
			delete(NewElements, id)
			continue
		}
	}
}

// GetReality returns a map representing the reality of the ledger.
func GetReality() map[int]int {
	reality := map[int]int{}
	stack := map[int]conflictForSort{}
	stack[idGenesis] = conflictForSort{weight: ledgerMap[idGenesis].Weight, HashName: hash(strconv.Itoa(idGenesis))}

	// Iterate until the stack is empty
	for len(stack) > 0 {
		maxWeight := -1.1
		maxWeightIds := map[int]conflictForSort{}

		// Find the maximum weight and the corresponding IDs in the stack
		for curID := range stack {
			if maxWeight+0.00001 < stack[curID].weight {
				maxWeight = stack[curID].weight
				maxWeightIds = map[int]conflictForSort{}
				maxWeightIds[curID] = stack[curID]
			}
			if 0.00001 > Abs(stack[curID].weight-maxWeight) {
				maxWeightIds[curID] = stack[curID]
			}
		}

		var maxHash string
		var winner int

		// Find the winner based on the maximum weight and hash value
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

		// Update the weight and add the winner to the reality map
		ledgerMap[winner].Weight = ledgerMap[winner].Weight + 1
		reality[winner] = 1

		// Remove transactions conflicting with the winner from stack and assign weights
		allVisited := make([]int, 0)
		localStack := make([]int, 0)
		for nextVertex := range ledgerMap[winner].DirectConflicts {
			localStack = append(localStack, nextVertex)
		}
		for len(localStack) > 0 {
			curVertex := localStack[len(localStack)-1]
			allVisited = append(allVisited, curVertex)
			localStack = localStack[:len(localStack)-1]

			// Check if the current vertex is explored and update its weight
			if exploredSearchLedger[curVertex] != winner && ledgerMap[curVertex].Weight > -0.5 {
				exploredSearchLedger[curVertex] = winner
				ledgerMap[curVertex].Weight = -1.0
				delete(stack, curVertex)

				// Traverse to the next vertices in the conflict chain
				for nextVertex := range ledgerMap[curVertex].ChildrenConflicts {
					localStack = append(localStack, nextVertex)
				}
			}
		}

		// Clean up exploredSearchLedger
		for t := range allVisited {
			exploredSearchLedger[allVisited[t]] = 0
		}

		// Leave only maximal elements from NewElements
		LeaveMaximalElements(NewElements, winner, stack)

		// Update stack with the remaining conflict vertices
		for nextVertex := range NewElements {
			stack[nextVertex] = conflictForSort{weight: ledgerMap[nextVertex].Weight, HashName: hash(strconv.Itoa(nextVertex))}
		}
	}

	return reality
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
