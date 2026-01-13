package torrent

import (
	bencode "bittorrent/decode"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

const MaxBlockSize = 16384

type Peer struct {
	Complete   int
	Incomplete int
	Interval   int      // How often queries should be re-sent
	Peers      []string // IP address of all peers
}

type PieceProgress struct {
	Client    *Client
	Hash      [20]byte
	Index     int
	Received  int
	Total     int
	Requested int
	BlockData []byte
}

func DownloadBlockFromPeer(state *PieceProgress) (*PieceProgress, error) {
	// Read bitfield message from peer
	err := ReadMessage(state)
	if err != nil {
		return state, err
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

	fmt.Println("Piece size is ", state.Total)
	fmt.Println("Block size is ", blockSize)

	for state.Received < state.Total {
		if state.Total-state.Received < blockSize {
			blockSize = state.Total - state.Received
		}

		// Download all the blocks in the piece
		SendRequest(state.Client, state.Index, state.Index*blockSize, blockSize)

		err = ReadMessage(state)

		if err != nil {
			return state, err
		}

		state.Received += blockSize
		state.Index++
	}

	return state, nil
}

func HandShakePeer(conn net.Conn, infoHash [20]byte, peerId [20]byte) ([]byte, error) {
	// Create handshake message
	message := make([]byte, 68)

	message[0] = 19                                    // Length of protocol
	copy(message[1:20], []byte("BitTorrent protocol")) // Protocol identifier
	copy(message[28:48], infoHash[:])                  // Info hash
	copy(message[48:68], peerId[:])                    // Peer ID

	// Send initial handshake
	_, err := conn.Write(message)
	if err != nil {
		return nil, err
	}

	// Read handshake response from peer
	// It should be in the same format as the sent message
	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func RequestPeers(torrentFile *TorrentFile, peerId [20]byte, port int) *Peer {
	trackerURL, err := buildTrackerURL(torrentFile, peerId, port)
	if err != nil {
		panic(err)
	}

	resp, err := http.Get(trackerURL)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	return parsePeersResponse(resp.Body)
}

func GeneratePeerId() [20]byte {
	var peerId [20]byte
	_, err := rand.Read(peerId[:])
	if err != nil {
		panic(err)
	}

	return peerId
}

///////////////////////////////// Helper Functions /////////////////////////////////

func hasPiece(bf []byte, index int) bool {
	byteIndex := index / 8
	bitIndex := index % 8

	return bf[byteIndex]&(1<<(7-bitIndex)) != 0
}

func setPiece(bf []byte, index int) {
	byteIndex := index / 8
	bitIndex := index % 8

	bf[byteIndex] |= 1 << (7 - bitIndex)
}

func parsePeersResponse(resp io.ReadCloser) *Peer {
	body, err := io.ReadAll(resp)
	if err != nil {
		panic(err)
	}

	peersResponse, _, _ := bencode.Decode(body)
	peersMap := peersResponse.(map[string]interface{})

	unparsedPeerIPs := []byte(peersMap["peers"].(string))
	parsedPeerIPs := []string{}

	// Convert the byte array of peer ips into a string of the format x.x.x.x:x
	for i := 0; i < len(unparsedPeerIPs); i += 6 {
		// The first 4 bytes are the ip
		ip := net.IP(unparsedPeerIPs[i : i+4]).String()

		// The last 2 bytes make the port
		port := strconv.Itoa(int(binary.BigEndian.Uint16(unparsedPeerIPs[i+4 : i+6])))

		parsedPeerIPs = append(parsedPeerIPs, ip+":"+port)
	}

	parsedPeers := Peer{
		Complete:   peersMap["complete"].(int),
		Incomplete: peersMap["incomplete"].(int),
		Interval:   peersMap["interval"].(int),
		Peers:      parsedPeerIPs,
	}

	return &parsedPeers

}

func buildTrackerURL(torrentFile *TorrentFile, peerId [20]byte, port int) (string, error) {
	baseURL, err := url.Parse(torrentFile.Announce)
	if err != nil {
		return "", err
	}

	query := baseURL.Query()
	query.Set("info_hash", string(torrentFile.InfoHash[:]))
	query.Set("peer_id", string(peerId[:]))
	query.Set("port", strconv.Itoa(port))
	query.Set("uploaded", "0")
	query.Set("downloaded", "0")
	query.Set("left", strconv.Itoa(torrentFile.Info.Length))
	query.Set("compact", "1")

	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}
