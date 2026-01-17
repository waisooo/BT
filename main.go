package main

import (
	peers "bittorrent/peers"
	torrent "bittorrent/torrent"
)

func main() {
	// Path to the torrent file
	torrentFilePath := "debian-13.3.0-amd64-netinst.iso.torrent"
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
