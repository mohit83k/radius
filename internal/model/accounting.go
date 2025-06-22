package model

import "time"

// AccountingRecord represents the parsed RADIUS accounting data.
type AccountingRecord struct {
	Username         string    `json:"username"`
	NASIPAddress     string    `json:"nas_ip_address"`
	NASPort          int       `json:"nas_port"`
	AcctStatusType   string    `json:"acct_status_type"`
	AcctSessionID    string    `json:"acct_session_id"`
	FramedIPAddress  string    `json:"framed_ip_address"`
	CallingStationID string    `json:"calling_station_id"`
	CalledStationID  string    `json:"called_station_id"`
	Timestamp        time.Time `json:"timestamp"`
	ClientIP         string    `json:"client_ip"`
	PacketType       string    `json:"packet_type"`
}
