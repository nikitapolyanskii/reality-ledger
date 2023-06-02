package main

import (
	"crypto/sha256"
	"fmt"
)

// Abs returns the absolute value of a float64 number.
func Abs(f float64) float64 {
	return -min(f, -f)
}

// min returns the minimum value between two float64 numbers.
func min(a float64, b float64) float64 {
	if a > b {
		return b
	} else {
		return a
	}
}

// hash calculates the SHA256 hash of a given string and returns it as a hexadecimal string.
func hash(str string) string {
	sum := sha256.Sum256([]byte(str)) // Calculate the SHA256 hash of the input string
	return fmt.Sprintf("%x", sum)     // Convert the hash to a hexadecimal string representation
}
