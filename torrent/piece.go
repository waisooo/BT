package torrent

import "crypto/sha1"

func ValidatePiece(pieceData []byte, expectedHash [20]byte) bool {
	actualHash := sha1.Sum(pieceData)
	return actualHash == expectedHash
}
