package download

import (
	peers "bittorrent/peers"
	torrent "bittorrent/torrent"
	"log"
)

func DownloadFile(torrentFilePath string) {
	torrentFile, err := torrent.ExtractTorrentInfo(torrentFilePath)
	if err != nil {
		log.Fatalf("Failed to extract torrent info: %v", err)
	}

	torrent.CalculatePiecesHash(torrentFile)

	peerId := peers.GeneratePeerId()
	peerIPs := peers.RequestPeers(torrentFile, peerId, 6881)

	peerIPs.DownloadFromPeers(torrentFile, peerId)
}
