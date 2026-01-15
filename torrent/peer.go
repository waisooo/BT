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
	"os"
	"strconv"
)

type Peers struct {
	Complete   int
	Incomplete int
	Interval   int      // How often queries should be re-sent
	Peers      []string // IP address of all peers
}

func DownloadFromPeers(peers *Peers, tf *TorrentFile, peerId [20]byte) {
	// Initialise worker queue and file data channel
	workerQueue := make(chan *PieceWork, len(tf.PiecesHash))
	for i := 0; i < len(tf.PiecesHash); i++ {
		piece := PieceWork{
			Index:            i,
			PieceHash:        tf.PiecesHash[i],
			PieceSize:        CalculatePieceSize(tf.Info.Length, tf.Info.PieceLength, i),
			DownloadAttempts: 0,
		}
		workerQueue <- &piece
	}

	fileData := make(chan *PieceResult, len(tf.PiecesHash))

	fmt.Println("There are", len(peers.Peers), "peers available for download")

	// Start download workers for each peer
	for _, ip := range peers.Peers {
		startDownloadWorker(ip, workerQueue, fileData, tf, peerId)
	}

	fmt.Println("All pieces have been downloaded")

	// Combine pieces and write to file
	finalData := make([]byte, tf.Info.Length)
	for i := 0; i < len(tf.PiecesHash); i++ {
		result := <-fileData

		copy(finalData[result.Index*tf.Info.PieceLength:], result.Data)
	}

	err := os.WriteFile(tf.Info.Name, finalData, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("File download complete:", tf.Info.Name)

	// Close channels
	close(fileData)
	close(workerQueue)
}

func HandShakePeer(conn net.Conn, infoHash [20]byte, peerId [20]byte) error {
	// Create handshake message
	message := make([]byte, 68)

	message[0] = 19                                    // Length of protocol
	copy(message[1:20], []byte("BitTorrent protocol")) // Protocol identifier
	copy(message[28:48], infoHash[:])                  // Info hash
	copy(message[48:68], peerId[:])                    // Peer ID

	// Send initial handshake
	_, err := conn.Write(message)
	if err != nil {
		return err
	}

	// Read handshake response from peer
	// It should be in the same format as the sent message
	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		return err
	}

	return nil
}

func RequestPeers(tf *TorrentFile, peerId [20]byte, port int) *Peers {
	trackerURL, err := buildTrackerURL(tf, peerId, port)
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

// /////////////////////////////// Helper Functions /////////////////////////////////
func startDownloadWorker(ip string, workerQueue chan *PieceWork, fileData chan *PieceResult, tf *TorrentFile, peerId [20]byte) {
	client, err := newPeerClient(ip, tf.InfoHash, peerId, len(tf.PiecesHash)/8+1)
	if err != nil {
		fmt.Println("Could not connect to peer", ip, ":", err)
		return
	}

	fmt.Println("Connection made with peer")

	haveUpdates := make(chan int, 64)

	go listenForHave(client, haveUpdates)

	// Start downloading pieces
	for len(fileData) < len(tf.PiecesHash) {
		select {
		case piece := <-haveUpdates:
			SetPiece(client.Bitfield, piece)
		default:
			pw := <-workerQueue
			if !hasPiece(client.Bitfield, pw.Index) {
				// Peer does not have this piece, re-queue it
				workerQueue <- pw
				continue
			}

			pr, err := TryDownloadPiece(client, pw)
			if err != nil {
				fmt.Printf("Error downloading piece %d from peer %s: %s, re-queuing\n", pw.Index, ip, err)
				// Error during download, re-queue the piece
				workerQueue <- pw
				continue
			}

			if !ValidatePiece(pr, pw) {
				fmt.Printf("Piece %d from peer %s failed validation, re-queuing\n", pw.Index, ip)
				// Piece failed validation, re-queue it
				workerQueue <- pw
				continue
			}

			fmt.Printf("Finished downloading piece %d from peer %s\n", pw.Index, ip)

			fileData <- pr
		}
	}

	fmt.Printf("Peer %s has no more pieces to download\n", ip)

	// Close the connection when done
	client.Conn.Close()
}

func newPeerClient(ip string, hash [20]byte, peerId [20]byte, bitfieldLength int) (*Client, error) {
	conn, err := net.Dial("tcp", ip)
	if err != nil {
		return nil, err
	}

	success := false
	defer func() {
		if !success {
			conn.Close()
		}
	}()

	err = HandShakePeer(conn, hash, peerId)
	if err != nil {
		return nil, err
	}

	// Check if first message is bitfield
	msg, err := recieveMessage(conn)
	if err != nil {
		return nil, err
	}

	success = true
	bf := make([]byte, bitfieldLength)
	if msg.Id == Bitfield {
		bf = msg.Payload
	}

	return &Client{
		PeerIP:   ip,
		Conn:     conn,
		Bitfield: bf,
		Choked:   true,
	}, nil

}

func listenForHave(conn *Client, haveUpdates chan int) {
	for {
		msg, err := recieveMessage(conn.Conn)
		if err != nil {
			fmt.Println("Error reading message from peer:", err)
			return
		}

		fmt.Println("Recived message ", msg.Id, " with contents ", msg.Payload)
		if msg.Id == Have {
			pieceIndex := binary.BigEndian.Uint32(msg.Payload)
			haveUpdates <- int(pieceIndex)
		}

	}
}

func parsePeersResponse(resp io.ReadCloser) *Peers {
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

	parsedPeers := Peers{
		Complete:   peersMap["complete"].(int),
		Incomplete: peersMap["incomplete"].(int),
		Interval:   peersMap["interval"].(int),
		Peers:      parsedPeerIPs,
	}

	return &parsedPeers

}

func buildTrackerURL(tf *TorrentFile, peerId [20]byte, port int) (string, error) {
	baseURL, err := url.Parse(tf.Announce)
	if err != nil {
		return "", err
	}

	query := baseURL.Query()
	query.Set("info_hash", string(tf.InfoHash[:]))
	query.Set("peer_id", string(peerId[:]))
	query.Set("port", strconv.Itoa(port))
	query.Set("uploaded", "0")
	query.Set("downloaded", "0")
	query.Set("left", strconv.Itoa(tf.Info.Length))
	query.Set("compact", "1")

	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}
