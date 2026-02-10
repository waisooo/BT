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
	"sync"
	"time"
)

type Peers struct {
	Interval int      // Time in seconds that the client should wait before re-contacting the tracker
	Peers    []string // IP address of all peers
}

type workerInfo struct {
	ip          string
	hash        [20]byte
	workerQueue chan *pieces.PieceWork
	fileData    chan *pieces.PieceResult
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

	bitfieldLength := (len(tf.PiecesHash) / 8) + 1

	var wg sync.WaitGroup

	// Start download workers for each peer
	for _, ip := range p.Peers {
		workerInfo := workerInfo{
			ip:          ip,
			hash:        tf.InfoHash,
			workerQueue: workerQueue,
			fileData:    fileData,
		}

		go handleWorker(&workerInfo, &wg, bitfieldLength, peerId, p.Interval)
	}

	wg.Wait()

	fmt.Println("All pieces have been downloaded")

	// Combine pieces and write to file
	finalData := make([]byte, tf.Info.Length)
	for i := 0; i < len(tf.PiecesHash); i++ {
		result := <-fileData

		copy(finalData[result.Index*tf.Info.PieceLength:], result.Data)
	}

	err := os.WriteFile(tf.Info.Name, finalData, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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
			fmt.Println("Error building tracker URL:", err)
			continue
		}

		resp, err := http.Get(trackerURL)
		if err != nil {
			fmt.Println("Error contacting tracker:", err)
			continue
		}

		defer resp.Body.Close()

		peers = parsePeersResponse(resp.Body)
		if len(peers.Peers) > 0 {
			break
		}
	}

	if len(peers.Peers) == 0 {
		fmt.Println("No peers found from any tracker")
		os.Exit(1)
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
func handleWorker(workerInfo *workerInfo, wg *sync.WaitGroup, bitfieldLength int, peerId [20]byte, interval int) {
	// Attempt to start the download worker, retrying up to 5 times if there are any errors
	for retries := 0; retries < 5; {
		err := startDownloadWorker(workerInfo, bitfieldLength, peerId)
		// If there are no errors, or if there are no more pieces to download, we exit the retry loop
		if err == nil || len(workerInfo.workerQueue) == 0 {
			break
		} else {
			// There was an error, which was either:
			// - An error connecting to peer
			// - An error during download
			// In either case, wait the recommended retry time before attempting to connect again
			time.Sleep(time.Duration(interval) * time.Second)
			interval *= 2
			retries++
			continue
		}

	}

	wg.Done()
}

func startDownloadWorker(workerInfo *workerInfo, bitfieldLength int, peerId [20]byte) error {
	ip := workerInfo.ip
	workerQueue := workerInfo.workerQueue

	client, err := newPeerClient(ip, workerInfo.hash, peerId, bitfieldLength)
	if err != nil {
		fmt.Println("Could not connect to peer", ip, ":", err)
		return err
	}

	// Start downloading pieces
	for pw := range workerQueue {
		index := pw.Index
		if !pieces.HasPiece(client.Bitfield, index) {
			// Peer does not have this piece, re-queue it
			workerQueue <- pw
			continue
		}

		piece, err := pieces.TryDownloadPiece(client, pw)
		if err == io.EOF {
			// If the peer connection has closed, we can stop trying to download from this peer
			client.Conn.Close()
			return err
		}

		if err != nil {
			fmt.Printf("Error downloading piece %d from peer %s: %s, re-queuing\n", index, ip, err)
			// Error during download, re-queue the piece
			workerQueue <- pw
			continue
		}

		if !pieces.ValidatePiece(piece, pw) {
			fmt.Printf("Piece %d from peer %s failed validation, re-queuing\n", index, ip)
			// Piece failed validation, re-queue it
			workerQueue <- pw
			continue
		}

		fmt.Printf("Finished downloading piece %d from peer %s\n", index, ip)

		workerInfo.fileData <- piece
	}

	// Close the connection when done
	client.Conn.Close()

	return nil
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
		fmt.Println(err)
		os.Exit(1)
	}

	peersResponse, _, _ := bencode.Decode(body)
	peersMap := peersResponse.(map[string]interface{})

	if peersMap["failure reason"] != nil {
		fmt.Println("Tracker response error:", peersMap["failure reason"])
		os.Exit(1)
	}

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
		Interval: peersMap["interval"].(int),
		Peers:    parsedPeerIPs,
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
