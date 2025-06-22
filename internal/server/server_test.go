package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"github.com/mohit83k/radius/internal/logger"
	"github.com/mohit83k/radius/internal/model"
)

// --- Mocks ---

type mockStore struct {
	savedRecords []model.AccountingRecord
	err          error
}

func (m *mockStore) Save(_ context.Context, rec model.AccountingRecord) error {
	m.savedRecords = append(m.savedRecords, rec)
	return m.err
}

type mockLogger struct {
	lastMsg  string
	lastErr  error
	lastData map[string]any
}

func (l *mockLogger) Info(msg string) {
	l.lastMsg = msg
}
func (l *mockLogger) Error(err error) {
	l.lastErr = err
}
func (l *mockLogger) WithFields(fields map[string]any) logger.Logger {
	l.lastData = fields
	return l
}

type mockUDPConn struct {
	called  bool
	payload []byte
	addr    net.Addr
	err     error
}

func (m *mockUDPConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	m.called = true
	m.payload = b
	m.addr = addr
	return len(b), m.err
}

// Dummy methods to satisfy interface, though unused
func (m *mockUDPConn) ReadFrom([]byte) (int, net.Addr, error) { return 0, nil, nil }
func (m *mockUDPConn) Close() error                           { return nil }
func (m *mockUDPConn) LocalAddr() net.Addr                    { return nil }
func (m *mockUDPConn) SetDeadline(time.Time) error            { return nil }
func (m *mockUDPConn) SetReadDeadline(time.Time) error        { return nil }
func (m *mockUDPConn) SetWriteDeadline(time.Time) error       { return nil }

// --- Test ---

func TestHandlePacket_ValidRequest(t *testing.T) {
	secret := []byte("testing123")
	mockLog := &mockLogger{}
	mockStore := &mockStore{}
	s := &Server{
		Secret: secret,
		Store:  mockStore,
		Logger: mockLog,
	}

	// build valid RADIUS packet
	pkt := radius.New(radius.CodeAccountingRequest, secret)
	rfc2865.UserName_AddString(pkt, "testuser")
	rfc2865.FramedIPAddress_Add(pkt, net.ParseIP("10.0.0.1"))
	rfc2865.CallingStationID_AddString(pkt, "caller")
	rfc2865.CalledStationID_AddString(pkt, "callee")
	rfc2866.AcctStatusType_Add(pkt, 1)
	rfc2866.AcctSessionID_AddString(pkt, "abc123")

	data, _ := pkt.Encode()

	// simulate request from remote IP
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	// use nil conn (we're not testing response write here)
	s.handlePacket(context.Background(), nil, data, remoteAddr)

	// validate Redis save was called
	if len(mockStore.savedRecords) != 1 {
		t.Fatalf("expected 1 saved record, got %d", len(mockStore.savedRecords))
	}

	rec := mockStore.savedRecords[0]
	if rec.Username != "testuser" || rec.AcctStatusType != "Start" {
		t.Errorf("unexpected record saved: %+v", rec)
	}
}

func TestHandlePacket_InvalidPacket(t *testing.T) {
	s := &Server{
		Secret: []byte("testing123"),
		Logger: &mockLogger{},
		Store:  &mockStore{},
	}

	// send invalid radius data
	data := []byte("not-radius")
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}

	s.handlePacket(context.Background(), nil, data, remoteAddr)
	// no panic = pass
}

func TestHandlePacket_SendsResponse(t *testing.T) {
	secret := []byte("testing123")
	mockConn := &mockUDPConn{}
	mockStore := &mockStore{}
	mockLog := &mockLogger{}

	s := &Server{
		Secret: secret,
		Store:  mockStore,
		Logger: mockLog,
	}

	pkt := radius.New(radius.CodeAccountingRequest, secret)
	rfc2865.UserName_AddString(pkt, "testuser")
	rfc2866.AcctSessionID_AddString(pkt, "abc123")
	rfc2866.AcctStatusType_Add(pkt, 1)

	data, _ := pkt.Encode()
	remoteAddr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999}

	s.handlePacket(context.Background(), mockConn, data, remoteAddr)

	if !mockConn.called {
		t.Errorf("expected WriteTo to be called, but it was not")
	}
}

func TestHandlePacket_IgnoresNonAccountingPacket(t *testing.T) {
	secret := []byte("testing123")
	mockLog := &mockLogger{}
	s := &Server{
		Secret: secret,
		Logger: mockLog,
		Store:  &mockStore{},
	}

	pkt := radius.New(radius.CodeAccessRequest, secret) // Wrong code
	data, _ := pkt.Encode()

	s.handlePacket(context.Background(), nil, data, &net.UDPAddr{})

	if mockLog.lastMsg == "" || mockLog.lastData["type"] != radius.CodeAccessRequest {
		t.Errorf("Expected log for non-accounting packet, got: %+v", mockLog.lastData)
	}
}

func TestListenAndServe_ProcessesPacket(t *testing.T) {
	secret := "testing123"
	mockStore := &mockStore{}
	mockLog := &mockLogger{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Bind manually so we can capture the actual port
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to resolve addr: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	actualPort := conn.LocalAddr().(*net.UDPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", actualPort)

	// Construct server with manually bound conn
	srv := &Server{
		Addr:   addr,
		Secret: []byte(secret),
		Store:  mockStore,
		Logger: mockLog,
	}

	// Run server loop
	go func() {
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, remoteAddr, err := conn.ReadFromUDP(buf)
				if err != nil {
					continue
				}
				srv.handlePacket(ctx, conn, buf[:n], remoteAddr)
				cancel() // stop after 1 packet
			}
		}
	}()

	// Allow server to start
	time.Sleep(100 * time.Millisecond)

	// Send packet
	clientConn, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatalf("failed to dial UDP: %v", err)
	}

	pkt := radius.New(radius.CodeAccountingRequest, []byte(secret))
	rfc2865.UserName_AddString(pkt, "testuser")
	rfc2866.AcctStatusType_Add(pkt, 1)
	rfc2866.AcctSessionID_AddString(pkt, "abc123")
	data, err := pkt.Encode()
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	_, err = clientConn.Write(data)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	// Wait for handler to process
	time.Sleep(300 * time.Millisecond)

	if len(mockStore.savedRecords) == 0 {
		t.Errorf("expected at least one saved record")
	}
}
