package torrent

import (
	bencode "bittorrent/decode"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

type Peer struct {
	complete   int
	incomplete int
	interval   int      // How often queries should be re-sent
	peers      []string // IP address of all peers
}

func RequestPeers(torrentFile *TorrentFile, peerId [20]byte, port int) Peer {
	trackerURL, err := buildTrackerURL(torrentFile, peerId, port)
	if err != nil {
		panic(err)
	}

	resp, err := http.Get(trackerURL)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	return parsePeersResponse(resp.Body)
}

func parsePeersResponse(resp io.ReadCloser) Peer {
	body, err := io.ReadAll(resp)
	if err != nil {
		panic(err)
	}

	peersResponse, _, _ := bencode.Decode(body)
	peersMap := peersResponse.(map[string]interface{})

	unparsedPeerIPs := []byte(peersMap["peers"].(string))
	parsedPeerIPs := []string{}

	// Convert the byte array of peer ips into a string of the format x.x.x.x:x
	for i := 0; i < len(unparsedPeerIPs); i += 6 {
		// The first 4 bytes are the ip
		ip := net.IP(unparsedPeerIPs[i : i+4]).String()

		// The last 2 bytes make the port
		port := strconv.Itoa(int(binary.BigEndian.Uint16(unparsedPeerIPs[i+4 : i+6])))

		parsedPeerIPs = append(parsedPeerIPs, ip+":"+port)
	}

	parsedPeers := Peer{
		complete:   peersMap["complete"].(int),
		incomplete: peersMap["incomplete"].(int),
		interval:   peersMap["interval"].(int),
		peers:      parsedPeerIPs,
	}

	return parsedPeers

}

func buildTrackerURL(torrentFile *TorrentFile, peerId [20]byte, port int) (string, error) {
	baseURL, err := url.Parse(torrentFile.Announce)
	if err != nil {
		return "", err
	}

	query := baseURL.Query()
	query.Set("info_hash", string(torrentFile.InfoHash[:]))
	query.Set("peer_id", string(peerId[:]))
	query.Set("port", strconv.Itoa(port))
	query.Set("uploaded", "0")
	query.Set("downloaded", "0")
	query.Set("left", strconv.Itoa(torrentFile.Info.Length))
	query.Set("compact", "1")

	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}
