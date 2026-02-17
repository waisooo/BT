package tracker

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/anthony/BT/bencode"
)

func RequestPeers(trackerUrl string, infoHash [20]byte, peerId [20]byte, port int) ([]string, error) {
	url, err := url.Parse(trackerUrl)
	if err != nil {
		fmt.Println("Error parsing tracker URL:", err)
		os.Exit(1)
	}

	switch url.Scheme {
	case "http", "https":
		return requestPeersFromHTTPTracker(url, infoHash, peerId, port)
	case "udp":
		return requestPeersFromUDPTracker(url, infoHash, peerId, port)
	default:
		return nil, fmt.Errorf("Unrecognised tracker url scheme: %s", url.Scheme)
	}
}

func requestPeersFromHTTPTracker(url *url.URL, infoHash [20]byte, peerId [20]byte, port int) ([]string, error) {
	trackerURL, err := buildTrackerURL(url, infoHash, peerId, port)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(trackerURL)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tracker responded with non-200 status code: %d", resp.StatusCode)
	}

	peersResponse, _, err := bencode.Decode(body)
	if err != nil {
		return nil, err
	}

	peersMap := peersResponse.(map[string]interface{})

	if peersMap["failure reason"] != nil {
		return nil, fmt.Errorf("Tracker response error: %s", peersMap["failure reason"])
	}

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

	return parsedPeerIPs, nil
}

func buildTrackerURL(url *url.URL, hash [20]byte, peerId [20]byte, port int) (string, error) {
	query := url.Query()
	query.Set("info_hash", string(hash[:]))
	query.Set("peer_id", string(peerId[:]))
	query.Set("port", strconv.Itoa(port))
	query.Set("uploaded", "0")
	query.Set("downloaded", "0")
	query.Set("left", "0")
	query.Set("compact", "1")

	url.RawQuery = query.Encode()

	return url.String(), nil
}
