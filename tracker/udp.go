package tracker

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"net/url"
	"time"
)

const protocolId uint64 = 0x41727101980
const connectAction uint32 = 0
const announceAction uint32 = 1

func requestPeersFromUDPTracker(url *url.URL, infoHash [20]byte, peerId [20]byte, port int) ([]net.TCPAddr, error) {
	udpAddr, err := net.ResolveUDPAddr(url.Scheme, url.Host)
	if err != nil {
		return nil, fmt.Errorf("Error resolving UDP address: %s", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("Error dialing UDP tracker: %s", err)
	}

	err = conn.SetReadBuffer(4096)
	if err != nil {
		return nil, fmt.Errorf("Error setting UDP read buffer: %s", err)
	}

	// Initiate handshake and get connection ID
	connectionId, err := initiateUDPHandshake(conn)
	if err != nil {
		return nil, err
	}

	transactionId, err := generateTransactionId()
	if err != nil {
		return nil, err
	}

	// This is the format of the announce request message as specified in the BEP 15:
	// https://www.bittorrent.org/beps/bep_0015.html#tracker-udp-protocol
	announceMsg := make([]byte, 98)
	binary.BigEndian.PutUint64(announceMsg[0:8], connectionId)
	binary.BigEndian.PutUint32(announceMsg[8:12], uint32(announceAction))
	binary.BigEndian.PutUint32(announceMsg[12:16], transactionId)

	copy(announceMsg[16:36], infoHash[:])
	copy(announceMsg[36:56], peerId[:])

	binary.BigEndian.PutUint64(announceMsg[56:64], 0) // downloaded
	binary.BigEndian.PutUint64(announceMsg[64:72], 0) // left
	binary.BigEndian.PutUint64(announceMsg[72:80], 0) // uploaded
	binary.BigEndian.PutUint32(announceMsg[80:84], 0) // event 0:none; 1:completed; 2:started; 3:stopped
	binary.BigEndian.PutUint32(announceMsg[84:88], 0) // IP address, 0 for default
	binary.BigEndian.PutUint32(announceMsg[88:92], 0) // Key

	// Trick to allowing the a negative unsigned int, it underflows
	neg1 := -1
	binary.BigEndian.PutUint32(announceMsg[92:96], uint32(neg1)) // num want, -1 for default
	binary.BigEndian.PutUint16(announceMsg[96:98], uint16(port)) // port

	// Send announce request and parse response
	_, err = conn.Write(announceMsg)
	if err != nil {
		fmt.Println("Error sending announce request to UDP tracker:", err)
		return nil, err
	}

	resp := make([]byte, 4096)
	n, err := conn.Read(resp)
	if err != nil {
		fmt.Println("Error sending announce request to UDP tracker:", err)
		return nil, err
	}

	resp = resp[:n]

	interval := binary.BigEndian.Uint32(resp[0:4])
	leechers := binary.BigEndian.Uint32(resp[4:8])
	seeders := binary.BigEndian.Uint32(resp[8:12])

	_, _, _ = interval, leechers, seeders // Currently unused, but could be used to prioritise certain trackers in the future

	peerIPs := []net.TCPAddr{}

	for i := 12; i < len(resp); i += 6 {
		ip := net.IP(resp[i : i+4]).String()
		port := int(binary.BigEndian.Uint16(resp[i+4 : i+6]))

		peerIPs = append(peerIPs, net.TCPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		})
	}

	return peerIPs, nil
}

func generateTransactionId() (uint32, error) {
	var transactionId uint32
	err := binary.Read(rand.Reader, binary.BigEndian, &transactionId)
	if err != nil {
		return 0, fmt.Errorf("Error generating transaction ID: %s", err)
	}

	return transactionId, nil
}

func parseUDPResponse(resp []byte, wantedAction uint32, wantedTransactionId uint32) ([]byte, error) {
	if len(resp) < 8 {
		return nil, fmt.Errorf("Invalid response from UDP tracker: too short")
	}

	action := binary.BigEndian.Uint32(resp[0:4])
	transactionId := binary.BigEndian.Uint32(resp[4:8])

	if action != wantedAction {
		return nil, fmt.Errorf("unexpected action in UDP tracker announce response")
	}

	if transactionId != wantedTransactionId {
		return nil, fmt.Errorf("transaction ID mismatch in UDP tracker announce response")
	}

	return resp[8:], nil
}

func initiateUDPHandshake(conn *net.UDPConn) (uint64, error) {
	// Generate a random transaction ID
	transactionId, err := generateTransactionId()
	if err != nil {
		return 0, err
	}

	// Send connection request
	msg := make([]byte, 16)
	binary.BigEndian.PutUint64(msg[0:8], uint64(protocolId))     // Protocol ID
	binary.BigEndian.PutUint32(msg[8:12], uint32(connectAction)) // Action (connect)
	binary.BigEndian.PutUint32(msg[12:16], transactionId)        // Transaction ID

	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // Set a read deadline to avoid hanging indefinitely
	defer conn.SetDeadline(time.Time{})

	_, err = conn.Write(msg)
	if err != nil {
		return 0, err
	}

	resp := make([]byte, 16)
	n, _, _, _, err := conn.ReadMsgUDP(resp, nil)
	if err != nil {
		return 0, err
	}

	if n != 16 {
		return 0, fmt.Errorf("Invalid response length from UDP tracker: expected 16 bytes, got %d", n)
	}

	resp, err = parseUDPResponse(resp, connectAction, transactionId)
	if err != nil {
		return 0, fmt.Errorf("Error parsing UDP tracker handshake response: %s", err)
	}

	connectionId := binary.BigEndian.Uint64(resp[0:8])

	return connectionId, nil
}
