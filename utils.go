package main

import (
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"golang.org/x/exp/maps"
)

// AllParameters represents a collection of parameters used for a specific calculation.
type AllParameters struct {
	timestamp       float64 // The timestamp of the calculation
	numConflicts    int     // The number of conflicts encountered
	numTransactions int     // The total number of transactions processed
}

// DrawDAG function generates a Directed Acyclic Graph (DAG) visualization.
func DrawDAG(filename string) {
	// Create a new directed graph
	g := graph.New(graph.IntHash, graph.Directed())

	// Keep track of all visited vertices
	allVisited := make([]int, 0)

	// Stack for Depth First Search, initialized with genesis ID
	dfsStack := []int{idGenesis}

	// Dictionary to keep track of visited nodes during this DFS
	exploredSearchLedger := make(map[int]int, len(ledgerMap))

	// Add the genesis transaction to the graph
	genesisIDStr := "TX: " + strconv.Itoa(idGenesis)
	if ledgerMap[idGenesis].IsConflict {
		_ = g.AddVertex(idGenesis, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", genesisIDStr), graph.VertexAttribute("color", "blue"))
	}

	// Perform DFS
	for len(dfsStack) > 0 {
		// Pop the last transaction ID from the DFS stack
		currentVertex := dfsStack[len(dfsStack)-1]
		dfsStack = dfsStack[:len(dfsStack)-1]

		allVisited = append(allVisited, currentVertex)

		// If the current vertex has not been explored in this DFS, explore it
		if exploredSearchLedger[currentVertex] != idGenesis {
			exploredSearchLedger[currentVertex] = idGenesis

			// Get the output labels for currentVertex and sort them
			outputLabels := make([]string, 0)
			for outputLabel := range ledgerMap[currentVertex].OutputLabels {
				outputLabels = append(outputLabels, outputLabel)
			}
			sort.Strings(outputLabels)

			// Iterate over output labels
			for _, output := range outputLabels {
				if len(outputLabelsMapConsumerIDs[output]) >= 1 {
					// Compute a unique ID for the output node
					outputNodeID := 0
					factor := 1
					for i := 1; i < len(output); i++ {
						outputNodeID += factor * int(output[i])
						factor *= 256
					}

					// Add the output node and edge to the graph
					_ = g.AddVertex(outputNodeID, graph.VertexAttribute("label", output[0:5]))
					_ = g.AddEdge(currentVertex, outputNodeID)

					// Get sorted consumer IDs for this output label
					consumerIDs := make([]int, 0)
					for consumerID := range outputLabelsMapConsumerIDs[output] {
						consumerIDs = append(consumerIDs, consumerID)
					}
					sort.Ints(consumerIDs)

					// Iterate over the consumer IDs
					for _, consumerID := range consumerIDs {
						// Add the consumer node and edge to the graph
						consumerIDStr := "TX: " + strconv.Itoa(consumerID)
						if ledgerMap[consumerID].Weight > threshold {
							_ = g.AddVertex(consumerID, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", consumerIDStr), graph.VertexAttribute("color", "blue"))
						} else {
							if ledgerMap[consumerID].IsConflict {
								_ = g.AddVertex(consumerID, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", consumerIDStr), graph.VertexAttribute("color", "red"))
							} else {
								_ = g.AddVertex(consumerID, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", consumerIDStr))
							}
						}

						_ = g.AddEdge(outputNodeID, consumerID)
					}
				}
			}

			// Get sorted child IDs of currentVertex
			childrenIDs := make([]int, 0)
			for childID := range ledgerMap[currentVertex].Children {
				childrenIDs = append(childrenIDs, childID)
			}
			sort.Ints(childrenIDs)

			// Push the children of the current vertex to the stack for further traversal
			dfsStack = append(dfsStack, childrenIDs...)
		}
	}

	// Reset the exploredSearchLedger
	for _, vertex := range allVisited {
		exploredSearchLedger[vertex] = 0
	}

	// Create the output file and write the graph in DOT format
	file, _ := os.Create("./" + filename + ".gv")
	_ = draw.DOT(g, file)
}

// GetBranch traverses the ledger to retrieve a branch of transactions from a given starting ID.
// It takes an ID of a transaction and returns a map of transaction IDs representing the branch.
func GetBranch(id int) map[int]int {
	branch := map[int]int{} // Initialize an empty map to store the branch of transactions
	stack := make([]int, 0) // Create a stack to perform depth-first search traversal

	if ledgerMap[id].IsConflict {
		branch[id] = 1 // Add the starting ID to the branch if it is a conflict
	}

	for nextVertex := range ledgerMap[id].ParentsConflicts {
		stack = append(stack, nextVertex) // Add the parents with conflicts to the stack for traversal
	}

	allVisited := make([]int, 0) // Keep track of all visited vertices during traversal

	for len(stack) > 0 {
		curVertex := stack[len(stack)-1]
		allVisited = append(allVisited, curVertex)
		stack = stack[:len(stack)-1]

		if exploredSearchLedger[curVertex] != id {
			exploredSearchLedger[curVertex] = id
			branch[curVertex] = 1 // Add the current vertex to the branch

			for nextVertex := range ledgerMap[curVertex].ParentsConflicts {
				stack = append(stack, nextVertex) // Add the parents with conflicts to the stack for traversal
			}
		}
	}

	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0 // Reset the explored search ledger
	}

	return branch
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

func Analytics(curTimestamp time.Duration) {
	curNumTransactions := len(ledgerMap)
	curNumConflicts := numConflicts
	allAnalytics = append(allAnalytics, AllParameters{numConflicts: curNumConflicts, numTransactions: curNumTransactions, timestamp: curTimestamp.Seconds()})
}
