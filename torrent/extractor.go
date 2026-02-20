package torrent

import (
	"fmt"
	"strings"
)

// ExtractInfo takes in a file path to a file and extracts the metadata from the file.
// It currently supports both .torrent files and magnet links,
// and returns a TorrentFile struct containing the metadata from the file.
func ExtractInfo(source string) (TorrentFile, error) {
	switch {
	case strings.HasPrefix(source, "magnet"):
		return extractMagnetInfo(source)
	case strings.HasSuffix(source, ".torrent"):
		return extractTorrentInfo(source)
	default:
		return TorrentFile{}, fmt.Errorf("Unknown file type: %s", source)
	}
}
