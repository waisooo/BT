# BitTorrent

This project is a basic implementation of a bitTorrent client from scratch in Go.

# Project Structure

- `bitTorrent/main.go`: Application entry point
- `bitTorrent/bencode/`
  - `bitTorrent/bencode/decode.go`: Contains the logic for decoding a bencoded `.torrent` file
  - `bitTorrent/bencode/encode.go`: Contains the logic for encoding a string into bencode format
  - `bitTorrent/bencode/decode_test.go`: Unit tests for the bencode decoder
  - `bitTorrent/bencode/encode_test.go`: Unit tests for the bencode encoder
- `bittorrent/download`
  - `bittorrent/download/download.go`: Abstracts the file downloading functionality away from main.go
- `bitTorrent/messages`
  - `bitTorrent/messages/messages.go`: Handles requests to send/recieve peer messages
- `bitTorrent/peers`:
  - `bitTorrent/peers/peers.go`: Implements peer related functionality including peer discovery, handshakes, initialising piece download and peer ID generation
- `bitTorrent/pieces`
  - `bitTorrent/pieces/piece.go`: Implements piece related functionality including downloading blocks, validating pieces and piece state management
- `bittorrent/test`
  - `bittorrent/test/large_download.torrent`: Torrent file used for testing large file download
  - `bittorrent/test/small_download.torrent`: Torrent file used for testing small file download
  - `bittorrent/test/test_large_download.sh`: Bash script that runs the client with `large_download.torrent` and verifies the SHA-256 checksum
  - `bittorrent/test/test_small_download.sh`: Bash script that runs the client with `small_download.torrent` and verifies the SHA-256 checksum
- `bitTorrent/torrent`
  - `bitTorrent/torrent/torrent.go`: Handles extracting metadata from a `.torrent` file
