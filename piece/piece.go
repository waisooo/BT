package piece

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/anthony/BT/message"
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
	Client     *message.Client
	BlockData  []byte
	Downloaded int
	Requested  int
	Backlog    int
}

type PieceResult struct {
	Index int
	Data  []byte
}

// TryDownloadPiece attempts to download the specified piece from the given peer client.
//
// It returns the piece data if the download is successful, or an error if the download fails.
func TryDownloadPiece(client *message.Client, pw PieceWork) ([]byte, error) {
	if !hasPiece(client.Bitfield, pw.Index) {
		return nil, fmt.Errorf("Peer does not have piece %d", pw.Index)
	}

	state := PieceProgress{
		Client:    client,
		BlockData: make([]byte, pw.PieceSize),
	}

	// Send an interested message to the peer to indicate that we want to download pieces from it
	client.SendInterested()

	if client.Choked {
		client.SendUnchoke()
	}

	blockSize := MaxBlockSize
	if pw.PieceSize < blockSize {
		blockSize = pw.PieceSize
	}

	client.Conn.SetDeadline(time.Now().Add(15 * time.Second))
	defer client.Conn.SetDeadline(time.Time{})

	for state.Downloaded < pw.PieceSize {
		// Fill the pipeline with requests, up to the maximum allowed
		for !client.Choked && state.Backlog < MaxPipelineRequests && state.Requested < pw.PieceSize {
			if pw.PieceSize-state.Requested < blockSize {
				blockSize = pw.PieceSize - state.Requested
			}

			client.SendRequest(pw.Index, state.Requested, blockSize)

			state.Requested += blockSize
			state.Backlog++
		}

		if client.Choked {
			client.SendUnchoke()
		}

		err := readMessage(client, &state)
		if err != nil {
			// Sometimes the client may send a piece with the incorrect index.
			// In this case, we can ignore the piece and request it again later
			if err.Error() == "Index not matching" {
				continue
			}

			return nil, err
		}
	}

	// Verify the piece hash matches the expected hash from the torrent file
	pieceHash := sha1.Sum(state.BlockData)
	if !bytes.Equal(pieceHash[:], pw.PieceHash[:]) {
		return nil, fmt.Errorf("Hash mismatch for piece %d", pw.Index)
	}

	// Notify the peer that we have successfully downloaded the piece
	client.SendHave(pw.Index)

	return state.BlockData, nil
}

/////////////////////////////// Helper Functions /////////////////////////////////

// readMessage reads a message from the peer client, and updates the state of the client and piece download progress accordingly.
//
// It returns an error if there is an issue reading the message or if the message is invalid.
func readMessage(client *message.Client, state *PieceProgress) error {
	msg, err := message.RecieveMessage(client.Conn)

	if err != nil {
		return err
	}

	switch msg.Id {
	case message.Choke:
		state.Client.Choked = true
	case message.Unchoke:
		state.Client.Choked = false
	case message.Bitfield:
		state.Client.Bitfield = msg.Payload
	case message.Have:
		pieceIndex := binary.BigEndian.Uint32(msg.Payload[0:4])
		setPiece(state.Client.Bitfield, int(pieceIndex))
	case message.Piece:
		index := binary.BigEndian.Uint32(msg.Payload[0:4])
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])

		// Check that the piece index in the message matches the expected piece index for this download
		if int(index) != state.Index {
			return errors.New("Index not matching")
		}

		// Append the received block to the piece's block data
		n := copy(state.BlockData[begin:], msg.Payload[8:])

		state.Downloaded += n
		if n != 0 {
			state.Backlog--
		}
	}

	return nil
}

// hasPiece checks if the peer has the specified piece based on the peer's bitfield.
func hasPiece(bf []byte, index int) bool {
	if bf == nil {
		return false
	}

	byteIndex := index / 8
	bitIndex := index % 8

	return bf[byteIndex]&(1<<(7-bitIndex)) != 0
}

// setPiece sets the bit corresponding to the specified piece index in the
// given bitfield to indicate that they have that piece.
func setPiece(bf []byte, index int) {
	byteIndex := index / 8
	bitIndex := index % 8

	bf[byteIndex] |= 1 << (7 - bitIndex)
}
