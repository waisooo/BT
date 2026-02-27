# BitTorrent

This project is a basic implementation of a bitTorrent client from scratch in Go.

## Supports

- Original Specification ([BEP0003])
  - Multi-file .torrent files
  - Orignal and Compact Peer List formats ([BEP0023][])
  - UDP trackers ([BEP0015][])
  - Extension Protocol ([BEP0010][])
  - Magnet links ([BEP0009][])
  - DHT Protocol ([BEP0005][])

## Project Structure

- `bitTorrent/main.go`: Application entry point
- `bitTorrent/bencode/`
  - `bitTorrent/bencode/decode.go`: Contains the logic for decoding a bencoded `.torrent` file
  - `bitTorrent/bencode/encode.go`: Contains the logic for encoding a string into bencode format
  - `bitTorrent/bencode/decode_test.go`: Unit tests for the bencode decoder
  - `bitTorrent/bencode/encode_test.go`: Unit tests for the bencode encoder
- `bitTorrent/dht`
  - `bitTorrent/dht/dht.go`: Contains the logic for interacting with the DHT network to retrieve peers
- `bittorrent/download`
  - `bittorrent/download/download.go`: Abstracts the file downloading functionality away from main.go
- `bitTorrent/message`
  - `bitTorrent/message/extension.go`: Contains the logic for extension for peers to send metadata files
  - `bitTorrent/message/message.go`: Handles requests to send/recieve peer messages
- `bitTorrent/peer`:
  - `bitTorrent/peers/peer.go`: Implements peer related functionality including peer discovery, handshakes and initialising piece download
- `bitTorrent/piece`
  - `bitTorrent/piece/piece.go`: Implements piece related functionality including downloading blocks, validating pieces and piece state management
- `bittorrent/test`
  - `bittorrent/test/large_download.torrent`: Torrent file used for testing large file download
  - `bittorrent/test/small_download.torrent`: Torrent file used for testing small file download
  - `bittorrent/test/test_large_download.sh`: Bash script that runs the client with `large_download.torrent` and verifies the SHA-256 checksum
  - `bittorrent/test/test_small_download.sh`: Bash script that runs the client with `small_download.torrent` and verifies the SHA-256 checksum
- `bitTorrent/torrent`
  - `bitTorrent/torrent/extractor.go`: Defines the extractor interface and abstracts extraction logic for both magnet links and .torrent files 
  - `bitTorrent/torrent/magnetic.go`: Handles extracting metadata from a magnet link
  - `bitTorrent/torrent/torrent.go`: Handles extracting metadata from a `.torrent` file
- `bitTorrent/tracker`
  - `bitTorrent/tracker/http.go`: Contains the logic for extracting peers from HTTP tracker
  - `bitTorrent/tracker/tracker.go`: Defines the tracker interface and abstracts peer retrieval logic
  - `bitTorrent/tracker/udp.go`: Contains the logic for extracting peers from UDP tracker

<!-- Reference links -->
[BEP0003]: https://bittorrent.org/beps/bep_0003.html 'Original bittorrent specification'
[BEP0023]: http://bittorrent.org/beps/bep_0023.html 'Compact Peer List specification'
[BEP0015]: https://www.bittorrent.org/beps/bep_0015.html 'UDP Tracker Protocol specification'
[BEP0010]: https://www.bittorrent.org/beps/bep_0010.html 'Extension Protocol specification'
[BEP0009]: https://www.bittorrent.org/beps/bep_0009.html 'Magnet URI specification'
[BEP0005]: https://www.bittorrent.org/beps/bep_0005.html 'DHT Protocol specification'
