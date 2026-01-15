package torrent

import (
	"crypto/sha1"
	"fmt"
	"time"
)

type PieceProgress struct {
	Client    *Client
	Hash      [20]byte
	Index     int
	Received  int
	Total     int
	Requested int
	BlockData []byte
}

func ConstructPieceProgress(client *Client, torrentFile TorrentFile) *PieceProgress {
	pieceLength := calculatePieceLength(torrentFile.Info.Length, torrentFile.Info.PieceLength, 0)
	return &PieceProgress{
		Client:    client,
		Hash:      torrentFile.PiecesHash[0],
		Index:     0,
		Received:  0,
		Total:     pieceLength,
		Requested: 0,
		BlockData: make([]byte, pieceLength),
	}
}

func calculatePieceLength(fileLength int, pieceLength int, index int) int {
	if (index+1)*pieceLength > fileLength {
		return fileLength - (index * pieceLength)
	}

	return pieceLength
}

func HasPiece(bf []byte, index int) bool {
	byteIndex := index / 8
	bitIndex := index % 8

	return bf[byteIndex]&(1<<(7-bitIndex)) != 0
}

func SetPiece(bf []byte, index int) {
	byteIndex := index / 8
	bitIndex := index % 8

	bf[byteIndex] |= 1 << (7 - bitIndex)
}

func TryDownloadPiece(state *PieceProgress) (*PieceProgress, error) {
	// Read bitfield message from peer
	err := ReadMessage(state)
	if err != nil {
		return state, err
	}

	if !HasPiece(state.Client.Bitfield, state.Index) {
		return state, fmt.Errorf("Peer does not have piece %d", state.Index)
	}

	SendInterested(state.Client)
	err = ReadMessage(state)
	if err != nil {
		return state, err
	}

	if state.Client.Choked {
		SendUnchoke(state.Client)
	}

	blockSize := MaxBlockSize
	if state.Total < blockSize {
		blockSize = state.Total
	}

	state.Client.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer state.Client.Conn.SetDeadline(time.Time{})

	for state.Received < state.Total {
		if !state.Client.Choked {
			if state.Total-state.Received < blockSize {
				blockSize = state.Total - state.Received
			}

			SendRequest(state.Client, state.Index, state.Received, blockSize)

			err = ReadMessage(state)

			if err != nil {
				return state, err
			}

			state.Received += blockSize
		}

	}

	return state, nil
}

func ValidatePiece(pieceData []byte, expectedHash [20]byte) bool {
	actualHash := sha1.Sum(pieceData)
	return actualHash == expectedHash
}
