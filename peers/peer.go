package peers

import (
	"bittorrent/bencode"
	"bittorrent/messages"
	"bittorrent/pieces"
	"bittorrent/torrent"

	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type Peers struct {
	Peers []string // IP address of all peers
}

func (p *Peers) DownloadFromPeers(tf *torrent.TorrentFile, peerId [20]byte) {
	// Initialise worker queue and file data channel
	workerQueue := make(chan *pieces.PieceWork, len(tf.PiecesHash))
	for i := 0; i < len(tf.PiecesHash); i++ {
		piece := pieces.PieceWork{
			Index:     i,
			PieceHash: tf.PiecesHash[i],
			PieceSize: pieces.CalculatePieceSize(tf.Info.Length, tf.Info.PieceLength, i),
		}
		workerQueue <- &piece
	}

	fileData := make(chan *pieces.PieceResult, len(tf.PiecesHash))

	fmt.Println("There are", len(p.Peers), "peers available for download")

	// Start download workers for each peer
	for _, ip := range p.Peers {
		go startDownloadWorker(ip, workerQueue, fileData, tf, peerId)
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
		log.Fatal(err)
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

func RequestPeers(tf *torrent.TorrentFile, peerId [20]byte, port int) *Peers {
	var peers *Peers
	for _, url := range tf.AnnounceList {
		trackerURL, err := buildTrackerURL(url, tf.InfoHash, tf.Info.Length, peerId, port)
		if err != nil {
			log.Println("Error building tracker URL:", err)
			continue
		}

		resp, err := http.Get(trackerURL)
		if err != nil {
			log.Println("Error contacting tracker:", err)
			continue
		}

		defer resp.Body.Close()

		peers = parsePeersResponse(resp.Body)
		break
	}

	return peers
}

func GeneratePeerId() [20]byte {
	var peerId [20]byte
	_, err := rand.Read(peerId[:])
	if err != nil {
		log.Fatal(err)
	}

	return peerId
}

// /////////////////////////////// Helper Functions /////////////////////////////////
func startDownloadWorker(ip string, workerQueue chan *pieces.PieceWork, fileData chan *pieces.PieceResult, tf *torrent.TorrentFile, peerId [20]byte) {
	client, err := newPeerClient(ip, tf.InfoHash, peerId, len(tf.PiecesHash)/8+1)
	if err != nil {
		fmt.Println("Could not connect to peer", ip, ":", err)
		return
	}

	// Start downloading pieces
	for pw := range workerQueue {
		if !pieces.HasPiece(client.Bitfield, pw.Index) {
			// Peer does not have this piece, re-queue it
			workerQueue <- pw
			continue
		}

		pr, err := pieces.TryDownloadPiece(client, pw)
		if err != nil {
			fmt.Printf("Error downloading piece %d from peer %s: %s, re-queuing\n", pw.Index, ip, err)
			// Error during download, re-queue the piece
			workerQueue <- pw
			continue
		}

		if !pieces.ValidatePiece(pr, pw) {
			fmt.Printf("Piece %d from peer %s failed validation, re-queuing\n", pw.Index, ip)
			// Piece failed validation, re-queue it
			workerQueue <- pw
			continue
		}

		fmt.Printf("Finished downloading piece %d from peer %s\n", pw.Index, ip)

		fileData <- pr
	}

	fmt.Printf("Peer %s has no more pieces to download\n", ip)

	// Close the connection when done
	client.Conn.Close()
}

func newPeerClient(ip string, hash [20]byte, peerId [20]byte, bitfieldLength int) (*messages.Client, error) {
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
	msg, err := messages.RecieveMessage(conn)
	if err != nil {
		return nil, err
	}

	success = true
	bf := make([]byte, bitfieldLength)
	if msg.Id == messages.Bitfield {
		bf = msg.Payload
	}
	return &messages.Client{
		PeerIP:   ip,
		Conn:     conn,
		Bitfield: bf,
		Choked:   true,
	}, nil

}

func parsePeersResponse(resp io.ReadCloser) *Peers {
	body, err := io.ReadAll(resp)
	if err != nil {
		log.Fatal(err)
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
		Peers: parsedPeerIPs,
	}

	return &parsedPeers

}

func buildTrackerURL(urlString string, hash [20]byte, pieceLength int, peerId [20]byte, port int) (string, error) {
	baseURL, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	query := baseURL.Query()
	query.Set("info_hash", string(hash[:]))
	query.Set("peer_id", string(peerId[:]))
	query.Set("port", strconv.Itoa(port))
	query.Set("uploaded", "0")
	query.Set("downloaded", "0")
	query.Set("left", strconv.Itoa(pieceLength))
	query.Set("compact", "1")

	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}
