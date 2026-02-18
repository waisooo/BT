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

// Struct for parsing the tracker's response when the tracker responds with a compact string of peers.
// This format is based off of the specification in BEP 23.
type compactHttpTrackerResp struct {
	Interval int    `mapstructure:"interval"`
	Peers    string `mapstructure:"peers"`
}

// Struct for parsing the tracker's response when the tracker responds with a non-compact list of peers.
// This format is based off of the specification in BEP 3.
type normalHttpTrackerResp struct {
	Peers []struct {
		Id   string `mapstructure:"peer_id"`
		Ip   string `mapstructure:"ip"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"peers"`
	Interval   int    `mapstructure:"interval"`
	InfoHash   string `mapstructure:"info_hash"`
	Uploaded   int    `mapstructure:"uploaded"`
	Downloaded int    `mapstructure:"downloaded"`
	Left       int    `mapstructure:"left"`
	Event      string `mapstructure:"event"`
}

// requestPeersFromHTTPTracker attempts to extract a list of peers from the given HTTP tracker url
//
// It returns a list of peer IP addresses and ports if the request is successful. Otherwise it returns an error.
func requestPeersFromHTTPTracker(url *url.URL, infoHash [20]byte, peerId [20]byte, port int) ([]net.TCPAddr, error) {
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

	var compactResp compactHttpTrackerResp
	err = bencode.Decode(body, &compactResp)
	peerAddr := []net.TCPAddr{}

	// The tracker can respond with a list of peers in two formats:
	// 1. A compact string of the format x.x.x.x:x, where each peer is represented by 6 bytes (4 for IP and 2 for port)
	// 2. A list of dictionaries, where each dictionary contains the keys "ip" and "port"
	if err == nil {
		for i := 0; i < len(compactResp.Peers); i += 6 {
			ip := net.IP(compactResp.Peers[i : i+4])
			port := binary.BigEndian.Uint16([]byte(compactResp.Peers[i+4 : i+6]))

			peerAddr = append(peerAddr, net.TCPAddr{
				IP:   ip,
				Port: int(port),
			})
		}
	} else {
		var normalResp normalHttpTrackerResp
		err = bencode.Decode(body, &normalResp)
		if err != nil {
			return nil, fmt.Errorf("Error decoding tracker response: %v", err)
		}

		for _, peer := range normalResp.Peers {
			peerAddr = append(peerAddr, net.TCPAddr{
				IP:   net.ParseIP(peer.Ip),
				Port: peer.Port,
			})
		}
	}

	return peerAddr, nil
}

// buildTrackerURL constructs the URL to send to the tracker based on the given parameters.
//
// It returns the constructed URL as a string, or an error if there is an issue building the URL.
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
