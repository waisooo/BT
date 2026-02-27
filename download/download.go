package download

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/anthony/BT/dht"
	"github.com/anthony/BT/peer"
	"github.com/anthony/BT/torrent"
	"github.com/anthony/BT/tracker"
)

// DownloadFile takes in the path to a torrent file and downloads the file(s) specified in the torrent file.
func DownloadFile(source string) {
	tf, err := torrent.ExtractInfo(source)
	if err != nil {
		fmt.Printf("Error: Failed to extract torrent file metadata, %s\n", err)
		os.Exit(1)
	}

	var peerId [20]byte
	rand.Read(peerId[:])
	const port = 6881

	var peerAddrs []net.TCPAddr
	var wg sync.WaitGroup
	var mut sync.Mutex

	// Request peers from all trackers in the announce list
	wg.Add(len(tf.AnnounceList))
	for _, trackerUrl := range tf.AnnounceList {
		go func() {
			defer wg.Done()
			addr, err := tracker.RequestPeers(trackerUrl, tf.InfoHash, peerId, port)
			if err != nil {
				return
			}

			mut.Lock()
			peerAddrs = append(peerAddrs, addr...)
			mut.Unlock()
		}()
	}

	wg.Wait()

	fmt.Println("////////////////////////////////////////////")
	fmt.Println("////// Getting peers to download from //////")
	fmt.Println("////////////////////////////////////////////")

	peers := requestPeers(peerAddrs, tf, peerId)
	if len(peers.Peers) == 0 {
		fmt.Println("No peers available for download")
		os.Exit(1)
	}

	// If the DHT table is supported, extract addtional peers from the DHT network to
	// increase the number of available peers for download.
	var extraPeers []net.TCPAddr
	for _, client := range peers.Peers {
		if client.SupportsDHT {
			addrs, err := dht.GetPeersFromDHT(client, tf.InfoHash)
			if err != nil {
				continue
			}

			extraPeers = append(extraPeers, addrs...)
		}
	}
	peers.Peers = append(peers.Peers, requestPeers(extraPeers, tf, peerId).Peers...)

	// If the torrent file is a magnet link, we need to request the metadata for the torrent file
	// from the peers before we can download the file(s) specified in the torrent file.
	if strings.HasPrefix(source, "magnet") {
		for _, client := range peers.Peers {
			metadata, err := client.RequestMetadata(tf.InfoHash)
			if err != nil {
				fmt.Printf("Error requesting metadata from peer %s: %s\n", client.Ip, err)
			}

			tf.Info.Pieces += string(metadata)
		}
	}

	tf.CalculatePiecesHash()

	peers.DownloadFromPeers(tf, peerId)
}

func requestPeers(peerAddrs []net.TCPAddr, tf torrent.TorrentFile, peerId [20]byte) peer.Peers {
	peerAddrs = peer.RemoveDuplicatePeers(peerAddrs)
	var peers peer.Peers

	// Attempt to establish a connection with each peer
	var wg sync.WaitGroup
	var mut sync.Mutex

	wg.Add(len(peerAddrs))
	for _, ip := range peerAddrs {
		go func() {
			defer wg.Done()
			client, err := peer.NewPeerClient(ip, tf, peerId)
			if err != nil {
				return
			}

			mut.Lock()
			peers.Peers = append(peers.Peers, client)
			mut.Unlock()
		}()
	}

	wg.Wait()

	return peers
}
