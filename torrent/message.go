package torrent

import (
	"encoding/binary"
	"fmt"
	"net"
)

// Meessage ids for peer communication
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
)

type Message struct {
	Id      byte
	Payload []byte
}

type Client struct {
	PeerIP   string
	Conn     net.Conn
	Bitfield []byte
	Choked   bool
}

func ReadMessage(pieceProgress *PieceProgress) error {
	msg, err := recieveMessage(pieceProgress.Client.Conn)

	if err != nil {
		return err
	}

	fmt.Println("Id is ", msg.Id)

	switch msg.Id {
	case Choke:
		pieceProgress.Client.Choked = true
	case Unchoke:
		pieceProgress.Client.Choked = false
	case Bitfield:
		pieceProgress.Client.Bitfield = msg.Payload
	case Piece:
		// Append the received block to the piece's block data
		pieceProgress.BlockData = append(pieceProgress.BlockData, msg.Payload[8:]...)
		setPiece(pieceProgress.Client.Bitfield, pieceProgress.Index)
	}

	return nil
}

func SendInterested(c *Client) {
	sendMessage(c.Conn, &Message{Id: Interested})
}

func SendUnchoke(c *Client) {
	sendMessage(c.Conn, &Message{Id: Unchoke})
}

func SendChoke(c *Client) {
	sendMessage(c.Conn, &Message{Id: Choke})
}

func SendNotInterested(c *Client) {
	sendMessage(c.Conn, &Message{Id: NotInterested})
}

func SendRequest(c *Client, index int, begin int, length int) {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	sendMessage(c.Conn, &Message{Id: Request, Payload: payload})
}

//////////////////////////////// Helper Functions /////////////////////////////////

func recieveMessage(conn net.Conn) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := conn.Read(lengthBuf)
	if err != nil {
		return nil, err
	}

	// Message length
	msgLen := binary.BigEndian.Uint32(lengthBuf[0:4])

	// Payload
	payload := make([]byte, msgLen)
	_, err = conn.Read(payload)
	if err != nil {
		return nil, err
	}

	peerMsg := Message{}
	if msgLen > 0 {
		peerMsg.Id = payload[0]
	}

	if msgLen > 1 {
		peerMsg.Payload = payload[1:]
	}

	return &peerMsg, nil
}

func sendMessage(conn net.Conn, msg *Message) {
	var buf []byte
	if msg == nil {
		buf = make([]byte, 4)
	} else {
		buf = make([]byte, 5+len(msg.Payload))
		binary.BigEndian.PutUint32(buf[0:4], uint32(1+len(msg.Payload)))
		buf[4] = msg.Id
		copy(buf[5:], msg.Payload)
	}

	_, err := conn.Write(buf)
	if err != nil {
		panic(err)
	}
}
