package torrent

import (
	"crypto/sha1"
	"fmt"
	"os"

	bencode "bittorrent/bencode"
)

type TorrentFile struct {
	AnnounceList []string
	InfoHash     [20]byte
	PiecesHash   [][20]byte
	Info         InfoDict
	Interval     int
}

type InfoDict struct {
	PieceLength int
	Pieces      string
	Name        string
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
		if pieceLength, ok := val["piece length"].(int); ok {
			infoDict.PieceLength = pieceLength
		}

		if pieces, ok := val["pieces"].(string); ok {
			infoDict.Pieces = pieces
		}

		if name, ok := val["name"].(string); ok {
			infoDict.Name = name
		}

		if length, ok := val["length"].(int); ok {
			infoDict.Length = length
		}
	}

	// If the announce-list key exists, the list of trackers is stored there
	annouceList := []string{}
	if list, ok := fileDict["announce-list"].([]interface{}); ok {
		for _, trackers := range list {
			if tracker, ok := trackers.([]interface{}); ok {
				for _, t := range tracker {
					annouceList = append(annouceList, t.(string))
				}
			}
		}
	}

	// Add the announce key at the end in case all other trackers fail
	annouceList = append(annouceList, fileDict["announce"].(string))

	torrent := TorrentFile{
		AnnounceList: annouceList,
		InfoHash:     infoHash,
		Info:         infoDict,
		Interval:     1800,
	}

	return &torrent, nil
}

func CalculatePiecesHash(torrentFile *TorrentFile) {
	infoMap := torrentFile.Info

	pieces := []byte(infoMap.Pieces)
	var piecesHash = [][20]byte{}
	for i := 0; i < len(pieces); i += 20 {
		end := i + 20
		if end > len(pieces) {
			end = len(pieces)
		}

		var pieceHash [20]byte
		copy(pieceHash[:], pieces[i:end])
		piecesHash = append(piecesHash, pieceHash)
	}

	torrentFile.PiecesHash = piecesHash
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
