module bittorrent/download

go 1.25.6

replace bittorrent/peers => ../peers

replace bittorrent/torrent => ../torrent

replace bittorrent/pieces => ../pieces

replace bittorrent/messages => ../messages

replace bittorrent/bencode => ../bencode

require bittorrent/peers v0.0.0-00010101000000-000000000000

require bittorrent/torrent v0.0.0-00010101000000-000000000000

require (
	bittorrent/bencode v0.0.0-00010101000000-000000000000 // indirect
	bittorrent/messages v0.0.0-00010101000000-000000000000 // indirect
	bittorrent/pieces v0.0.0-00010101000000-000000000000 // indirect
)
