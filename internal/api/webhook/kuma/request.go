package kuma

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// KumaWebhookPayload parses standard and custom webhook payloads sent by Uptime Kuma
type KumaWebhookPayload struct {
	Heartbeat  *KumaHeartbeat `json:"heartbeat,omitempty"`
	Monitor    *KumaMonitor   `json:"monitor,omitempty"`
	Msg        string         `json:"msg,omitempty"`
	Status     *int           `json:"status,omitempty"` // 0 = DOWN, 1 = UP
	StatusText string         `json:"-"`

	// Flat field fallbacks if user configures a custom webhook body
	MonitorIDFallback   interface{} `json:"monitor_id,omitempty"`
	MonitorIDCamel      interface{} `json:"monitorId,omitempty"`
	MonitorNameFallback string      `json:"monitor_name,omitempty"`
	MonitorNameCamel    string      `json:"monitorName,omitempty"`
	DeviceNameFallback  string      `json:"device_name,omitempty"`
	PingFallback        interface{} `json:"ping,omitempty"`
	PingResponseTime    interface{} `json:"ping_response_time,omitempty"`
	ResponseTime        interface{} `json:"responseTime,omitempty"`
	StatusFallback      interface{} `json:"status_code,omitempty"`
}

func (p *KumaWebhookPayload) UnmarshalJSON(data []byte) error {
	type alias KumaWebhookPayload
	aux := struct {
		Status json.RawMessage `json:"status"`
		*alias
	}{
		alias: (*alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(aux.Status) == 0 || string(aux.Status) == "null" {
		return nil
	}
	var number int
	if err := json.Unmarshal(aux.Status, &number); err == nil {
		p.Status = &number
		return nil
	}
	var text string
	if err := json.Unmarshal(aux.Status, &text); err == nil {
		p.StatusText = strings.ToLower(strings.TrimSpace(text))
		switch p.StatusText {
		case "1", "up", "online":
			number = 1
			p.Status = &number
		case "0", "down", "offline":
			number = 0
			p.Status = &number
		case "pending", "maintenance", "degraded":
			p.StatusText = p.StatusText
		}
	}
	return nil
}

type KumaHeartbeat struct {
	MonitorID int         `json:"monitorID"`
	Status    int         `json:"status"` // 0 = DOWN, 1 = UP, 2 = PENDING, 3 = MAINTENANCE
	Time      string      `json:"time"`
	Msg       string      `json:"msg"`
	Ping      interface{} `json:"ping"` // Can be float64 or int or null
	Duration  int         `json:"duration"`
}

type KumaMonitor struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url,omitempty"`
	Type     string `json:"type,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     int    `json:"port,omitempty"`
}

// GetMonitorID resolves the monitor ID as string
func (p *KumaWebhookPayload) GetMonitorID() string {
	if p.Monitor != nil && p.Monitor.ID > 0 {
		return strconv.Itoa(p.Monitor.ID)
	}
	if p.Heartbeat != nil && p.Heartbeat.MonitorID > 0 {
		return strconv.Itoa(p.Heartbeat.MonitorID)
	}
	if p.MonitorIDFallback != nil {
		return fmt.Sprintf("%v", p.MonitorIDFallback)
	}
	if p.MonitorIDCamel != nil {
		return fmt.Sprintf("%v", p.MonitorIDCamel)
	}
	return "0"
}

// GetMonitorName resolves the display name of the monitored device
func (p *KumaWebhookPayload) GetMonitorName() string {
	if p.Monitor != nil && strings.TrimSpace(p.Monitor.Name) != "" {
		return strings.TrimSpace(p.Monitor.Name)
	}
	if strings.TrimSpace(p.MonitorNameFallback) != "" {
		return strings.TrimSpace(p.MonitorNameFallback)
	}
	if strings.TrimSpace(p.DeviceNameFallback) != "" {
		return strings.TrimSpace(p.DeviceNameFallback)
	}
	if strings.TrimSpace(p.MonitorNameCamel) != "" {
		return strings.TrimSpace(p.MonitorNameCamel)
	}
	// Try parsing name from top-level msg, e.g. "[🔴 Down] [Router Induk] Reason"
	if p.Msg != "" {
		start := strings.Index(p.Msg, "[")
		if start != -1 {
			nextStart := strings.Index(p.Msg[start+1:], "[")
			if nextStart != -1 {
				realStart := start + 1 + nextStart + 1
				end := strings.Index(p.Msg[realStart:], "]")
				if end != -1 {
					return strings.TrimSpace(p.Msg[realStart : realStart+end])
				}
			}
		}
	}
	return fmt.Sprintf("Monitor-%s", p.GetMonitorID())
}

// GetStatus returns normalized status string: "up", "down", or "pending"
func (p *KumaWebhookPayload) GetStatus() string {
	// Priority 1: heartbeat.status
	if p.Heartbeat != nil {
		switch p.Heartbeat.Status {
		case 1:
			return "up"
		case 0:
			return "down"
		case 2:
			return "pending"
		case 3:
			return "maintenance"
		}
	}

	// Priority 2: top-level status
	if p.Status != nil {
		if *p.Status == 1 {
			return "up"
		}
		return "down"
	}
	if p.StatusText != "" {
		switch p.StatusText {
		case "up", "online", "1":
			return "up"
		case "down", "offline", "0":
			return "down"
		case "pending", "maintenance", "degraded":
			return p.StatusText
		}
	}

	// Priority 3: custom status_code field
	if p.StatusFallback != nil {
		switch v := p.StatusFallback.(type) {
		case float64:
			if v == 1 {
				return "up"
			}
			return "down"
		case string:
			text := strings.ToLower(strings.TrimSpace(v))
			if text == "1" || text == "up" || text == "online" {
				return "up"
			}
			if text == "0" || text == "down" || text == "offline" {
				return "down"
			}
		}
	}

	// Priority 3: analyze text in msg
	msgLower := strings.ToLower(p.Msg)
	if strings.Contains(msgLower, "down") || strings.Contains(msgLower, "🔴") {
		return "down"
	}
	if strings.Contains(msgLower, "up") || strings.Contains(msgLower, "✅") {
		return "up"
	}

	return "unknown"
}

// GetPingMs returns response time in milliseconds
func (p *KumaWebhookPayload) GetPingMs() float64 {
	if p.Heartbeat != nil && p.Heartbeat.Ping != nil {
		switch v := p.Heartbeat.Ping.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	for _, candidate := range []interface{}{p.PingFallback, p.PingResponseTime, p.ResponseTime} {
		switch v := candidate.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return f
			}
		case nil:
			continue
		}
	}
	return 0.0
}

// GetMessage returns status description or reason
func (p *KumaWebhookPayload) GetMessage() string {
	if p.Heartbeat != nil && strings.TrimSpace(p.Heartbeat.Msg) != "" {
		return strings.TrimSpace(p.Heartbeat.Msg)
	}
	if strings.TrimSpace(p.Msg) != "" {
		return strings.TrimSpace(p.Msg)
	}
	return fmt.Sprintf("Status changed to %s", p.GetStatus())
}
