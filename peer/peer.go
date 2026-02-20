package peer

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/anthony/BT/message"
	"github.com/anthony/BT/piece"
	"github.com/anthony/BT/torrent"
)

type Peers struct {
	Peers []*message.Client
}

// DownloadFromPeers takes in a torrent file and peer id and attempts to download all the pieces of the file
// from available peers. Upon downloading all the pieces, it reconstructs the original file and writes it to disk.
func (p *Peers) DownloadFromPeers(tf torrent.TorrentFile, peerId [20]byte) {
	// Initialise worker queue and file data channel
	workerQueue := make(chan piece.PieceWork, len(tf.PiecesHash))
	results := make(chan piece.PieceResult)

	fmt.Println("Starting download")
	fmt.Println("There are", len(p.Peers), "peers available for download")

	// Start download workers for each peer
	for _, client := range p.Peers {
		go func() {
			defer client.Conn.Close()
			for pw := range workerQueue {
				data, err := piece.TryDownloadPiece(client, pw)
				if err != nil {
					workerQueue <- pw

					if err == io.EOF {
						return
					}

					continue
				}

				results <- piece.PieceResult{
					Index: pw.Index,
					Data:  data,
				}
			}
		}()
	}

	// Send all the piece to the worker queue
	for i, hash := range tf.PiecesHash {
		length := tf.Info.PieceLength
		if i == len(tf.PiecesHash)-1 {
			length = tf.Info.Length - (tf.Info.PieceLength * (len(tf.PiecesHash) - 1))
		}

		workerQueue <- piece.PieceWork{
			Index:     i,
			PieceHash: hash,
			PieceSize: length,
		}
	}

	// Combine pieces into final file data as they are downloaded
	finalData := make([]byte, tf.Info.Length)
	for i := 0; i < len(tf.PiecesHash); i++ {
		result := <-results

		copy(finalData[result.Index*tf.Info.PieceLength:], result.Data)
		fmt.Printf("%0.2f%% complete\n", float64(i)/float64(len(tf.PiecesHash))*100)
	}

	// Determine whether the file to be downloaded is a single file or multiple files and write to disk
	if len(tf.Info.Files) == 0 {
		err := os.WriteFile(tf.Info.Name, finalData, 0644)
		if err != nil {
			fmt.Printf("Error: Failed to write file to disk, %s\n", err)
			os.Exit(1)
		}
	} else {
		index := 0
		for _, path := range tf.Info.Files {
			dir := filepath.Dir(path.Path)
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				fmt.Printf("Error: Failed to create directory, %s\n", err)
				os.Exit(1)
			}

			err = os.WriteFile(path.Path, finalData[index:path.Length], 0644)
			if err != nil {
				fmt.Printf("Error: Failed to write file to disk, %s\n", err)
				os.Exit(1)
			}

			index += path.Length
		}
	}
}

// RemoveDuplicatePeers takes in a list of peers and returns a new list with duplicate peers removed.
func RemoveDuplicatePeers(peers []net.TCPAddr) []net.TCPAddr {
	peerSet := make(map[string]bool)
	uniquePeers := []net.TCPAddr{}

	for _, peer := range peers {
		if !peerSet[peer.String()] {
			uniquePeers = append(uniquePeers, peer)
			peerSet[peer.String()] = true
		}
	}

	return uniquePeers
}

// NewPeerClient attempts to establish a connection with the peer, perform the BitTorrent handshake, and
// recieve the initial bitfield message from the peer if it is sent.
//
// If the handshake is successful, it returns a new Client representing the peer. Otherwise, it returns an error.
func NewPeerClient(addr net.TCPAddr, tf torrent.TorrentFile, peerId [20]byte) (*message.Client, error) {
	conn, err := net.DialTimeout("tcp", addr.String(), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dialing: %w", err)
	}

	client := &message.Client{
		Ip:   addr.IP.String(),
		Conn: conn,
	}

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	err = handshakePeer(client, tf.InfoHash, peerId)
	if err != nil {
		return nil, err
	}

	return client, nil
}

//////////////////////////////// Helper Functions /////////////////////////////////

// handShakePeer performs the BitTorrent handshake with a peer as specified by the BitTorrent protocol.
// It sends a handshake message to the peer and waits for a response.
//
// If the handshake is successful, it returns nil. Otherwise, it returns an error.
func handshakePeer(client *message.Client, infoHash [20]byte, peerId [20]byte) error {
	// Create handshake message
	msg := make([]byte, 68)

	msg[0] = 19                            // Length of protocol
	copy(msg[1:20], "BitTorrent protocol") // Protocol identifier

	// The reserved bytes are used to indicate support for extensions to the BitTorrent protocol.
	// Currently, we support only:
	// - Extended handshake (BEP_10) which allows us to request metadata from peers when dowloading from a magnet link.
	client.SupportsExtension = true

	extensionBytes := make([]byte, 8)
	extensionBytes[5] |= 0x10

	copy(msg[20:28], extensionBytes) // Reserved bytes
	copy(msg[28:48], infoHash[:])    // Info hash
	copy(msg[48:68], peerId[:])      // Peer ID

	// Send initial handshake
	_, err := client.Conn.Write(msg)
	if err != nil {
		return fmt.Errorf("sending handshake: %w", err)
	}

	// Read handshake response from peer
	// It should be in the same format as the sent msg
	response := make([]byte, 68)
	_, err = client.Conn.Read(response)
	if err != nil {
		return fmt.Errorf("reading handshake response: %w", err)
	}

	// Check that the response is in the correct format and contains the expected info hash
	if int(response[0]) != 19 || string(response[1:20]) != "BitTorrent protocol" {
		return fmt.Errorf("Invalid handshake response from peer")
	}

	if string(response[28:48]) != string(infoHash[:]) {
		return fmt.Errorf("Info hash mismatch in handshake response from peer")
	}

	// Continously read messages from the peer since clients send bitfield messages and other messages in random order after the handshake.
	if client.SupportsExtension {
		err = client.ExtendedPeerHandshake()
		if err != nil {
			return fmt.Errorf("Error performing extended handshake: %w", err)
		}
	}

	if len(client.Bitfield) == 0 {
		resp, err := client.RecieveMessage()
		if err != nil {
			return fmt.Errorf("Error reading message from peer: %w", err)
		}

		if resp.Id == message.Bitfield {
			client.Bitfield = resp.Payload
		}

		if len(client.Bitfield) == 0 {
			return fmt.Errorf("Peer did not send bitfield message")
		}
	}

	return nil
}
