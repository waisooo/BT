package messages

import (
	"encoding/binary"
	"io"
	"net"
)

// Message ids for peer communication
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

func (c *Client) SendHave(pieceIndex int) {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))

	sendMessage(c.Conn, &Message{Id: Have, Payload: payload})
}

func (c *Client) SendInterested() {
	sendMessage(c.Conn, &Message{Id: Interested})
}

func (c *Client) SendUnchoke() {
	sendMessage(c.Conn, &Message{Id: Unchoke})
}

func (c *Client) SendChoke() {
	sendMessage(c.Conn, &Message{Id: Choke})
}

func (c *Client) SendNotInterested() {
	sendMessage(c.Conn, &Message{Id: NotInterested})
}

func (c *Client) SendRequest(index int, begin int, length int) {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	sendMessage(c.Conn, &Message{Id: Request, Payload: payload})
}

func RecieveMessage(conn net.Conn) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := conn.Read(lengthBuf)
	if err != nil {
		return nil, err
	}

	// Message length
	msgLen := binary.BigEndian.Uint32(lengthBuf)

	// Keep-alive message
	if msgLen == 0 {
		return &Message{}, nil
	}

	// Message ID
	id := make([]byte, 1)
	_, err = conn.Read(id)
	if err != nil {
		return nil, err
	}

	// Payload
	payload := make([]byte, msgLen-1)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return nil, err
	}

	peerMsg := Message{
		Id:      id[0],
		Payload: payload,
	}

	return &peerMsg, nil
}

func sendMessage(conn net.Conn, msg *Message) {
	buf := make([]byte, 4)

	if msg == nil {
		binary.BigEndian.PutUint32(buf, 0)
		conn.Write(buf)
		return
	}

	binary.BigEndian.PutUint32(buf, uint32(1+len(msg.Payload)))
	conn.Write(buf)
	conn.Write([]byte{msg.Id})
	if msg.Payload != nil {
		conn.Write(msg.Payload)
	}

}
