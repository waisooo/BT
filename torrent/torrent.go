package torrent

import (
	"crypto/sha1"
	"fmt"
	"os"

	bencode "bittorrent/decode"
)

type TorrentFile struct {
	Announce   string
	InfoHash   [20]byte
	PiecesHash [][20]byte
	Info       InfoDict
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

	torrent := TorrentFile{
		Announce: fileDict["announce"].(string),
		InfoHash: infoHash,
		Info:     infoDict,
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
		panic(err)
	}

	return sha1.Sum(encodedInfo)
}
