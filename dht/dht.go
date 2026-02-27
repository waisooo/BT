package dht

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/anthony/BT/bencode"
	"github.com/anthony/BT/message"
)

type compactNode struct {
	conn *net.UDPConn
	id   string
	addr net.UDPAddr
}

// Custom types for DHT responses
type token string
type nodes []compactNode

func GetPeersFromDHT(client *message.Client, infoHash [20]byte) ([]net.TCPAddr, error) {
	// Create a compact node info for the initial DHT node
	node := compactNode{
		id: "",
		addr: net.UDPAddr{
			IP:   net.ParseIP(client.Ip),
			Port: client.DHT.Port,
		}}

	var err error
	node.conn, err = net.DialUDP("udp", nil, &node.addr)
	if err != nil {
		return nil, err
	}

	selfId, err := generateNodeId()
	if err != nil {
		return nil, err
	}

	// Get the node ID of the intial DHT node by pinging it
	node.id, err = node.ping(selfId)
	if err != nil {
		return nil, err
	}

	nodes, err := node.findNode(selfId)

	return lookupPeers(selfId, infoHash, nodes), nil
}

type dhtResult struct {
	Peers []net.TCPAddr
	Nodes []compactNode
}

// Performs a DHT lookup for peers by iteratively querying the closest nodes to the target info hash
// until we either find enough peers or exhaust the search space.
func lookupPeers(selfID string, infoHash [20]byte, initial []compactNode) []net.TCPAddr {
	const (
		K        = 8
		Alpha    = 3
		MaxNodes = 100
	)

	shortlist := initial
	sortByDistance(shortlist, infoHash)

	queried := make(map[string]bool)
	var foundPeers []net.TCPAddr

	for {
		// Pick the alpha closest nodes from the shortlist that have not been queried yet
		batch := pickAlphaCandidates(shortlist, queried)

		// If there are no more nodes to query, we can stop
		if len(batch) == 0 {
			break
		}

		// If we found 50 peers, we can stop as 50 peers is a good number to have for downloading
		if len(foundPeers) >= 50 {
			break
		}

		results := make(chan dhtResult, len(batch))
		var wg sync.WaitGroup

		for _, n := range batch {
			queried[n.id] = true

			wg.Add(1)
			go func() {
				defer wg.Done()

				conn, err := net.DialUDP("udp", nil, &n.addr)
				if err != nil {
					return
				}

				defer conn.Close()
				n.conn = conn

				_, peers, nodes, err := n.getPeers(selfID, infoHash)
				if err != nil {
					return
				}

				results <- dhtResult{
					Peers: peers,
					Nodes: nodes,
				}
			}()
		}

		wg.Wait()
		close(results)

		for res := range results {
			foundPeers = append(foundPeers, res.Peers...)
			shortlist = append(shortlist, res.Nodes...)
		}

		var unique []compactNode
		for _, n := range shortlist {
			if !queried[n.id] {
				unique = append(unique, n)
			}
		}

		shortlist = unique
		sortByDistance(shortlist, infoHash)

		// Limit the shortlist to the K closest nodes to prevent it from growing too large
		if len(shortlist) > MaxNodes {
			shortlist = shortlist[:MaxNodes]
		}
	}

	return foundPeers
}

// Sends a ping to a node, and returns the node id that was sent in the response
func (d *compactNode) ping(selfId string) (string, error) {
	// Ping query dictionary as per BEP_5 specifications
	pingQuery := map[string]interface{}{
		"t": "aa",
		"y": "q",
		"q": "ping",
		"a": map[string]interface{}{
			"id": selfId,
		},
	}

	encodedPingQuery, err := bencode.Encode(pingQuery)
	if err != nil {
		return "", err
	}

	_, err = d.conn.Write(encodedPingQuery)
	if err != nil {
		return "", err
	}

	d.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	resp := make([]byte, 4096)
	n, err := d.conn.Read(resp)
	if err != nil {
		return "", err
	}

	dhtResp, err := parseKRPCReponse(resp[:n])
	if err != nil {
		return "", err
	}

	return dhtResp.R.Id, nil
}

// Sends a find_node query to a node, and returns the list of nodes that were sent in the response.
func (d *compactNode) findNode(selfId string) ([]compactNode, error) {
	findNodeQuery := map[string]interface{}{
		"t": "aa",
		"y": "q",
		"q": "find_node",
		"a": map[string]interface{}{
			"id":     selfId,
			"target": d.id,
		},
	}

	encodedFindNodeQuery, err := bencode.Encode(findNodeQuery)
	if err != nil {
		return nil, err
	}

	_, err = d.conn.Write(encodedFindNodeQuery)
	if err != nil {
		return nil, err
	}

	d.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	resp := make([]byte, 4096)
	n, err := d.conn.Read(resp)
	if err != nil {
		return nil, nil
	}

	findNodeResp, err := parseKRPCReponse(resp[:n])
	if err != nil {
		return nil, err
	}

	var nodes []compactNode
	for i := 0; i < len(findNodeResp.R.Nodes); i += 26 {
		nodeInfo := parseCompactNodeInfo(findNodeResp.R.Nodes[i : i+26])
		nodes = append(nodes, nodeInfo)
	}

	return nodes, nil
}

// Sends a get_peers query to a node, and returns the token, list of peers, and list of nodes that were sent in the response.
func (d *compactNode) getPeers(selfId string, infoHash [20]byte) (token, []net.TCPAddr, nodes, error) {
	getPeersQuery := map[string]interface{}{
		"t": "aa",
		"y": "q",
		"q": "get_peers",
		"a": map[string]interface{}{
			"id":        selfId,
			"info_hash": string(infoHash[:]),
		},
	}

	encodedGetPeersQuery, err := bencode.Encode(getPeersQuery)
	if err != nil {
		return "", nil, nil, err
	}

	_, err = d.conn.Write(encodedGetPeersQuery)
	if err != nil {
		return "", nil, nil, err
	}

	d.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	resp := make([]byte, 4096)
	n, err := d.conn.Read(resp)
	if err != nil {
		return "", nil, nil, err
	}

	getPeersResp, err := parseKRPCReponse(resp[:n])
	if err != nil {
		return "", nil, nil, err
	}

	if getPeersResp.R.Token == "" {
		return "", nil, nil, fmt.Errorf("DHT node does not have peers for the given info hash")
	}

	var nodes []compactNode
	for i := 0; i < len(getPeersResp.R.Nodes); i += 26 {
		nodeInfo := parseCompactNodeInfo(getPeersResp.R.Nodes[i : i+26])
		nodes = append(nodes, nodeInfo)
	}

	var peers []net.TCPAddr
	for _, peer := range getPeersResp.R.Values {
		peerInfo := parseCompactPeerInfo(peer)
		peers = append(peers, peerInfo)
	}

	return token(getPeersResp.R.Token), peers, nodes, nil
}

// Sends an announce_peer query to a node, announcing that we have a peer for the given info hash.
func (d *compactNode) announcePeer(infoHash [20]byte, port uint16, token token) error {
	announcePeerQuery := map[string]interface{}{
		"t": "aa",
		"y": "q",
		"q": "announce_peer",
		"a": map[string]interface{}{
			"id":        d.id,
			"info_hash": infoHash,
			"port":      port,
			"token":     token,
		},
	}

	encodedAnnouncePeerQuery, err := bencode.Encode(announcePeerQuery)
	if err != nil {
		return err
	}

	_, err = d.conn.Write(encodedAnnouncePeerQuery)
	if err != nil {
		return err
	}

	d.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	resp := make([]byte, 4096)
	n, err := d.conn.Read(resp)
	if err != nil {
		return err
	}

	announcePeerResp, err := parseKRPCReponse(resp[:n])
	if err != nil {
		return err
	}

	if !bytes.Equal(infoHash[:], []byte(announcePeerResp.R.Id)) {
		return fmt.Errorf("DHT node did not acknowledge the announce peer request")
	}

	return nil

}

///////////////////////////// Helper functions /////////////////////////////

// KRPC response as specified in BEP_5
type krpcResp struct {
	T string        `mapstructure:"t"`
	Y string        `mapstructure:"y"`
	R dhtResp       `mapstructure:"r,omitempty"`
	E []interface{} `mapstructure:"e,omitempty"`
}

// Possible DHT response types as specified in BEP_5
type dhtResp struct {
	Id     string   `mapstructure:"id,omitempty"`
	Nodes  string   `mapstructure:"nodes,omitempty"`
	Token  string   `mapstructure:"token,omitempty"`
	Values []string `mapstructure:"values,omitempty"`
}

// Parses a KRPC response and validates it according to BEP_5 specifications.
// Returns the response as a krpcResp struct if valid,
// otherwise returns an error with details about the invalid response.
func parseKRPCReponse(data []byte) (krpcResp, error) {
	var resp krpcResp
	err := bencode.Decode(data, &resp)
	if err != nil {

		return krpcResp{}, err
	}

	// Validate transaction ID
	if resp.T != "aa" {
		return krpcResp{}, fmt.Errorf("Invalid KRPC transaction ID: %s", resp.T)
	}

	// Validate response type
	// The response type must be one of "r" (response), or "e" (error)
	if resp.Y != "r" && resp.Y != "e" {
		return krpcResp{}, fmt.Errorf("Invalid KRPC response type: %s", resp.Y)
	}

	// If the response type is "e", return the error details
	if resp.Y == "e" {
		return krpcResp{}, fmt.Errorf("KRPC error: %v", resp.E)
	}

	return resp, nil
}

// Parses a compact node info string (26 bytes) into a node struct with id, IP address, and port.
func parseCompactNodeInfo(node string) compactNode {
	compactNodeBytes := []byte(node)

	id := string(compactNodeBytes[:20])
	ip := net.IP(compactNodeBytes[20:24])
	port := binary.BigEndian.Uint16(compactNodeBytes[24:26])

	return compactNode{
		id: id,
		addr: net.UDPAddr{
			IP:   ip,
			Port: int(port),
		},
	}
}

// Parses a compact peer info string (6 bytes) into a TCP address with IP and port.
func parseCompactPeerInfo(peer string) net.TCPAddr {
	compactPeerBytes := []byte(peer)

	ip := net.IP(compactPeerBytes[:4])
	port := binary.BigEndian.Uint16(compactPeerBytes[4:6])
	return net.TCPAddr{
		IP:   ip,
		Port: int(port),
	}
}

// Generates a random 20-byte node ID by creating 20 random bytes and
// hashing them with SHA-1 to produce a unique identifier for the DHT node.
func generateNodeId() (string, error) {
	entropy := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, entropy); err != nil {
		return "", err
	}

	hash := sha1.Sum(entropy)

	return string(hash[:]), nil
}

// Picks the alpha closest nodes from the shortlist that have not been queried yet.
// Alpha is set to 3 in this case
func pickAlphaCandidates(shortlist []compactNode, queried map[string]bool) []compactNode {
	out := []compactNode{}

	const alpha = 3
	for _, node := range shortlist {
		if !queried[node.id] {
			out = append(out, node)
		}

		if len(out) == alpha {
			break
		}
	}

	return out
}

// Sorts the node by their distance to the target info hash
// using the Kademlia distance metric (XOR of node ID and target ID).
func sortByDistance(nodes []compactNode, target [20]byte) {
	sort.Slice(nodes, func(i, j int) bool {
		return compareDistances([]byte(nodes[i].id), []byte(nodes[j].id), target[:])
	})
}

// Computes the Kademlia distance between two node IDs and a target ID by performing a bitwise XOR operation on the node IDs and the target ID.
func kademliaDistance(a, b []byte) []byte {
	distance := make([]byte, len(a))
	for i := range a {
		distance[i] = a[i] ^ b[i]
	}

	return distance
}

// Compares the Kademlia distance of two node IDs to a target ID and returns true if the first node is closer to the target than the second node.
func compareDistances(a, b, target []byte) bool {
	return bytes.Compare(kademliaDistance(a, target), kademliaDistance(b, target)) < 0
}
