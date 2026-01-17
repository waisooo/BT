package pieces

import (
	messages "bittorrent/messages"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"time"
)

const MaxBlockSize = 16384
const MaxPipelineRequests = 5

type PieceWork struct {
	Index     int
	PieceHash [20]byte
	PieceSize int
}

type PieceProgress struct {
	Index      int
	Client     *messages.Client
	BlockData  []byte
	Downloaded int
	Requested  int
	Backlog    int
}

type PieceResult struct {
	Index int
	Data  []byte
}

func TryDownloadPiece(client *messages.Client, pw *PieceWork) (*PieceResult, error) {
	if !HasPiece(client.Bitfield, pw.Index) {
		return &PieceResult{}, fmt.Errorf("Peer does not have piece %d", pw.Index)
	}

	state := PieceProgress{
		Client:    client,
		BlockData: make([]byte, pw.PieceSize),
	}

	fmt.Println("Attempting to download piece", pw.Index, "from peer", client.PeerIP)
	client.SendInterested()

	if client.Choked {
		client.SendUnchoke()
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

				client.SendRequest(pw.Index, state.Requested, blockSize)

				state.Requested += blockSize
				state.Backlog++
			}
		}

		err := readMessage(client, &state)
		if err != nil {
			return &PieceResult{}, err
		}
	}

	result := PieceResult{
		Index: pw.Index,
		Data:  state.BlockData,
	}

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

func HasPiece(bf []byte, index int) bool {
	if bf == nil {
		return false
	}

	byteIndex := index / 8
	bitIndex := index % 8

	return bf[byteIndex]&(1<<(7-bitIndex)) != 0
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

/////////////////////////////// Helper Functions /////////////////////////////////

func readMessage(client *messages.Client, state *PieceProgress) error {
	msg, err := messages.RecieveMessage(client.Conn)

	if err != nil {
		return err
	}

	switch msg.Id {
	case messages.Choke:
		state.Client.Choked = true
	case messages.Unchoke:
		state.Client.Choked = false
	case messages.Bitfield:
		state.Client.Bitfield = msg.Payload
	case messages.Have:
		pieceIndex := binary.BigEndian.Uint32(msg.Payload[0:4])
		SetPiece(state.Client.Bitfield, int(pieceIndex))
	case messages.Piece:
		// Append the received block to the piece's block data
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		copy(state.BlockData[begin:], msg.Payload[8:])

		state.Downloaded += len(msg.Payload) - 8
		state.Backlog--
	}

	return nil
}
