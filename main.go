package main

import (
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

	peerId := torrent.GeneratePeerId()
	peers := torrent.RequestPeers(torrentFile, peerId, 6881)

	torrent.DownloadFromPeers(peers, torrentFile, peerId)

}
