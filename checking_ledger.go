package main

import "fmt"

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
