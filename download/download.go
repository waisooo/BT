package download

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/anthony/BT/peer"
	"github.com/anthony/BT/torrent"
	"github.com/anthony/BT/tracker"
)

const port = 6881

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

	peerId := peer.GeneratePeerId()

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

	// Connect to all the peers and start downloading pieces
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
		fmt.Println("Error: No peers available for download")
		os.Exit(1)
	}

	peers.DownloadFromPeers(tf, peerId)
}
