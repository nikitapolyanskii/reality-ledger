package main

import (
	"os"
	"sort"
	"strconv"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

// function to draw the DAG
func DrawDAG(fileName string, ledgerMap map[int]*Transaction, outputLabelsMapConsumerIDs map[string]map[int]int, threshold float64, idGenesis int) {
	g := graph.New(graph.IntHash, graph.Directed())

	allVisited := make([]int, 0)
	// create exploredSearchLedger of size len(ledgerMap)

	exploredSearchLedger := make(map[int]int, len(ledgerMap))
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

	// clean exploredSearchLedger here
	for t := range allVisited {
		exploredSearchLedger[allVisited[t]] = 0
	}

	file, _ := os.Create("./" + fileName + ".gv")
	_ = draw.DOT(g, file)
}
