package torrent

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

func extractMagnetInfo(magnetURI string) (TorrentFile, error) {
	url, err := url.Parse(magnetURI)
	if err != nil {
		return TorrentFile{}, err
	}

	xts := url.Query()["xt"]
	if len(xts) == 0 {
		return TorrentFile{}, fmt.Errorf("Invalid magnet URI: missing info hash")
	}

	var infoHash [20]byte
	for _, xt := range xts {
		if strings.HasPrefix(xt, "urn:btih:") {
			encodedHashInfo := strings.TrimPrefix(xt, "urn:btih:")

			switch len(encodedHashInfo) {
			case 40: // Hexadecimal encoding
				decodedHash, err := hex.DecodeString(encodedHashInfo)
				if err != nil {
					return TorrentFile{}, fmt.Errorf("Invalid magnet URI: invalid hexadecimal info hash")
				}
				copy(infoHash[:], decodedHash)
			case 32: // Base32 encoding
				decodedHash, err := base32.HexEncoding.DecodeString(encodedHashInfo)
				if err != nil {
					return TorrentFile{}, fmt.Errorf("Invalid magnet URI: invalid base32 info hash")
				}
				copy(infoHash[:], decodedHash)
			default:
				return TorrentFile{}, fmt.Errorf("Invalid magnet URI: unsupported info hash encoding")
			}
		}
	}

	if infoHash == [20]byte{} {
		return TorrentFile{}, fmt.Errorf("Invalid magnet URI: missing valid info hash")
	}

	announceList := url.Query()["tr"]
	if len(announceList) == 0 {
		return TorrentFile{}, fmt.Errorf("Invalid magnet URI: missing tracker URLs")
	}

	torrentFile := TorrentFile{
		Name:         url.Query().Get("dn"),
		InfoHash:     infoHash,
		AnnounceList: announceList,
	}

	return torrentFile, nil
}
