package main

func DeleteRejectedTransactions(ledgerMap map[int]*Transaction, outputLabelsMapConsumerIDs map[string]map[int]int, threshold float64, idGenesis int) {
	allVisited := make([]int, 0)
	//Traverse up and mark all directly conflicting txs
	Stack := make([]int, 0)
	Stack = append(Stack, idGenesis)
	ToDeleteConflicts := make([]int, 0)
	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != idGenesis {
			exploredSearchLedger[curVertex] = idGenesis
			if ledgerMap[curVertex].Weight > threshold {
				for curConflictToDelete := range ledgerMap[curVertex].DirectConflicts {
					// mark all directly conflicting txs
					ToDeleteConflicts = append(ToDeleteConflicts, curConflictToDelete)
				}
			}
			for nextVertex := range ledgerMap[curVertex].ChildrenConflicts {
				Stack = append(Stack, nextVertex)
			}
		}
	}
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}
	if len(ToDeleteConflicts) < 1 {
		return
	}
	// delete all conflicting transactions
	allOutputLabelsToDelete := map[string]int{}
	for i := range ToDeleteConflicts {
		curConflictID := ToDeleteConflicts[i]

		_, ok := ledgerMap[curConflictID]
		if ok {
			allVisited = make([]int, 0)
			//Traverse past cone until conflicts and update ChildrenConflicts

			for t := range allVisited {
				exploredSearchLedger[allVisited[t]] = 0
			}

			allVisited = make([]int, 0)
			//Traverse future cone and mark all directly conflicting txs
			Stack = make([]int, 0)
			Stack = append(Stack, curConflictID)
			for len(Stack) > 0 {

				curVertex := Stack[len(Stack)-1]

				//

				allVisited = append(allVisited, curVertex)
				Stack = Stack[:len(Stack)-1]
				if exploredSearchLedger[curVertex] != curConflictID {
					exploredSearchLedger[curVertex] = curConflictID
					for nextVertex := range ledgerMap[curVertex].Children {
						Stack = append(Stack, nextVertex)
					}
				}
				// delete information related to conflicts conflicting with a given one
				allIDsVisitNow := map[int]int{}

				for curLabel := range ledgerMap[curVertex].InputLabels { //SHOULD BE FOR ALL INPUTS?
					delete(outputLabelsMapConsumerIDs[curLabel], curVertex)
					if len(outputLabelsMapConsumerIDs[curLabel]) == 0 {
						delete(outputLabelsMapConsumerIDs, curLabel)
						// It becomes unspent and we add curLabel to unspentLabelsSlice and remove it from  unconfirmedSpentLabelsSlice
						if outputLabelsMapOwnerID[curLabel].indexUnspentSlice == -1 {
							curOwner := outputLabelsMapOwnerID[curLabel].ID
							unspentLabelsSlice = append(unspentLabelsSlice, CreateOutputID(curOwner, curLabel))
							outputLabelsMapOwnerID[curLabel].indexUnspentSlice = len(unspentLabelsSlice) - 1
						}

						if outputLabelsMapOwnerID[curLabel].indexSpentSlice != -1 {
							curIndexSpent := outputLabelsMapOwnerID[curLabel].indexSpentSlice
							toCopyOut := unconfirmedSpentLabelsSlice[len(unconfirmedSpentLabelsSlice)-1]
							unconfirmedSpentLabelsSlice[curIndexSpent] = toCopyOut
							unconfirmedSpentLabelsSlice = unconfirmedSpentLabelsSlice[:len(unconfirmedSpentLabelsSlice)-1]
							outputLabelsMapOwnerID[curLabel].indexSpentSlice = -1
							if curIndexSpent < len(unconfirmedSpentLabelsSlice) {
								outputLabelsMapOwnerID[toCopyOut.OutputLabel].indexSpentSlice = curIndexSpent
							} else {
								outputLabelsMapOwnerID[toCopyOut.OutputLabel].indexSpentSlice = -1
							}
						}
					}
					for curIDConsumer := range outputLabelsMapConsumerIDs[curLabel] {
						allIDsVisitNow[curIDConsumer] = 1
						delete(ledgerMap[curIDConsumer].DirectConflicts, curVertex)
						if len(outputLabelsMapConsumerIDs[curLabel]) < 2 {
							delete(ledgerMap[curIDConsumer].InputConflictLabels, curLabel)
						}
					}
				}
				// check whether conflicts remain conflicts
				for curChangedIds := range allIDsVisitNow {
					if len(ledgerMap[curChangedIds].DirectConflicts) < 1 {
						// it is no longer a conflict
						if ledgerMap[curChangedIds].IsConflict == true {
							numConflicts = numConflicts - 1
							ledgerMap[curChangedIds].IsConflict = false
						}

						// Traverse down and change children conflict by removing curChangedIds
						conflictMapToAdd := ledgerMap[curChangedIds].ChildrenConflicts
						NewStack := make([]int, 0)
						allVisitedNew := make([]int, 0)
						NewStack = append(NewStack, curChangedIds)
						for len(NewStack) > 0 {
							curVertexNew := NewStack[len(NewStack)-1]
							allVisitedNew = append(allVisitedNew, curVertexNew)
							NewStack = NewStack[:len(NewStack)-1]
							_, okNew := ledgerMap[curVertexNew]
							if !okNew {
								continue
							}
							if exploredNestedSearchLedger[curVertexNew] != curChangedIds {
								exploredNestedSearchLedger[curVertexNew] = curChangedIds
								if curVertexNew != curChangedIds {
									// Finally delete curChangedIds
									delete(ledgerMap[curVertexNew].ChildrenConflicts, curChangedIds)
									for conflictToAdd := range conflictMapToAdd {
										ledgerMap[curVertexNew].ChildrenConflicts[conflictToAdd] = 1
									}
								}
								if !ledgerMap[curVertexNew].IsConflict {
									for nextVertexNew := range ledgerMap[curVertexNew].Parents {
										NewStack = append(NewStack, nextVertexNew) // Probably do not go over already explored
									}
								}
							}
						}
						// Traverse up and change parent conflict by removing curChangedIds
						conflictMapToAdd = ledgerMap[curChangedIds].ParentsConflicts
						NewStack = make([]int, 0)
						NewStack = append(NewStack, curChangedIds)
						for len(NewStack) > 0 {
							curVertexNew := NewStack[len(NewStack)-1]
							allVisitedNew = append(allVisitedNew, curVertexNew)
							NewStack = NewStack[:len(NewStack)-1]
							if exploredNestedSearchLedger[curVertexNew] != -curChangedIds {
								exploredNestedSearchLedger[curVertexNew] = -curChangedIds
								if curVertexNew != curChangedIds {
									// Finally delete curChangedIds
									delete(ledgerMap[curVertexNew].ParentsConflicts, curChangedIds)
									for conflictToAdd := range conflictMapToAdd {
										ledgerMap[curVertexNew].ParentsConflicts[conflictToAdd] = 1
									}
								}
								if !ledgerMap[curVertexNew].IsConflict {
									for nextVertexNew := range ledgerMap[curVertexNew].Children {
										NewStack = append(NewStack, nextVertexNew) // Probably do not go over already explored
									}
								}
							}
						}
						for t := range allVisitedNew {
							exploredNestedSearchLedger[allVisitedNew[t]] = 0
						}
					}
				}
				for curOutputLabel := range ledgerMap[curVertex].OutputLabels {
					//delete(outputLabelsMapOwnerID, curOutputLabel)
					allOutputLabelsToDelete[curOutputLabel] = 1
				}

				// Remove children.field for parents of curVertex and update past cones (until conflicts) ChildrenConflicts
				StackToTraverseUp := make([]int, 0)
				for j := range ledgerMap[curVertex].Parents {
					_, ok2 := ledgerMap[j]
					if ok2 {
						delete(ledgerMap[j].Children, curVertex)
						StackToTraverseUp = append(StackToTraverseUp, j)
					}

				}
				allVisitedNew := []int{}
				conflictMapToRemove := ledgerMap[curVertex].ChildrenConflicts
				if ledgerMap[curVertex].IsConflict {
					conflictMapToRemove[curVertex] = 1
				}

				for len(StackToTraverseUp) > 0 {
					curVertexUp := StackToTraverseUp[len(StackToTraverseUp)-1]
					allVisitedNew = append(allVisitedNew, curVertexUp)
					StackToTraverseUp = StackToTraverseUp[:len(StackToTraverseUp)-1]
					if exploredNestedSearchLedger[curVertexUp] != curVertex {
						exploredNestedSearchLedger[curVertexUp] = curVertex
						for curConflictToRemove := range conflictMapToRemove {
							delete(ledgerMap[curVertexUp].ChildrenConflicts, curConflictToRemove)
						}
						if !ledgerMap[curVertexUp].IsConflict {
							for nextVertex := range ledgerMap[curVertexUp].Parents {
								StackToTraverseUp = append(StackToTraverseUp, nextVertex)
							}
						}
					}
				}
				for t := range allVisitedNew {
					exploredNestedSearchLedger[allVisitedNew[t]] = 0
				}

				// Finally delete the vertex
				if ledgerMap[curVertex].IsConflict == true {
					numConflicts = numConflicts - 1
				}
				delete(ledgerMap, curVertex) // Probably wrong, memory leaks
			}
			for t := range allVisited {
				exploredSearchLedger[allVisited[t]] = 0
			}
		}
	}
	// Cleaning two structures outputLabelsMapOwnerID and
	numDel := 0
	numDelUnspent := 0
	for curOutput := range allOutputLabelsToDelete {
		numDel = numDel + 1
		curIndex := outputLabelsMapOwnerID[curOutput].indexSlice
		toCopyOut := outputLabelsSliceOwnerID[len(outputLabelsSliceOwnerID)-numDel]
		outputLabelsSliceOwnerID[curIndex] = toCopyOut
		outputLabelsMapOwnerID[toCopyOut.OutputLabel].indexSlice = curIndex
		outputLabelsMapOwnerID[curOutput].indexSlice = -1

		if outputLabelsMapOwnerID[curOutput].indexUnspentSlice != -1 {
			numDelUnspent = numDelUnspent + 1
			curIndex = outputLabelsMapOwnerID[curOutput].indexUnspentSlice
			toCopyOut = unspentLabelsSlice[len(unspentLabelsSlice)-numDelUnspent]
			unspentLabelsSlice[curIndex] = toCopyOut
			outputLabelsMapOwnerID[toCopyOut.OutputLabel].indexUnspentSlice = curIndex
			outputLabelsMapOwnerID[curOutput].indexUnspentSlice = -1
		}
		curIndex = outputLabelsMapOwnerID[curOutput].indexSpentSlice
		if curIndex != -1 {
			//numDelSpent = numDelSpent + 1
			toCopyOut = unconfirmedSpentLabelsSlice[len(unconfirmedSpentLabelsSlice)-1]
			unconfirmedSpentLabelsSlice[curIndex] = toCopyOut
			unconfirmedSpentLabelsSlice = unconfirmedSpentLabelsSlice[:len(unconfirmedSpentLabelsSlice)-1]
			if curIndex < len(unconfirmedSpentLabelsSlice) {
				outputLabelsMapOwnerID[toCopyOut.OutputLabel].indexSpentSlice = curIndex
			} else {
				outputLabelsMapOwnerID[toCopyOut.OutputLabel].indexSpentSlice = -1
			}

			outputLabelsMapOwnerID[curOutput].indexSpentSlice = -1
		}

		delete(outputLabelsMapOwnerID, curOutput)
	}
	outputLabelsSliceOwnerID = outputLabelsSliceOwnerID[:len(outputLabelsSliceOwnerID)-numDel]
	unspentLabelsSlice = unspentLabelsSlice[:len(unspentLabelsSlice)-numDelUnspent]
	//unconfirmedSpentLabelsSlice = unconfirmedSpentLabelsSlice[:len(unconfirmedSpentLabelsSlice)-numDelSpent]
}
