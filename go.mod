module main/main

go 1.22.2

replace bittorrent/decode => ./bencode

replace bittorrent/torrent => ./torrent

require bittorrent/decode v0.0.0-00010101000000-000000000000

require bittorrent/torrent v0.0.0-00010101000000-000000000000
