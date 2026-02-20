package message

import (
	"fmt"

	"github.com/anthony/BT/bencode"
)

const (
	metdataRequest   = 0
	metadataResponse = 1
	metadataReject   = 2
)

type extensionMessage struct {
	M struct {
		Metadata int `mapstructure:"metadata"`
	} `mapstructure:"m"`

	MetadataSize int `mapstructure:"metadata_size"`
}

type metadataMessage struct {
	MessageType int `mapstructure:"msg_type"`
	Piece       int `mapstructure:"piece"`
	TotalSize   int `mapstructure:"total_size"`
}

func (c *Client) ExtendedPeerHandshake() error {
	// Read response from peer
	resp, err := c.RecieveMessage()
	if err != nil {
		return err
	}

	messageId := resp.Id
	for i := 0; i < 50 && messageId != Extension; i++ {
		resp, err = c.RecieveMessage()
		messageId = resp.Id
		if err != nil {
			return err
		}
	}

	if messageId != Extension {
		return fmt.Errorf("Extension message was not sent")
	}

	// Check if the peer supports the extension handshake
	extMsgId := resp.Payload[0]
	payload := resp.Payload[1:]
	if extMsgId != 0 {
		return fmt.Errorf("Peer does not support the extension handshake")
	}

	var extHandshakeMsg extensionMessage
	err = bencode.Decode(payload, &extHandshakeMsg)
	if err != nil {
		return fmt.Errorf("Error decoding extension handshake message: %w", err)
	}

	c.MetadataExtension.MessageID = extHandshakeMsg.M.Metadata
	c.MetadataExtension.MetadataSize = extHandshakeMsg.MetadataSize

	return nil
}

func (c *Client) RequestMetadata(infoHash [20]byte) ([]byte, error) {
	if c.MetadataExtension.MessageID == 0 || c.MetadataExtension.MetadataSize == 0 {
		return nil, fmt.Errorf("Peer does not support metadata exchange")
	}

	metadataPieceSize := 16 * 1024 // 16kiB per piece as specified in BEP_9
	metadata := make([]byte, c.MetadataExtension.MetadataSize)

	recieved := 0
	for piece := 0; recieved < c.MetadataExtension.MetadataSize; piece++ {
		err := c.SendRequestMetadata(piece)
		if err != nil {
			return nil, fmt.Errorf("Error sending metadata request: %w", err)
		}

		resp, err := c.RecieveMessage()
		if err != nil {
			return nil, err
		}

		if resp.Id != Extension {
			return nil, fmt.Errorf("Expected extension message in response to metadata request")
		}

		respPayload := resp.Payload[1:]

		// Extract the bencoded dictionary from the response
		var dictResp []byte
		for i := 1; i < len(respPayload); i++ {
			if string(respPayload[i-1:i+1]) == "ee" {
				dictResp = respPayload[:i+1]
				break
			}
		}

		var extResp metadataMessage
		err = bencode.Decode(dictResp, &extResp)
		if err != nil {
			return nil, fmt.Errorf("Error decoding metadata response: %w", err)
		}

		switch extResp.MessageType {
		case metadataResponse:
			// Check the size of the piece is correct. Unless it is the last piece.
			if extResp.TotalSize != metadataPieceSize {
				return nil, fmt.Errorf("Metadata piece size mismatch")
			}

			metadata = append(metadata, resp.Payload[len(dictResp):]...)

			recieved += (len(resp.Payload) - len(dictResp) - 1)
		case metadataReject:
			return nil, fmt.Errorf("Peer rejected metadata request")
		default:
			return nil, fmt.Errorf("Reuqested metadata piece but got unknown message id")
		}
	}

	return metadata, nil
}
