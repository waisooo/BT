package main

import (
	peers "bittorrent/peers"
	torrent "bittorrent/torrent"
)

func main() {
	// Path to the torrent file
	torrentFilePath := "ubuntu-22.04.5-desktop-amd64.iso.torrent"
	// torrentFilePath := "sample.torrent"

	// Parse the torrent file to extract metadata
	torrentFile, err := torrent.ExtractTorrentInfo(torrentFilePath)
	if err != nil {
		panic(err)
	}

	torrent.CalculatePiecesHash(torrentFile)

	peerId := peers.GeneratePeerId()
	peerIPs := peers.RequestPeers(torrentFile, peerId, 6881)

	peerIPs.DownloadFromPeers(torrentFile, peerId)

}
