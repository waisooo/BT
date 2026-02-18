package tracker

import (
	"fmt"
	"net"
	"net/url"
	"os"
)

// RequestPeers attempts to extract a list of peers from the given tracker url.
// It currently supports both HTTP and UDP trackers.
//
// It returns a list of peer IP addresses and ports if the request is successful. Otherwise it returns an error.
func RequestPeers(trackerUrl string, infoHash [20]byte, peerId [20]byte, port int) ([]net.TCPAddr, error) {
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
