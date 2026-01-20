package models

import (
	"encoding/json"
	"net"
	"strings"
	"time"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

func (s Severity) Valid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	default:
		return false
	}
}

// What client sends
type EventIngestDTO struct {
	Ts       time.Time       `json:"ts"`
	Type     string          `json:"type"`
	Severity Severity        `json:"severity"`
	SrcIP    string          `json:"src_ip,omitempty"`
	Host     string          `json:"host,omitempty"`
	Username string          `json:"username,omitempty"`
	Msg      string          `json:"msg,omitempty"`
	Raw      json.RawMessage `json:"raw,omitempty"`
}

func (e EventIngestDTO) Validate() (srcIP net.IP, ok bool, errMsg string) {
	if e.Ts.IsZero() {
		return nil, false, "ts is required"
	}
	if strings.TrimSpace(e.Type) == "" {
		return nil, false, "type is required"
	}
	if !e.Severity.Valid() {
		return nil, false, "severity must be one of: low, medium, high, critical"
	}

	if e.SrcIP != "" {
		ip := net.ParseIP(e.SrcIP)
		if ip == nil {
			return nil, false, "src_ip is invalid"
		}
		return ip, true, ""
	}
	return nil, true, ""
}
