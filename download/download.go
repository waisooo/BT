package download

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/anthony/BT/peer"
	"github.com/anthony/BT/torrent"
	"github.com/anthony/BT/tracker"
)

// DownloadFile takes in the path to a torrent file and downloads the file(s) specified in the torrent file.
func DownloadFile(torrentFilePath string) {
	tf, err := torrent.ExtractTorrentInfo(torrentFilePath)
	if err != nil {
		fmt.Printf("Error: Failed to extract torrent file metadata, %s\n", err)
		os.Exit(1)
	}

	err = torrent.CalculatePiecesHash(tf)
	if err != nil {
		fmt.Printf("Error: Failed to calculate pieces hash, %s\n", err)
		os.Exit(1)
	}

	var peerId [20]byte
	rand.Read(peerId[:])
	const port = 6881

	peerIps := []net.TCPAddr{}
	var wg sync.WaitGroup
	var mut sync.Mutex

	wg.Add(len(tf.AnnounceList))
	for _, trackerUrl := range tf.AnnounceList {
		go func() {
			defer wg.Done()
			addr, err := tracker.RequestPeers(trackerUrl, tf.InfoHash, peerId, port)
			if err != nil {
				return
			}

			mut.Lock()
			peerIps = append(peerIps, addr...)
			mut.Unlock()
		}()
	}

	wg.Wait()

	peerIps = peer.RemoveDuplicatePeers(peerIps)

	var peers peer.Peers

	wg.Add(len(peerIps))
	for _, ip := range peerIps {
		go func() {
			defer wg.Done()
			client, err := peer.NewPeerClient(ip, tf.InfoHash, peerId, (len(tf.PiecesHash)/8)+1)
			if err != nil {
				return
			}

			mut.Lock()
			peers.Peers = append(peers.Peers, client)
			mut.Unlock()
		}()
	}

	wg.Wait()

	if len(peers.Peers) == 0 {
		fmt.Println("No peers available for download")
		os.Exit(1)
	}

	peers.DownloadFromPeers(tf, peerId)
}
