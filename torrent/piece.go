package torrent

import (
	"crypto/sha1"
	"fmt"
)

func ValidatePiece(pieceData []byte, expectedHash [20]byte) bool {
	actualHash := sha1.Sum(pieceData)
	fmt.Printf("Expected hash is %x\n", expectedHash)
	fmt.Printf("Actual hash is   %x\n", actualHash)
	return actualHash == expectedHash
}
