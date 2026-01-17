module bittorrent/peers

go 1.22.2

replace bittorrent/bencode => ../bencode

replace bittorrent/torrent => ../torrent

replace bittorrent/pieces => ../pieces

replace bittorrent/messages => ../messages

require (
	bittorrent/pieces v0.0.0-00010101000000-000000000000
	bittorrent/torrent v0.0.0-00010101000000-000000000000
)

require (
	bittorrent/bencode v0.0.0-00010101000000-000000000000
	bittorrent/messages v0.0.0-00010101000000-000000000000
)
