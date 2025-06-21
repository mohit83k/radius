// File: github.com/mohit83k/radius/internal/server/server.go
package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"github.com/mohit83k/radius/internal/logger"
	"github.com/mohit83k/radius/internal/model"
	"github.com/mohit83k/radius/internal/redisclient"
	"github.com/sirupsen/logrus"
)

// Server handles incoming RADIUS Accounting-Request packets.
type Server struct {
	Addr   string
	Secret []byte
	Store  redisclient.Store
	Logger logger.Logger
}

// NewServer returns a new RADIUS accounting server.
func NewServer(addr string, secret string, store redisclient.Store, log logger.Logger) *Server {
	return &Server{
		Addr:   addr,
		Secret: []byte(secret),
		Store:  store,
		Logger: log,
	}
}

// ListenAndServe listens for RADIUS packets and processes them.
func (s *Server) ListenAndServe(ctx context.Context) error {

	addr, err := net.ResolveUDPAddr("udp", s.Addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}
	defer conn.Close()

	s.Logger.Info("RADIUS server listening on " + s.Addr)
	go func() {
		<-ctx.Done()
		_ = conn.Close() // this will unblock ReadFromUDP
	}()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			s.Logger.Info("Shutting down RADIUS server")
			return nil
		default:
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				if ctx.Err() != nil {
					return nil // graceful exit after unblock
				}
				s.Logger.Error(fmt.Errorf("failed to read UDP: %w", err))
				continue
			}

			go s.handlePacket(ctx, conn, buf[:n], remoteAddr)
		}
	}
}

func (s *Server) handlePacket(ctx context.Context, conn net.PacketConn, data []byte, remoteAddr *net.UDPAddr) {

	packet, err := radius.Parse(data, s.Secret)
	if err != nil {
		s.Logger.WithFields(logrus.Fields{
			"err":   err,
			"bytes": len(data),
			"from":  remoteAddr,
		}).Error(err)
		return
	}

	if packet.Code != radius.CodeAccountingRequest {
		s.Logger.WithFields(map[string]any{"type": packet.Code}).Info("Ignoring non-accounting packet")
		return
	}

	rec := model.AccountingRecord{
		Username:         rfc2865.UserName_GetString(packet),
		NASIPAddress:     rfc2865.NASIPAddress_Get(packet).String(),
		NASPort:          int(rfc2865.NASPort_Get(packet)),
		AcctStatusType:   rfc2866.AcctStatusType_Get(packet).String(),
		AcctSessionID:    rfc2866.AcctSessionID_GetString(packet),
		FramedIPAddress:  rfc2865.FramedIPAddress_Get(packet).String(),
		CallingStationID: rfc2865.CallingStationID_GetString(packet),
		CalledStationID:  rfc2865.CalledStationID_GetString(packet),
		Timestamp:        time.Now().UTC(),
		ClientIP:         remoteAddr.IP.String(),
		PacketType:       "Accounting-Request",
	}

	if err := s.Store.Save(ctx, rec); err != nil {
		s.Logger.Error(fmt.Errorf("failed to save record: %w", err))
		return
	}

	s.Logger.WithFields(map[string]any{
		"username": rec.Username,
		"status":   rec.AcctStatusType,
		"session":  rec.AcctSessionID,
	}).Info("Stored accounting record")

	// Send back Accounting-Response
	resp := radius.New(radius.CodeAccountingResponse, s.Secret)
	resp.Identifier = packet.Identifier
	resp.Authenticator = packet.Authenticator
	encodedResp, err := resp.Encode()
	if err != nil {
		s.Logger.Error(fmt.Errorf("failed to encode response: %w", err))
		return
	}
	if conn != nil {
		_, err = conn.WriteTo(encodedResp, remoteAddr)
		if err != nil {
			s.Logger.Error(fmt.Errorf("failed to send response: %w", err))
		}
	}
}
