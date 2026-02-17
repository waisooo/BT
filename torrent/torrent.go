package torrent

import (
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/anthony/BT/bencode"
)

type TorrentFile struct {
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
}

func ExtractTorrentInfo(filePath string) (*TorrentFile, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contents, _, err := bencode.Decode(file)
	if err != nil {
		return nil, err
	}

	fileDict, ok := contents.(map[string]interface{})
	if !ok {
		panic("Error casting contents of file")
	}

	infoHash := calculateHash(fileDict["info"])

	infoDict := InfoDict{}
	if val, ok := fileDict["info"].(map[string]interface{}); ok {
		if name, ok := val["name"].(string); ok {
			infoDict.Name = name
		}

		if pieceLength, ok := val["piece length"].(int); ok {
			infoDict.PieceLength = pieceLength
		}

		if pieces, ok := val["pieces"].(string); ok {
			infoDict.Pieces = pieces
		}

		if length, ok := val["length"].(int); ok {
			infoDict.Length = length
		}
	}

	// If the announce-list key exists, the list of trackers is stored there
	annouceList := []string{}
	if list, ok := fileDict["announce-list"].([]interface{}); ok {
		for _, trackers := range list {
			annouceList = append(annouceList, trackers.([]string)...)
		}
	}

	// Only `announce` if `announce list` is not present
	if len(annouceList) == 0 {
		annouceList = append(annouceList, fileDict["announce"].(string))
	}

	torrent := TorrentFile{
		AnnounceList: annouceList,
		InfoHash:     infoHash,
		Info:         infoDict,
		Interval:     1800,
	}

	return &torrent, nil
}

func CalculatePiecesHash(torrentFile *TorrentFile) error {
	infoMap := torrentFile.Info
	const hashLen = 20

	if len(infoMap.Pieces)%hashLen != 0 {
		return fmt.Errorf("Invalid length for info pieces")
	}

	pieceHashes := make([][20]byte, len(infoMap.Pieces)/hashLen)
	for i := 0; i < len(pieceHashes); i++ {
		piece := infoMap.Pieces[i*hashLen : (i+1)*hashLen]
		copy(pieceHashes[i][:], piece)
	}

	torrentFile.PiecesHash = pieceHashes

	return nil
}

// ////////////////////////////// Helper Functions /////////////////////////////////

func calculateHash(data interface{}) [20]byte {
	encodedInfo, err := bencode.Encode(data)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return sha1.Sum(encodedInfo)
}
