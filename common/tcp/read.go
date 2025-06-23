package tcp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"time"
)

func ReadData(conn *net.TCPConn) ([]byte, error) {
	var dataLen uint32
	dataLenBuf := make([]byte, 4)
	if err := readFixedData(conn, dataLenBuf); err != nil {
		return nil, err
	}
	// fmt.Printf("readFixedData:%+v\n", dataLenBuf)
	buffer := bytes.NewBuffer(dataLenBuf)
	if err := binary.Read(buffer, binary.BigEndian, &dataLen); err != nil {
		return nil, fmt.Errorf("[ERROR] read headlen error:%s", err.Error())
	}
	if dataLen <= 0 {
		return nil, fmt.Errorf("[ERROR] wrong headlen :%d", dataLen)
	}
	dataBuf := make([]byte, dataLen)
	// fmt.Printf("readFixedData.dataLen:%+v\n", dataLen)
	if err := readFixedData(conn, dataBuf); err != nil {
		return nil, fmt.Errorf("[ERROR] read headlen error:%s", err.Error())
	}
	return dataBuf, nil
}

// read fixed length data
func readFixedData(conn *net.TCPConn, buf []byte) error {
	_ = (*conn).SetReadDeadline(time.Now().Add(time.Duration(120) * time.Second))
	var pos int = 0
	var totalSize int = len(buf)
	for {
		c, err := (*conn).Read(buf[pos:])
		if err != nil {
			return err
		}
		pos = pos + c
		if pos == totalSize {
			break
		}
	}
	return nil
}

// ReadDataNonBlocking reads data in edge-triggered non-blocking mode
// Returns all complete messages available in the socket buffer
func ReadDataNonBlocking(conn *net.TCPConn) ([][]byte, error) {
	var messages [][]byte
	buffer := make([]byte, 8192) // 8KB buffer

	// Read all available data from socket
	allData := make([]byte, 0)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if isEAGAIN(err) {
				// No more data available
				break
			}
			return messages, err
		}
		if n == 0 {
			break
		}
		allData = append(allData, buffer[:n]...)
	}

	// Parse complete messages from the accumulated data
	pos := 0
	for pos < len(allData) {
		// Need at least 4 bytes for length header
		if pos+4 > len(allData) {
			// Incomplete header, cannot process further
			break
		}

		// Read message length
		lenBytes := allData[pos : pos+4]
		var messageLen uint32
		buf := bytes.NewBuffer(lenBytes)
		if err := binary.Read(buf, binary.BigEndian, &messageLen); err != nil {
			return messages, fmt.Errorf("[ERROR] parse message length: %s", err.Error())
		}

		if messageLen <= 0 {
			return messages, fmt.Errorf("[ERROR] invalid message length: %d", messageLen)
		}

		// Check if we have complete message
		totalMsgSize := 4 + int(messageLen)
		if pos+totalMsgSize > len(allData) {
			// Incomplete message, cannot process further
			break
		}

		// Extract complete message
		messageData := allData[pos+4 : pos+totalMsgSize]
		messages = append(messages, messageData)
		pos += totalMsgSize
	}

	return messages, nil
}

// isEAGAIN checks if error is EAGAIN or EWOULDBLOCK
func isEAGAIN(err error) bool {
	if netErr, ok := err.(*net.OpError); ok {
		if sysErr, ok := netErr.Err.(*syscall.Errno); ok {
			return *sysErr == syscall.EAGAIN || *sysErr == syscall.EWOULDBLOCK
		}
	}
	return false
}
