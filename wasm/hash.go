package main

import (
	"crypto/sha256"
	"strconv"
)

func hasLeadingZeros(hashStr string, zeros int) bool {
	for i := 0; i < zeros; i++ {
		if i >= len(hashStr) || hashStr[i] != '0' {
			return false
		}
	}
	return true
}

func hashToHexString(hash [32]byte) string {
	hashStr := ""
	for _, b := range hash {
		// Convert each byte to its 2-digit hex representation
		hexByte := strconv.FormatInt(int64(b), 16)
		if len(hexByte) == 1 {
			hexByte = "0" + hexByte
		}
		hashStr += hexByte
	}
	return hashStr
}

//export run
func run(input uint32, zeros uint32) uint32 {
	var counter uint32
	found := false

	for !found {
		counter++
		concatenatedStr := strconv.FormatUint(uint64(input), 10) + strconv.FormatUint(uint64(counter), 10)
		hash := sha256.Sum256([]byte(concatenatedStr))
		hashStr := hashToHexString(hash)

		if hasLeadingZeros(hashStr, int(zeros)) {
			return counter
		}
	}
	return 0
}

func main() {
}
