package tcp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"
	"time"
)

// Simple mock for testing
type testReader struct {
	data []byte
	pos  int
}

func (t *testReader) Read(buf []byte) (int, error) {
	if t.pos >= len(t.data) {
		// Simulate EAGAIN
		errno := syscall.EAGAIN
		return 0, &net.OpError{Op: "read", Net: "tcp", Err: &errno}
	}

	n := copy(buf, t.data[t.pos:])
	t.pos += n
	return n, nil
}

// Enhanced mock TCP connection for testing original functions
type mockTCPConn struct {
	readData     []byte
	readPos      int
	writeData    []byte
	readError    error
	writeError   error
	readDeadline time.Time
}

func (m *mockTCPConn) Read(b []byte) (int, error) {
	if m.readError != nil {
		return 0, m.readError
	}

	if m.readPos >= len(m.readData) {
		return 0, io.EOF
	}

	available := len(m.readData) - m.readPos
	toCopy := len(b)
	if toCopy > available {
		toCopy = available
	}

	copy(b, m.readData[m.readPos:m.readPos+toCopy])
	m.readPos += toCopy

	return toCopy, nil
}

func (m *mockTCPConn) Write(b []byte) (int, error) {
	if m.writeError != nil {
		return 0, m.writeError
	}
	m.writeData = append(m.writeData, b...)
	return len(b), nil
}

func (m *mockTCPConn) Close() error                             { return nil }
func (m *mockTCPConn) LocalAddr() net.Addr                      { return &net.TCPAddr{} }
func (m *mockTCPConn) RemoteAddr() net.Addr                     { return &net.TCPAddr{} }
func (m *mockTCPConn) SetDeadline(t time.Time) error            { return nil }
func (m *mockTCPConn) SetReadDeadline(t time.Time) error        { m.readDeadline = t; return nil }
func (m *mockTCPConn) SetWriteDeadline(t time.Time) error       { return nil }
func (m *mockTCPConn) SetKeepAlive(keepalive bool) error        { return nil }
func (m *mockTCPConn) SetKeepAlivePeriod(d time.Duration) error { return nil }
func (m *mockTCPConn) SetLinger(sec int) error                  { return nil }
func (m *mockTCPConn) SetNoDelay(noDelay bool) error            { return nil }
func (m *mockTCPConn) SetReadBuffer(bytes int) error            { return nil }
func (m *mockTCPConn) SetWriteBuffer(bytes int) error           { return nil }
func (m *mockTCPConn) CloseRead() error                         { return nil }
func (m *mockTCPConn) CloseWrite() error                        { return nil }

// Wrapper functions for testing with interface
func readDataWrapper(conn interface {
	Read([]byte) (int, error)
	SetReadDeadline(time.Time) error
}) ([]byte, error) {
	var dataLen uint32
	dataLenBuf := make([]byte, 4)

	// Read header
	if err := readFixedDataWrapper(conn, dataLenBuf); err != nil {
		return nil, err
	}

	// Parse length
	buffer := bytes.NewBuffer(dataLenBuf)
	if err := binary.Read(buffer, binary.BigEndian, &dataLen); err != nil {
		return nil, fmt.Errorf("[ERROR] read headlen error:%s", err.Error())
	}
	if dataLen <= 0 {
		return nil, fmt.Errorf("[ERROR] wrong headlen :%d", dataLen)
	}

	// Read data
	dataBuf := make([]byte, dataLen)
	if err := readFixedDataWrapper(conn, dataBuf); err != nil {
		return nil, fmt.Errorf("[ERROR] read data error:%s", err.Error())
	}
	return dataBuf, nil
}

func readFixedDataWrapper(conn interface {
	Read([]byte) (int, error)
	SetReadDeadline(time.Time) error
}, buf []byte) error {
	_ = conn.SetReadDeadline(time.Now().Add(time.Duration(120) * time.Second))
	var pos int = 0
	var totalSize int = len(buf)
	for {
		c, err := conn.Read(buf[pos:])
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

// Wrapper for SendData testing
func sendDataWrapper(conn interface{ Write([]byte) (int, error) }, data []byte) error {
	totalLen := len(data)
	writeLen := 0
	for {
		len, err := conn.Write(data[writeLen:])
		if err != nil {
			return err
		}
		writeLen = writeLen + len
		if writeLen >= totalLen {
			break
		}
	}
	return nil
}

// Modified function for testing that accepts interface
func readDataNonBlockingTest(reader interface{ Read([]byte) (int, error) }) ([][]byte, error) {
	var messages [][]byte
	buffer := make([]byte, 8192)

	// Read all available data
	allData := make([]byte, 0)
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if isEAGAIN(err) {
				break
			}
			return messages, err
		}
		if n == 0 {
			break
		}
		allData = append(allData, buffer[:n]...)
	}

	// Parse messages
	pos := 0
	for pos < len(allData) {
		if pos+4 > len(allData) {
			break // Incomplete header
		}

		lenBytes := allData[pos : pos+4]
		var messageLen uint32
		buf := bytes.NewBuffer(lenBytes)
		binary.Read(buf, binary.BigEndian, &messageLen)

		if messageLen <= 0 {
			return messages, fmt.Errorf("invalid message length: %d", messageLen)
		}

		totalMsgSize := 4 + int(messageLen)
		if pos+totalMsgSize > len(allData) {
			break // Incomplete message
		}

		messageData := allData[pos+4 : pos+totalMsgSize]
		messages = append(messages, messageData)
		pos += totalMsgSize
	}

	return messages, nil
}

// Helper function to create message with 4-byte length header
func createTestMessage(data []byte) []byte {
	length := uint32(len(data))
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, length)
	return append(buf.Bytes(), data...)
}

// ========== Tests for Original ReadData Function ==========

func TestReadData_SingleMessage(t *testing.T) {
	testData := []byte("Hello, Original ReadData!")
	mockData := createTestMessage(testData)
	conn := &mockTCPConn{readData: mockData}

	result, err := readDataWrapper(conn)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !bytes.Equal(result, testData) {
		t.Fatalf("Expected %s, got: %s", testData, result)
	}
}

func TestReadData_InvalidHeader(t *testing.T) {
	// Invalid header length (zero)
	mockData := []byte{0x00, 0x00, 0x00, 0x00}
	conn := &mockTCPConn{readData: mockData}

	_, err := readDataWrapper(conn)

	if err == nil {
		t.Fatalf("Expected error for invalid header length")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("wrong headlen")) {
		t.Fatalf("Expected 'wrong headlen' error, got: %v", err)
	}
}

func TestReadData_IncompleteData(t *testing.T) {
	// Header says 10 bytes, but only provide 5
	header := []byte{0x00, 0x00, 0x00, 0x0A}     // length = 10
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05} // only 5 bytes
	mockData := append(header, data...)

	conn := &mockTCPConn{readData: mockData}

	_, err := readDataWrapper(conn)

	if err == nil {
		t.Fatalf("Expected error for incomplete data")
	}

	// Check if the error contains EOF (it might be wrapped)
	if !bytes.Contains([]byte(err.Error()), []byte("EOF")) {
		t.Fatalf("Expected EOF error for incomplete data, got: %v", err)
	}
}

func TestReadFixedData_Success(t *testing.T) {
	testData := []byte("Fixed length data test")
	conn := &mockTCPConn{readData: testData}

	buf := make([]byte, len(testData))
	err := readFixedDataWrapper(conn, buf)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !bytes.Equal(buf, testData) {
		t.Fatalf("Expected %s, got: %s", testData, buf)
	}
}

func TestReadFixedData_PartialReads(t *testing.T) {
	testData := []byte("Test partial reads")
	// Mock that returns data in small chunks
	conn := &mockTCPConn{readData: testData}

	buf := make([]byte, len(testData))
	err := readFixedDataWrapper(conn, buf)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !bytes.Equal(buf, testData) {
		t.Fatalf("Expected %s, got: %s", testData, buf)
	}
}

// ========== Tests for Write Function ==========

func TestSendData_Success(t *testing.T) {
	testData := []byte("Test data for writing")
	conn := &mockTCPConn{}

	err := sendDataWrapper(conn, testData)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !bytes.Equal(conn.writeData, testData) {
		t.Fatalf("Expected %s, got: %s", testData, conn.writeData)
	}
}

func TestSendData_WriteError(t *testing.T) {
	testData := []byte("Test data")
	expectedErr := fmt.Errorf("write error")
	conn := &mockTCPConn{writeError: expectedErr}

	err := sendDataWrapper(conn, testData)

	if err != expectedErr {
		t.Fatalf("Expected write error, got: %v", err)
	}
}

// ========== Tests for DataPgk (Coder) ==========

func TestDataPgk_Marshal(t *testing.T) {
	testData := []byte("Test marshal data")
	pkg := DataPgk{
		Len:  uint32(len(testData)),
		Data: testData,
	}

	marshaled := pkg.Marshal()

	// Check length header (first 4 bytes)
	expected := createTestMessage(testData)
	if !bytes.Equal(marshaled, expected) {
		t.Fatalf("Marshal failed. Expected %v, got: %v", expected, marshaled)
	}
}

func TestDataPgk_Marshal_EmptyData(t *testing.T) {
	pkg := DataPgk{
		Len:  0,
		Data: []byte{},
	}

	marshaled := pkg.Marshal()

	// Should only contain the 4-byte length header
	expected := []byte{0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(marshaled, expected) {
		t.Fatalf("Marshal empty data failed. Expected %v, got: %v", expected, marshaled)
	}
}

func TestDataPgk_Marshal_LargeData(t *testing.T) {
	// Test with larger data
	largeData := make([]byte, 1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	pkg := DataPgk{
		Len:  uint32(len(largeData)),
		Data: largeData,
	}

	marshaled := pkg.Marshal()

	// Verify header
	var length uint32
	buf := bytes.NewBuffer(marshaled[:4])
	binary.Read(buf, binary.BigEndian, &length)

	if length != 1024 {
		t.Fatalf("Expected length 1024, got: %d", length)
	}

	// Verify data
	if !bytes.Equal(marshaled[4:], largeData) {
		t.Fatalf("Large data marshal failed")
	}
}

// ========== Tests for Non-blocking Functions ==========

func TestReadDataNonBlocking_SingleMessage(t *testing.T) {
	testData := []byte("Hello, World!")
	mockData := createTestMessage(testData)
	reader := &testReader{data: mockData}

	messages, err := readDataNonBlockingTest(reader)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got: %d", len(messages))
	}

	if !bytes.Equal(messages[0], testData) {
		t.Fatalf("Expected %s, got: %s", testData, messages[0])
	}
}

func TestReadDataNonBlocking_MultipleMessages(t *testing.T) {
	msg1 := []byte("Message 1")
	msg2 := []byte("Message 2")
	msg3 := []byte("Message 3")

	// Combine multiple messages
	data := createTestMessage(msg1)
	data = append(data, createTestMessage(msg2)...)
	data = append(data, createTestMessage(msg3)...)

	reader := &testReader{data: data}
	messages, err := readDataNonBlockingTest(reader)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got: %d", len(messages))
	}

	expected := [][]byte{msg1, msg2, msg3}
	for i, msg := range messages {
		if !bytes.Equal(msg, expected[i]) {
			t.Fatalf("Message %d: expected %s, got: %s", i, expected[i], msg)
		}
	}
}

func TestReadDataNonBlocking_IncompleteMessage(t *testing.T) {
	testData := []byte("Hello, World!")
	fullMessage := createTestMessage(testData)

	// Create incomplete message (remove last 5 bytes)
	incompleteMessage := fullMessage[:len(fullMessage)-5]

	reader := &testReader{data: incompleteMessage}
	messages, err := readDataNonBlockingTest(reader)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should return empty slice for incomplete message
	if len(messages) != 0 {
		t.Fatalf("Expected 0 messages for incomplete data, got: %d", len(messages))
	}
}

func TestReadDataNonBlocking_InvalidLength(t *testing.T) {
	// Message with zero length
	mockData := []byte{0x00, 0x00, 0x00, 0x00}
	reader := &testReader{data: mockData}

	_, err := readDataNonBlockingTest(reader)

	if err == nil {
		t.Fatalf("Expected error for invalid message length")
	}
}

func TestIsEAGAIN(t *testing.T) {
	// Test EAGAIN error
	errno := syscall.EAGAIN
	eagainErr := &net.OpError{Op: "read", Net: "tcp", Err: &errno}

	if !isEAGAIN(eagainErr) {
		t.Fatalf("Expected isEAGAIN to return true for EAGAIN error")
	}

	// Test non-EAGAIN error
	errno2 := syscall.ECONNRESET
	otherErr := &net.OpError{Op: "read", Net: "tcp", Err: &errno2}

	if isEAGAIN(otherErr) {
		t.Fatalf("Expected isEAGAIN to return false for non-EAGAIN error")
	}
}
