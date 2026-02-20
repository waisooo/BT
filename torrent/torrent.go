package torrent

import (
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/anthony/BT/bencode"
)

type bencodeInfo struct {
	Pieces      string `mapstructure:"pieces"`
	PieceLength int    `mapstructure:"piece length"`
	Name        string `mapstructure:"name"`
	Length      int    `mapstructure:"length"`
	Files       []struct {
		Length int      `mapstructure:"length"`
		Path   []string `mapstructure:"path"`
	} `mapstructure:"files"`
}

type bencodeTorrent struct {
	Announce     string      `mapstructure:"announce"`
	AnnounceList [][]string  `mapstructure:"announce-list"`
	Info         bencodeInfo `mapstructure:"info"`
}

type TorrentFile struct {
	Name         string
	AnnounceList []string
	InfoHash     [20]byte
	PiecesHash   [][20]byte
	Info         InfoDict
	Interval     int
}

type InfoDict struct {
	Name        string
	PieceLength int
	Pieces      string
	Length      int
	Files       []FileDict
}

type FileDict struct {
	Length int
	Path   string
}

// extractTorrentInfo takes in the path to a torrent file and extracts
// the metadata from the torrent file, returning a TorrentFile struct.
func extractTorrentInfo(filePath string) (TorrentFile, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return TorrentFile{}, err
	}

	var bcodedTorrent bencodeTorrent
	err = bencode.Decode(file, &bcodedTorrent)
	if err != nil {
		return TorrentFile{}, err
	}

	infoHash := calculateInfoHash(bcodedTorrent.Info)

	bcodedInfo := bcodedTorrent.Info
	infoDict := InfoDict{}

	infoDict.Name = bcodedInfo.Name
	infoDict.PieceLength = bcodedInfo.PieceLength
	infoDict.Pieces = bcodedInfo.Pieces

	// If the file key exists, the torrent file is a multi-file torrent, and the list of files is stored there
	// Otherwise, the torrent file is a single-file torrent, and the length of the file is stored in the length key
	if len(bcodedInfo.Files) > 0 {
		for _, file := range bcodedInfo.Files {
			path := strings.Join(file.Path, "/")
			infoDict.Files = append(infoDict.Files, FileDict{
				Length: file.Length,
				Path:   path,
			})
		}
	} else {
		infoDict.Length = bcodedInfo.Length
	}

	// If the announce-list key exists, the list of trackers is stored there
	var announceList []string
	for _, tracker := range bcodedTorrent.AnnounceList {
		announceList = append(announceList, tracker...)
	}

	// Only `announce` if `announce list` is not present
	if len(announceList) == 0 {
		announceList = append(announceList, bcodedTorrent.Announce)
	}

	torrent := TorrentFile{
		Name:         infoDict.Name,
		AnnounceList: announceList,
		InfoHash:     infoHash,
		Info:         infoDict,
		Interval:     1800,
	}

	err = torrent.CalculatePiecesHash()
	if err != nil {
		return TorrentFile{}, err
	}

	return torrent, nil
}

// CalculatePiecesHash takes in a TorrentFile struct and calculates the pieces hash
// for the torrent file, storing the pieces hash in the TorrentFile struct.
func (tf *TorrentFile) CalculatePiecesHash() error {
	infoMap := tf.Info
	const hashLen = 20

	if len(infoMap.Pieces)%hashLen != 0 {
		return fmt.Errorf("Invalid length for info pieces")
	}

	pieceHashes := make([][20]byte, len(infoMap.Pieces)/hashLen)
	for i := 0; i < len(pieceHashes); i++ {
		piece := infoMap.Pieces[i*hashLen : (i+1)*hashLen]
		copy(pieceHashes[i][:], piece)
	}

	tf.PiecesHash = pieceHashes

	return nil
}

//////////////////////////////// Helper Functions /////////////////////////////////

// calculateInfoHash takes in the bencoded info struct calculates the SHA-1 hash
// of the bencoded info dictionary, returning the hash as a byte array.
func calculateInfoHash(data bencodeInfo) [20]byte {
	dataMap := map[string]interface{}{
		"name":         data.Name,
		"piece length": data.PieceLength,
		"pieces":       data.Pieces,
	}

	if len(data.Files) > 0 {
		var files []map[string]interface{}
		for _, file := range data.Files {
			files = append(files, map[string]interface{}{
				"length": file.Length,
				"path":   file.Path,
			})
		}
		dataMap["files"] = files
	} else {
		dataMap["length"] = data.Length
	}

	encodedInfo, err := bencode.Encode(dataMap)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return sha1.Sum(encodedInfo)
}
