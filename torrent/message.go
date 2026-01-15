package torrent

import (
	"encoding/binary"
	"io"
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

func ReadMessage(state *PieceProgress) error {
	msg, err := recieveMessage(state.Client.Conn)

	if err != nil {
		return err
	}

	switch msg.Id {
	case Choke:
		state.Client.Choked = true
	case Unchoke:
		state.Client.Choked = false
	case Bitfield:
		state.Client.Bitfield = msg.Payload
	case Piece:
		// Append the received block to the piece's block data
		begin := binary.BigEndian.Uint32(msg.Payload[4:8])

		copy(state.BlockData[begin:], msg.Payload[8:])
		setPiece(state.Client.Bitfield, state.Index)
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
