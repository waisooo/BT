package torrent

import (
	"crypto/sha1"
	"fmt"
	"time"
)

const MaxBlockSize = 16384
const MaxPipelineRequests = 5

type PieceWork struct {
	Index            int
	PieceHash        [20]byte
	PieceSize        int
	DownloadAttempts int // Number of download attempts that have been made
}

type PieceProgress struct {
	Index      int
	Client     *Client
	BlockData  []byte
	Downloaded int
	Requested  int
	Backlog    int
}

type PieceResult struct {
	Index int
	Data  []byte
}

func CalculatePieceSize(fileLength int, pieceLength int, index int) int {
	if (index+1)*pieceLength > fileLength {
		return fileLength - (index * pieceLength)
	}

	return pieceLength
}

func SetPiece(bf []byte, index int) {
	byteIndex := index / 8
	bitIndex := index % 8

	bf[byteIndex] |= 1 << (7 - bitIndex)
}

func TryDownloadPiece(client *Client, pw *PieceWork) (*PieceResult, error) {
	if !hasPiece(client.Bitfield, pw.Index) {
		return &PieceResult{}, nil
	}

	state := PieceProgress{
		Client:    client,
		BlockData: make([]byte, pw.PieceSize),
	}

	client.lock.Lock()
	fmt.Println("Attempting to download piece", pw.Index)
	SendInterested(client)

	fmt.Println("Send interested")
	if client.Choked {
		SendUnchoke(client)
	}

	blockSize := MaxBlockSize
	if pw.PieceSize < blockSize {
		blockSize = pw.PieceSize
	}

	client.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer client.Conn.SetDeadline(time.Time{})

	for state.Downloaded < pw.PieceSize {
		if !client.Choked {
			// Fill the pipeline with requests, up to the maximum allowed
			for state.Backlog < MaxPipelineRequests && state.Requested < pw.PieceSize {
				if pw.PieceSize-state.Requested < blockSize {
					blockSize = pw.PieceSize - state.Requested
				}

				SendRequest(client, pw.Index, state.Requested, blockSize)

				state.Requested += blockSize
				state.Backlog++
			}
		}

		err := ReadMessage(&state)
		if err != nil {
			return &PieceResult{}, err
		}
	}

	result := PieceResult{
		Index: pw.Index,
		Data:  state.BlockData,
	}

	client.lock.Unlock()

	return &result, nil
}

func ValidatePiece(piece *PieceResult, pw *PieceWork) bool {
	// The piece has not been fully downloaded
	if len(piece.Data) != pw.PieceSize {
		return false
	}

	// Validate the piece hash
	return sha1.Sum(piece.Data) == pw.PieceHash
}

// ////////////////////////////// Helper Functions /////////////////////////////////
func hasPiece(bf []byte, index int) bool {
	if bf == nil {
		return false
	}

	byteIndex := index / 8
	bitIndex := index % 8

	return bf[byteIndex]&(1<<(7-bitIndex)) != 0
}
