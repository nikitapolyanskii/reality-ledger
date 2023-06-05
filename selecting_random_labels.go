package main

import (
	"math/rand"
	"strconv"
)

// TraversePastLabels performs a Depth-First Search (DFS) to explore all
// connected vertices from a given starting vertex. It marks any conflicting labels
// in the Labels map provided.
func TraversePastLabels(startVertexID int, conflictingLabels map[string]int) {
	visitedVertices := make([]int, 0) // Slice to store all visited vertices

	// Initialize the Stack with the startVertexID as the starting point of the DFS
	DFSStack := []int{startVertexID}

	// Perform DFS to explore all connected vertices
	for len(DFSStack) > 0 {
		// Pop the last vertex from the stack
		currentVertex := DFSStack[len(DFSStack)-1]
		DFSStack = DFSStack[:len(DFSStack)-1]

		visitedVertices = append(visitedVertices, currentVertex) // Mark the current vertex as visited

		// If the current vertex has not been explored before
		if exploredSearchLedger[currentVertex] != startVertexID {
			exploredSearchLedger[currentVertex] = startVertexID
			// Add all input labels of the current vertex to the conflictingLabels map
			for label := range ledgerMap[currentVertex].InputLabels {
				conflictingLabels[label] = 1
			}
			// Add all parent vertices to the DFS stack
			for parentVertex := range ledgerMap[currentVertex].Parents {
				DFSStack = append(DFSStack, parentVertex)
			}
		}
	}

	// Clear the exploredSearchLedger for the visited vertices
	for _, vertex := range visitedVertices {
		exploredSearchLedger[vertex] = 0
	}
}

// GenerateRandomInputsAndOutputs generates random numbers for inputs and outputs
// based on a distribution. It uses a weighted random selection where weights are
// specified in the inputOutputDistribution.
func GenerateRandomInputsAndOutputs() (int, int) {
	randomValue := rand.Float64() // Generate a random float number between 0 and 1
	currentIndex := 0

	// Iterate over the inputOutputDistribution until it finds the first item
	// where the weight is greater than or equal to the random number
	for inputOutputDistribution[currentIndex].weight < randomValue {
		currentIndex++
	}

	// Returns the inputs and outputs of the selected item from inputOutputDistribution
	return inputOutputDistribution[currentIndex].inputs, inputOutputDistribution[currentIndex].outputs
}

// GenerateLabels creates new labels for the input and output vertices of a graph based
// on a conflict probability. It also adds new labels to the input and output label maps passed as arguments.
func GenerateLabels(inputLabels, outputLabels map[string]int, conflictProbability float64) {

	// Generate a random number of input and output labels
	inputLabelCount, outputLabelCount := GenerateRandomInputsAndOutputs()

	// Generate input labels
	currentInputLabelCount := 0
	allPastLabels := map[string]int{}

	// Pick a random existing spent output if the random value is less than the conflict probability
	if len(unconfirmedSpentLabelsSlice) > 0 && rand.Float64() < conflictProbability {
		spentLabelIndex := rand.Intn(len(unconfirmedSpentLabelsSlice))
		selectedLabel := unconfirmedSpentLabelsSlice[spentLabelIndex].OutputLabel
		parentVertex := outputLabelsMapOwnerID[selectedLabel].ID

		allPastLabels[selectedLabel] = 1
		TraversePastLabels(parentVertex, allPastLabels)

		currentInputLabelCount++
		inputLabels[selectedLabel] = 1
	}

	// Limit on the number of attempts to create a non-conflicting input label
	badAttempts := 0

	// Continue generating input labels until we reach the desired number
	for currentInputLabelCount < inputLabelCount {
		if badAttempts > numBadAttemptsInputLabel {
			inputLabelCount--
			badAttempts = 0
		}

		unspentLabelIndex := rand.Intn(len(unspentLabelsSlice))
		selectedInputLabel := unspentLabelsSlice[unspentLabelIndex]
		parentVertex := outputLabelsMapOwnerID[selectedInputLabel.OutputLabel].ID

		pastLabelsOfCurrentVertex := map[string]int{}
		pastLabelsOfCurrentVertex[selectedInputLabel.OutputLabel] = 1
		TraversePastLabels(parentVertex, pastLabelsOfCurrentVertex)

		isLabelValid := true

		// Check if current label conflicts with any past labels
		for label := range pastLabelsOfCurrentVertex {
			if _, exists := allPastLabels[label]; exists {
				badAttempts++
				isLabelValid = false
				break
			}
		}

		if !isLabelValid {
			continue
		}

		currentInputLabelCount++
		for label := range pastLabelsOfCurrentVertex {
			allPastLabels[label] = 1
		}
		inputLabels[selectedInputLabel.OutputLabel] = 1
	}

	// Generate output labels
	for j := 0; j < outputLabelCount; j++ {
		outputLabel := hash(strconv.Itoa(rand.Int()))
		outputLabels[outputLabel] = j
	}
}
