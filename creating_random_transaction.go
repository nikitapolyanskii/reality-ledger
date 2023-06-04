package main

import (
	"math/rand"
	"strconv"
)

// compute bad labels in the past cone
func PastLabels(id int, Labels map[string]int) {

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

func GetRandNumInputOutput() (int, int) {
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

func CreateLabels(curInputLabels map[string]int, curOutputLabels map[string]int, probabilityConflict float64) {
	// number of inputs and outputs are random
	numInputLabels, numOutputLabels := GetRandNumInputOutput()
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
