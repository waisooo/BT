package message

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/anthony/BT/bencode"
)

// Message ids for peer communication as specified in the bitTorrent protocol specification. (BEP_3)
const (
	Choke         = 0
	Unchoke       = 1
	Interested    = 2
	NotInterested = 3
	Have          = 4
	Bitfield      = 5
	Request       = 6
	Piece         = 7
	Cancel        = 8
	Extension     = 20 // Extended message id as specified in BEP_10
)

type Message struct {
	Id      byte
	Payload []byte
}

type Client struct {
	Conn              net.Conn
	Ip                string
	Bitfield          []byte
	IsChoked          bool
	SupportsExtension bool
	MetadataExtension struct {
		MessageID    int
		MetadataSize int
	}
}

// Methods for sending messages with the specific message ids and payloads as specified in the BitTorrent protocol specification.

func (c *Client) SendHave(pieceIndex int) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))

	return c.sendMessage(Message{Id: Have, Payload: payload})
}

func (c *Client) SendInterested() error {
	return c.sendMessage(Message{Id: Interested})
}

func (c *Client) SendUnchoke() error {
	return c.sendMessage(Message{Id: Unchoke})
}

func (c *Client) SendChoke() error {
	return c.sendMessage(Message{Id: Choke})
}

func (c *Client) SendNotInterested() error {
	return c.sendMessage(Message{Id: NotInterested})
}

func (c *Client) SendRequestBlock(index int, begin int, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return c.sendMessage(Message{Id: Request, Payload: payload})
}

func (c *Client) SendRequestMetadata(piece int) error {
	requestDict := map[string]int{
		"msg_type": metdataRequest,
		"piece":    piece,
	}

	payload, err := bencode.Encode(requestDict)
	if err != nil {
		return err
	}

	return c.sendMessage(Message{Id: Extension, Payload: payload})
}

// Recieve a message from the peer and if successful,
// return a struct containing the message id and payload,
// otherwise return any error encountered.
func (c *Client) RecieveMessage() (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := c.Conn.Read(lengthBuf)
	if err != nil {
		return nil, err
	}

	// Message length
	msgLen := binary.BigEndian.Uint32(lengthBuf)

	// Keep-alive message
	if msgLen == 0 {
		return nil, nil
	}

	// Message ID
	id := make([]byte, 1)
	_, err = c.Conn.Read(id)
	if err != nil {
		return nil, err
	}

	// Payload
	payload := make([]byte, msgLen-1)
	_, err = io.ReadFull(c.Conn, payload)
	if err != nil {
		return nil, err
	}

	resp := Message{
		Id:      id[0],
		Payload: payload,
	}

	return &resp, nil
}

// readMessage reads a message from the peer client, and updates the state of the client and piece download progress accordingly.
//
// It returns an error if there is an issue reading the message or if the message is invalid.
func (c *Client) ReadMessage() error {
	msg, err := c.RecieveMessage()

	if err != nil {
		return err
	}

	switch msg.Id {
	case Choke:
		c.IsChoked = true
	case Unchoke:
		c.IsChoked = false
	case Bitfield:
		c.Bitfield = msg.Payload
	case Have:
		pieceIndex := binary.BigEndian.Uint32(msg.Payload[0:4])
		setPiece(c.Bitfield, int(pieceIndex))
	}

	return nil
}

///////////////////////////// Helper functions /////////////////////////////

// Helper function to send a message to the peer with the specified message id and payload.
func (c *Client) sendMessage(msg Message) error {
	// Construct payload that contains:
	// - 4 byte for message length
	// - 1 byte for message id
	// - payload of variable length
	// As specified in the BitTorrent protocol specification (BEP_3)
	buf := make([]byte, 5+len(msg.Payload))
	binary.BigEndian.PutUint32(buf, uint32(1+len(msg.Payload)))
	buf[4] = msg.Id
	copy(buf[5:], msg.Payload)

	_, err := c.Conn.Write(buf)
	if err != nil {
		return err
	}

	return nil
}

// setPiece sets the bit corresponding to the specified piece index in the
// given bitfield to indicate that they have that piece.
func setPiece(bf []byte, index int) {
	byteIndex := index / 8
	bitIndex := index % 8

	bf[byteIndex] |= 1 << (7 - bitIndex)
}
