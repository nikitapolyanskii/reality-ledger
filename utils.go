package main

import (
	"os"
	"sort"
	"strconv"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

// DrawDAG generates a Directed Acyclic Graph (DAG) visualization in Graphviz DOT format.
// It takes the following parameters:
// - filename: The name of the output file without the extension.
// - transactions: A map of transaction IDs to Transaction objects.
// - outputConsumers: A map that contains UTXO and a map of all transaction IDs consuming this UTXO.
// - threshold: A threshold value used for determining the color of vertices in the graph.
// - genesisID: The ID of the genesis transaction.
func DrawDAG(filename string, transactions map[int]*Transaction, outputConsumers map[string]map[int]int, threshold float64, genesisID int) {
	g := graph.New(graph.IntHash, graph.Directed())

	allVisited := make([]int, 0)

	// Create exploredSearchLedger of size len(transactions)
	exploredSearchLedger := make(map[int]int, len(transactions))

	// Missing: Traverse down till conflicts and mark the past conflicts BFS
	stack := make([]int, 0)
	stack = append(stack, genesisID)

	strGenesisID := "TX: " + strconv.Itoa(genesisID)
	if transactions[genesisID].IsConflict {
		_ = g.AddVertex(genesisID, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", strGenesisID), graph.VertexAttribute("color", "blue"))
	}

	for len(stack) > 0 {
		curVertex := stack[len(stack)-1]
		allVisited = append(allVisited, curVertex)
		stack = stack[:len(stack)-1]
		if exploredSearchLedger[curVertex] != genesisID {
			exploredSearchLedger[curVertex] = genesisID
			keys := make([]string, 0)
			for k := range transactions[curVertex].OutputLabels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, out := range keys {
				if len(outputConsumers[out]) >= 1 {
					curNodeInt := 0
					curFactor := 1
					for p := 1; p < len(out); p++ {
						curByte := int(out[p])
						curNodeInt = curNodeInt + curFactor*curByte
						curFactor = curFactor * 256
					}
					_ = g.AddVertex(curNodeInt, graph.VertexAttribute("label", out[0:5]))
					_ = g.AddEdge(curVertex, curNodeInt)
					consumerIDs := make([]int, 0)
					for k2 := range outputConsumers[out] {
						consumerIDs = append(consumerIDs, k2)
					}
					sort.Ints(consumerIDs)
					for _, consumer := range consumerIDs {
						strConsumer := "TX: " + strconv.Itoa(consumer)
						if transactions[consumer].IsConflict {
							_ = g.AddVertex(consumer, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", strConsumer), graph.VertexAttribute("color", "red"))
						} else {
							if transactions[consumer].Weight > threshold {
								_ = g.AddVertex(consumer, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", strConsumer), graph.VertexAttribute("color", "blue"))
							} else {
								_ = g.AddVertex(consumer, graph.VertexAttribute("shape", "polygon"), graph.VertexAttribute("label", strConsumer))
							}
						}
						_ = g.AddEdge(curNodeInt, consumer)
					}
				}
			}
			childIDs := make([]int, 0)
			for k := range transactions[curVertex].Children {
				childIDs = append(childIDs, k)
			}
			sort.Ints(childIDs)
			for _, nextVertex := range childIDs {
				stack = append(stack, nextVertex)
			}
		}
	}

	// Clean exploredSearchLedger here
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}

	file, _ := os.Create("./" + filename + ".gv")
	_ = draw.DOT(g, file)
}
