package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/woodywood117/stopwatch"
	"golang.org/x/exp/maps"
)

// main global parameters of DAG

var globalSeed int64 = 100
var ifDrawOut = true

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

// function that draws DAG
func DrawDAG(name string) {
	g := graph.New(graph.IntHash, graph.Directed())

	allVisited := make([]int, 0)
	//Missing: Traverse down till conflicts and mark the past conflicts BFS
	Stack := make([]int, 0)
	Stack = append(Stack, idGenesis)

	strID := "TX: " + strconv.Itoa(idGenesis)
	if ledgerMap[idGenesis].IsConflict {
		_ = g.AddVertex(idGenesis, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", strID), graph.VertexAttribute("color", "blue"))
	}

	for len(Stack) > 0 {
		curVertex := Stack[len(Stack)-1]
		allVisited = append(allVisited, curVertex)
		Stack = Stack[:len(Stack)-1]
		if exploredSearchLedger[curVertex] != idGenesis {
			exploredSearchLedger[curVertex] = idGenesis
			keys := make([]string, 0)
			for k, _ := range ledgerMap[curVertex].OutputLabels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, out := range keys {
				if len(outputLabelsMapConsumerIDs[out]) >= 1 {
					curNodeInt := 0
					curFactor := 1
					for p := 1; p < len(out); p++ {
						curByte := int(out[p])
						curNodeInt = curNodeInt + curFactor*curByte
						curFactor = curFactor * 256
					}
					_ = g.AddVertex(curNodeInt, graph.VertexAttribute("label", out[0:5]))
					_ = g.AddEdge(curVertex, curNodeInt)
					keys2 := make([]int, 0)
					for k2, _ := range outputLabelsMapConsumerIDs[out] {
						keys2 = append(keys2, k2)
					}
					sort.Ints(keys2)
					for _, cons := range keys2 {
						strCONS := "TX: " + strconv.Itoa(cons)
						if ledgerMap[cons].IsConflict {
							_ = g.AddVertex(cons, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", strCONS), graph.VertexAttribute("color", "red"))
						} else {
							if ledgerMap[cons].Weight > threshold {
								_ = g.AddVertex(cons, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", strCONS), graph.VertexAttribute("color", "blue"))
							} else {
								_ = g.AddVertex(cons, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", strCONS))
							}
						}
						_ = g.AddEdge(curNodeInt, cons)
					}
				}
			}
			keys3 := make([]int, 0)
			for k, _ := range ledgerMap[curVertex].Children {
				keys3 = append(keys3, k)
			}
			sort.Ints(keys3)
			for _, nextVertex := range keys3 {
				Stack = append(Stack, nextVertex)
			}
		}
	}
	// clean exploredSearchLedger
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}

	file, _ := os.Create("./mygraph" + name + ".gv")
	_ = draw.DOT(g, file)
}

func DeleteRejectedTransactions() {
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
		// outputLabelsMapConsumerIDs[curLabel] = map[int]int{} Questinable
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
	probabilityConflict := 0.01
	numTransactionsStart := 40000000
	upBoundConflicts := 5000
	//go outLedgerPruneLimitConflict()
	GrowingLedgerPruningConflictsLimit(probabilityConflict, numTransactionsStart, upBoundConflicts)
	CleaningStructures()
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
		DeleteRejectedTransactions()
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