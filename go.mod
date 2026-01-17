module main/main

go 1.22.2

replace bittorrent/bencode => ./bencode

replace bittorrent/torrent => ./torrent

replace bittorrent/peers => ./peers

replace bittorrent/messages => ./messages

replace bittorrent/pieces => ./pieces

require (
	bittorrent/bencode v0.0.0-00010101000000-000000000000 // indirect
	bittorrent/messages v0.0.0-00010101000000-000000000000 // indirect
	bittorrent/pieces v0.0.0-00010101000000-000000000000 // indirect
)

require (
	bittorrent/peers v0.0.0-00010101000000-000000000000
	bittorrent/torrent v0.0.0-00010101000000-000000000000
)
