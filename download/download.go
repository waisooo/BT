package download

import (
	peers "bittorrent/peers"
	torrent "bittorrent/torrent"
	"fmt"
	"os"
)

func DownloadFile(torrentFilePath string) {
	torrentFile, err := torrent.ExtractTorrentInfo(torrentFilePath)
	if err != nil {
		fmt.Printf("Error: Failed to extract torrent file metadata, %s\n", err)
		os.Exit(1)
	}

	torrent.CalculatePiecesHash(torrentFile)

	peerId := peers.GeneratePeerId()
	peerIPs := peers.RequestPeers(torrentFile, peerId, 6881)

	peerIPs.DownloadFromPeers(torrentFile, peerId)
}
