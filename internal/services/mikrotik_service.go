package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"skripsi-be/internal/models/entities"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type MikroTikService struct {
	sshClient   *ssh.Client
	config      *MikroTikConfig
	isConnected bool
}

type MikroTikConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type MikroTikLog struct {
	Host      string    `json:"host"`
	Comment   string    `json:"comment"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Category  string    `json:"category"`
	Raw       string    `json:"raw,omitempty"`
}

type ConnectionStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

func NewMikroTikService(config *MikroTikConfig) *MikroTikService {
	return &MikroTikService{
		config: config,
	}
}

func (s *MikroTikService) Connect() error {
	sshConfig := &ssh.ClientConfig{
		User: s.config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         20 * time.Second,
		// Add cipher and key exchange algorithms compatible with MikroTik RouterOS
		// Prioritize older algorithms that MikroTik RouterOS supports
		Config: ssh.Config{
			Ciphers: []string{
				"aes128-cbc", "aes192-cbc", "aes256-cbc",
				"3des-cbc",
				"aes128-ctr", "aes192-ctr", "aes256-ctr",
			},
			KeyExchanges: []string{
				"diffie-hellman-group1-sha1",
				"diffie-hellman-group14-sha1",
				"diffie-hellman-group-exchange-sha1",
				"diffie-hellman-group14-sha256",
				"diffie-hellman-group-exchange-sha256",
			},
			MACs: []string{
				"hmac-md5", "hmac-sha1",
				"hmac-sha2-256", "hmac-sha2-512",
			},
		},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", s.config.Host, s.config.Port), sshConfig)
	if err != nil {
		s.isConnected = false
		return fmt.Errorf("failed to connect to MikroTik: %v", err)
	}

	s.sshClient = client
	s.isConnected = true
	log.Printf("Connected to MikroTik at %s:%d", s.config.Host, s.config.Port)
	return nil
}

func (s *MikroTikService) Disconnect() error {
	if s.sshClient != nil {
		err := s.sshClient.Close()
		s.sshClient = nil
		s.isConnected = false
		log.Println("Disconnected from MikroTik")
		return err
	}
	return nil
}

// ReconnectWithBackoff restores the SSH connection after link failures.
func (s *MikroTikService) ReconnectWithBackoff(maxAttempts int) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var lastErr error
	delay := 200 * time.Millisecond
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = s.Connect()
		if lastErr == nil {
			return nil
		}
		if attempt < maxAttempts {
			time.Sleep(delay)
			delay *= 2
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
		}
	}
	return fmt.Errorf("MikroTik reconnect failed after %d attempts: %w", maxAttempts, lastErr)
}

func (s *MikroTikService) IsConnected() bool {
	return s.isConnected && s.sshClient != nil
}

func (s *MikroTikService) GetConnectionStatus() ConnectionStatus {
	if s.IsConnected() {
		return ConnectionStatus{
			Status:  "connected",
			Message: "Connected to MikroTik",
			Host:    s.config.Host,
			Port:    s.config.Port,
		}
	}
	return ConnectionStatus{
		Status:  "disconnected",
		Message: "Not connected",
		Host:    s.config.Host,
		Port:    s.config.Port,
	}
}

func (s *MikroTikService) GetLogs(timeRange string) ([]MikroTikLog, error) {
	if !s.IsConnected() {
		return nil, fmt.Errorf("not connected to MikroTik")
	}

	// Convert timeRange to MikroTik time format
	var timeFilter string
	switch timeRange {
	case "1h":
		timeFilter = "1h"
	case "6h":
		timeFilter = "6h"
	case "1d":
		timeFilter = "1d"
	case "7d":
		timeFilter = "7d"
	case "30d":
		timeFilter = "30d"
	default:
		timeFilter = "1d"
	}

	// Get Netwatch logs specifically - focus on UP/DOWN status changes
	command := fmt.Sprintf("/log print where topics~\"netwatch\" and time>%s", timeFilter)

	session, err := s.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	output, err := session.Output(command)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %v", err)
	}

	return s.parseLogs(string(output)), nil
}

func (s *MikroTikService) parseLogs(logData string) []MikroTikLog {
	lines := strings.Split(logData, "\n")
	var logs []MikroTikLog

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		log := s.parseLogLine(line)
		if log != nil {
			logs = append(logs, *log)
		}
	}

	// Reverse to show newest first
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	return logs
}

func (s *MikroTikService) parseLogLine(line string) *MikroTikLog {
	// Parse Netwatch logs - they typically contain UP/DOWN status
	// Example format: "netwatch host 192.168.1.100 DOWN"
	// or more detailed format with timestamp and message

	// Check if this is a Netwatch UP/DOWN log
	if strings.Contains(strings.ToLower(line), "netwatch") {
		// Extract host/IP address
		var host string
		var status string
		var message string = line

		// Try to extract IP address pattern
		if strings.Contains(line, ".") {
			// Look for IP address in the line
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Count(part, ".") == 3 {
					host = part
					break
				}
			}
		}

		// Determine status from the log content
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "down") || strings.Contains(lineLower, "timeout") || strings.Contains(lineLower, "unreachable") {
			status = "DOWN"
		} else if strings.Contains(lineLower, "up") || strings.Contains(lineLower, "reachable") || strings.Contains(lineLower, "alive") {
			status = "UP"
		} else {
			// Default based on log level
			if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "critical") {
				status = "DOWN"
			} else {
				status = "UP"
			}
		}

		if host == "" {
			host = "unknown"
		}

		return &MikroTikLog{
			Host:      host,
			Comment:   message,
			Status:    status,
			Timestamp: time.Now(), // In real implementation, parse actual timestamp
			Type:      s.getLogType(status),
			Category:  "netwatch",
			Raw:       line,
		}
	}

	// Handle pipe-separated format (custom format)
	if strings.Contains(line, "|") {
		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			host := strings.TrimSpace(parts[0])
			comment := strings.TrimSpace(parts[1])
			status := strings.TrimSpace(parts[2])

			return &MikroTikLog{
				Host:      host,
				Comment:   comment,
				Status:    strings.ToUpper(status),
				Timestamp: time.Now(),
				Type:      s.getLogType(status),
				Category:  "netwatch",
				Raw:       line,
			}
		}
	}

	// Fallback for other log formats - but prioritize UP/DOWN detection
	timestamp := time.Now() // In real implementation, parse actual timestamp

	// Extract status based on keywords
	lineLower := strings.ToLower(line)
	status := "INFO"
	if strings.Contains(lineLower, "down") || strings.Contains(lineLower, "error") || strings.Contains(lineLower, "critical") {
		status = "DOWN"
	} else if strings.Contains(lineLower, "up") {
		status = "UP"
	} else if strings.Contains(lineLower, "warning") {
		status = "WARNING"
	}

	// Extract host/topic if available
	topic := "system"
	if strings.Contains(line, "topic=") {
		if topicStart := strings.Index(line, "topic="); topicStart != -1 {
			topicEnd := strings.Index(line[topicStart:], " ")
			if topicEnd == -1 {
				topicEnd = len(line) - topicStart
			}
			topic = strings.TrimSpace(line[topicStart+6 : topicStart+topicEnd])
		}
	}

	// Get the message part
	message := line
	if strings.Contains(line, `message="`) {
		if msgStart := strings.Index(line, `message="`); msgStart != -1 {
			msgEnd := strings.Index(line[msgStart+9:], `"`)
			if msgEnd != -1 {
				message = line[msgStart+9 : msgStart+9+msgEnd]
			}
		}
	}

	return &MikroTikLog{
		Host:      topic,
		Comment:   message,
		Status:    status,
		Timestamp: timestamp,
		Type:      s.getLogType(status),
		Category:  "system",
		Raw:       line,
	}
}

func (s *MikroTikService) getLogType(status string) string {
	statusLower := strings.ToLower(status)
	switch statusLower {
	case "up", "info", "reachable", "alive":
		return "success"
	case "down", "error", "critical", "timeout", "unreachable":
		return "error"
	case "warning":
		return "warning"
	default:
		return "info"
	}
}

func (s *MikroTikService) ExecuteCommand(command string) (string, error) {
	if !s.IsConnected() {
		if err := s.ReconnectWithBackoff(3); err != nil {
			return "", err
		}
	}

	// Check if SSH client is still valid before creating session
	if s.sshClient == nil {
		s.isConnected = false
		return "", fmt.Errorf("SSH client is nil, connection lost")
	}

	session, err := s.sshClient.NewSession()
	if err != nil {
		// Mark connection as lost if session creation fails
		s.isConnected = false
		return "", fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Set a timeout for command execution (30 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a channel to handle timeout
	type result struct {
		output string
		err    error
	}
	resultChan := make(chan result, 1)

	go func() {
		output, err := session.Output(command)
		resultChan <- result{string(output), err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			// Check if it's a connection error - mark connection as lost
			errMsg := res.err.Error()
			if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "timeout") {
				s.isConnected = false
			}
			return "", fmt.Errorf("failed to execute command: %v", res.err)
		}
		return res.output, nil
	case <-ctx.Done():
		s.isConnected = false
		return "", fmt.Errorf("command execution timeout after 30 seconds")
	}
}

// Hotspot IP Binding Management
type HotspotIPBinding struct {
	ID       string `json:"id"`
	MAC      string `json:"mac"`
	Address  string `json:"address"`
	Type     string `json:"type"`
	Server   string `json:"server"`
	Comment  string `json:"comment"`
	Disabled bool   `json:"disabled"`
}

// SetHotspotIPBindingType changes the type of a hotspot IP binding
func (s *MikroTikService) SetHotspotIPBindingType(macAddress, bindingType string) error {
	if !s.IsConnected() {
		return fmt.Errorf("not connected to MikroTik")
	}

	// Command to set the IP binding type
	command := fmt.Sprintf("/ip hotspot ip-binding set [find mac-address=%s] type=%s", macAddress, bindingType)

	output, err := s.ExecuteCommand(command)
	if err != nil {
		return fmt.Errorf("failed to set hotspot IP binding type: %v", err)
	}

	// Check if the command was successful
	if strings.Contains(strings.ToLower(output), "no such item") {
		return fmt.Errorf("no IP binding found for MAC address: %s", macAddress)
	}

	log.Printf("Successfully set IP binding type to %s for MAC %s", bindingType, macAddress)
	return nil
}

// GetHotspotIPBindings retrieves all hotspot IP bindings
func (s *MikroTikService) GetHotspotIPBindings() ([]HotspotIPBinding, error) {
	if !s.IsConnected() {
		return nil, fmt.Errorf("not connected to MikroTik")
	}

	command := "/ip hotspot ip-binding print"
	output, err := s.ExecuteCommand(command)
	if err != nil {
		return nil, fmt.Errorf("failed to get hotspot IP bindings: %v", err)
	}

	var bindings []HotspotIPBinding
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Flags:") || strings.HasPrefix(line, "Columns:") {
			continue
		}

		// Parse the binding information
		// Format: "0 X 40:EE:15:C8:67:5D 10.10.21.10 10.10.21.10 all bypassed"
		parts := strings.Fields(line)
		if len(parts) >= 6 {
			binding := HotspotIPBinding{
				ID:       parts[0],
				MAC:      parts[2],
				Address:  parts[3],
				Type:     parts[5],
				Server:   parts[4],
				Disabled: strings.Contains(parts[1], "X"),
			}
			bindings = append(bindings, binding)
		}
	}

	return bindings, nil
}

// GetHotspotIPBindingByMAC retrieves a specific hotspot IP binding by MAC address
func (s *MikroTikService) GetHotspotIPBindingByMAC(macAddress string) (*HotspotIPBinding, error) {
	if !s.IsConnected() {
		return nil, fmt.Errorf("not connected to MikroTik")
	}

	command := fmt.Sprintf("/ip hotspot ip-binding print where mac-address=%s", macAddress)
	output, err := s.ExecuteCommand(command)
	if err != nil {
		return nil, fmt.Errorf("failed to get hotspot IP binding: %v", err)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Flags:") || strings.HasPrefix(line, "Columns:") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 6 {
			return &HotspotIPBinding{
				ID:       parts[0],
				MAC:      parts[2],
				Address:  parts[3],
				Type:     parts[5],
				Server:   parts[4],
				Disabled: strings.Contains(parts[1], "X"),
			}, nil
		}
	}

	return nil, fmt.Errorf("no IP binding found for MAC address: %s", macAddress)
}

func (s *MikroTikService) GetSystemInfo() (map[string]interface{}, error) {
	if !s.IsConnected() {
		return nil, fmt.Errorf("not connected to MikroTik")
	}

	info := make(map[string]interface{})

	// Get system resource info using correct MikroTik commands
	commands := map[string]string{
		"resource": "/system resource print",
		"identity": "/system identity print",
		"clock":    "/system clock print",
		"version":  "/system package print where name=\"system\"",
	}

	for key, cmd := range commands {
		output, err := s.ExecuteCommand(cmd)
		if err != nil {
			log.Printf("Failed to execute %s: %v", cmd, err)
			continue
		}
		info[key] = strings.TrimSpace(output)
	}

	// Parse resource info to extract specific values
	if resourceOutput, ok := info["resource"].(string); ok {
		// Parse CPU load, memory usage, uptime from resource output
		lines := strings.Split(resourceOutput, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "cpu-load:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					info["cpu_load"] = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(line, "free-memory:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					info["memory"] = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(line, "uptime:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					info["uptime"] = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return info, nil
}

// GetNetwatchDevices gets all Netwatch devices from MikroTik
func (s *MikroTikService) GetNetwatchDevices() ([]map[string]interface{}, error) {
	if !s.IsConnected() {
		return nil, fmt.Errorf("not connected to MikroTik")
	}

	command := "/tool netwatch print"
	output, err := s.ExecuteCommand(command)
	if err != nil {
		return nil, fmt.Errorf("failed to get netwatch devices: %v", err)
	}

	// Debug logging
	log.Printf("Netwatch devices raw output: %s", output)

	devices := s.parseNetwatchDevices(output)
	log.Printf("Parsed netwatch devices: %+v", devices)

	return devices, nil
}

// parseNetwatchDevices parses the netwatch devices output
func (s *MikroTikService) parseNetwatchDevices(output string) []map[string]interface{} {
	var devices []map[string]interface{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "Columns:") {
			continue
		}

		// Parse MikroTik netwatch output format:
		// "0 simple  10.0.130.34  10s      1m        down    2025-09-28 17:16:56"
		// "1 icmp    10.10.21.10                     down    2025-09-29 19:44:36"
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		device := make(map[string]interface{})

		// Extract fields based on position
		device[".id"] = parts[0]  // ID
		device["type"] = parts[1] // TYPE
		device["host"] = parts[2] // HOST

		// Handle optional fields
		if len(parts) > 3 {
			// Check if next field is timeout (contains 's' or 'm')
			if strings.Contains(parts[3], "s") || strings.Contains(parts[3], "m") {
				device["timeout"] = parts[3]
				if len(parts) > 4 {
					device["interval"] = parts[4]
					if len(parts) > 5 {
						device["status"] = strings.ToLower(parts[5])
					}
				}
			} else {
				// No timeout/interval, status is in position 3
				device["status"] = strings.ToLower(parts[3])
			}
		}

		// Only add if we have essential fields
		if _, hasHost := device["host"]; hasHost {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetNetwatchLogs gets real-time netwatch logs
func (s *MikroTikService) GetNetwatchLogs(timeRange string) ([]MikroTikLog, error) {
	if !s.IsConnected() {
		return nil, fmt.Errorf("not connected to MikroTik")
	}

	// Convert timeRange to MikroTik time format
	var timeFilter string
	switch timeRange {
	case "1h":
		timeFilter = "1h"
	case "6h":
		timeFilter = "6h"
	case "1d":
		timeFilter = "1d"
	case "7d":
		timeFilter = "7d"
	case "30d":
		timeFilter = "30d"
	default:
		timeFilter = "1d"
	}

	// Get Netwatch logs specifically - focus on UP/DOWN status changes
	command := fmt.Sprintf("/log print where topics~\"netwatch\" and time>%s", timeFilter)

	session, err := s.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	output, err := session.Output(command)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %v", err)
	}

	return s.parseNetwatchLogs(string(output)), nil
}

// parseNetwatchLogs parses netwatch-specific logs
func (s *MikroTikService) parseNetwatchLogs(logData string) []MikroTikLog {
	lines := strings.Split(logData, "\n")
	var logs []MikroTikLog

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		log := s.parseNetwatchLogLine(line)
		if log != nil {
			logs = append(logs, *log)
		}
	}

	// Reverse to show newest first
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	return logs
}

// parseNetwatchLogLine parses a single netwatch log line
func (s *MikroTikService) parseNetwatchLogLine(line string) *MikroTikLog {
	// Parse Netwatch logs - they typically contain UP/DOWN status
	// Example format: "netwatch host 192.168.1.100 DOWN"
	// or more detailed format with timestamp and message

	// Check if this is a Netwatch UP/DOWN log
	if strings.Contains(strings.ToLower(line), "netwatch") {
		// Extract host/IP address
		var host string
		var status string
		var message string = line

		// Try to extract IP address pattern
		if strings.Contains(line, ".") {
			// Look for IP address in the line
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Count(part, ".") == 3 {
					host = part
					break
				}
			}
		}

		// Determine status from the log content
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "down") || strings.Contains(lineLower, "timeout") || strings.Contains(lineLower, "unreachable") {
			status = "DOWN"
		} else if strings.Contains(lineLower, "up") || strings.Contains(lineLower, "reachable") || strings.Contains(lineLower, "alive") {
			status = "UP"
		} else {
			// Default based on log level
			if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "critical") {
				status = "DOWN"
			} else {
				status = "UP"
			}
		}

		if host == "" {
			host = "unknown"
		}

		return &MikroTikLog{
			Host:      host,
			Comment:   message,
			Status:    status,
			Timestamp: time.Now(), // In real implementation, parse actual timestamp
			Type:      s.getLogType(status),
			Category:  "netwatch",
			Raw:       line,
		}
	}

	return nil
}

// PPPoEActiveSession represents an active PPPoE subscriber session on MikroTik
type PPPoEActiveSession struct {
	ID              string `json:"id"`
	Name            string `json:"name"`      // PPPoE username / account
	Service         string `json:"service"`   // "pppoe"
	CallerID        string `json:"caller_id"` // MAC address of customer router/CVT
	Address         string `json:"address"`   // Assigned IP address
	Uptime          string `json:"uptime"`    // Uptime duration string (e.g. "2h15m")
	Encoding        string `json:"encoding,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	CustomerID      string `json:"customer_id,omitempty"`
	CustomerName    string `json:"customer_name,omitempty"`
	CustomerPhone   string `json:"customer_phone,omitempty"`
	CustomerAddress string `json:"customer_address,omitempty"`
	AreaName        string `json:"area_name,omitempty"`
	Status          string `json:"status"` // "up" / "online"
	IsMatched       bool   `json:"is_matched"`
}

// GetActivePPPoESessions retrieves active PPPoE sessions from MikroTik and cross-references them with the CRM customers
func (s *MikroTikService) GetActivePPPoESessions(dbs ...*gorm.DB) ([]PPPoEActiveSession, error) {
	if !s.IsConnected() {
		if err := s.ReconnectWithBackoff(3); err != nil {
			return nil, fmt.Errorf("mikrotik not connected: %w", err)
		}
	}

	// 1. Execute RouterOS command to print detail of active PPP sessions
	output, err := s.ExecuteCommand("/ppp active print detail")
	if err != nil {
		// Fallback to simple print
		output, err = s.ExecuteCommand("/ppp active print")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch active PPPoE sessions: %w", err)
		}
	}

	// 2. Parse sessions from output
	sessions := s.parsePPPoEActiveOutput(output)

	// 3. Match with database customers if DB is provided
	var db *gorm.DB
	if len(dbs) > 0 && dbs[0] != nil {
		db = dbs[0]
	}

	if db != nil && len(sessions) > 0 {
		sessions = s.matchSessionsWithCustomers(sessions, db)
	}

	return sessions, nil
}

func scanRouterOSFields(line string) map[string]string {
	fields := make(map[string]string)
	runes := []rune(strings.TrimSpace(line))
	for i := 0; i < len(runes); {
		for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
			i++
		}
		if i >= len(runes) {
			break
		}

		start := i
		for i < len(runes) && runes[i] != '=' && runes[i] != ' ' && runes[i] != '\t' {
			i++
		}
		if i >= len(runes) || runes[i] != '=' {
			for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
				i++
			}
			continue
		}

		key := strings.ToLower(strings.TrimSpace(string(runes[start:i])))
		i++
		for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
			i++
		}

		var value strings.Builder
		if i < len(runes) && runes[i] == '"' {
			i++
			for i < len(runes) {
				if runes[i] == '"' && (i == 0 || runes[i-1] != '\\') {
					i++
					break
				}
				value.WriteRune(runes[i])
				i++
			}
		} else {
			for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
				value.WriteRune(runes[i])
				i++
			}
		}
		fields[key] = strings.TrimSpace(value.String())
	}
	return fields
}

// parsePPPoEActiveOutput parses both detail and tabular output from RouterOS /ppp active print
func (s *MikroTikService) parsePPPoEActiveOutput(output string) []PPPoEActiveSession {
	var sessions []PPPoEActiveSession
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Flags:") || strings.HasPrefix(line, "Columns:") {
			continue
		}

		// Detail mode: e.g. "0 name=\"user1\" service=pppoe caller-id=\"40:EE:15:C8:67:5D\" address=10.10.21.10 uptime=2h15m"
		if strings.Contains(line, "name=") || strings.Contains(line, "service=") {
			sess := PPPoEActiveSession{
				Service: "pppoe",
				Status:  "up",
			}
			fields := scanRouterOSFields(line)
			for key, val := range fields {
				switch key {
				case "name":
					sess.Name = val
				case "service":
					sess.Service = val
				case "caller-id":
					sess.CallerID = val
				case "address":
					sess.Address = val
				case "uptime":
					sess.Uptime = val
				case "encoding":
					sess.Encoding = val
				case "session-id":
					sess.SessionID = val
				}
			}

			if sess.Name != "" || sess.Address != "" {
				sessions = append(sessions, sess)
			}
			continue
		}

		// Tabular mode: e.g. "0 user1 pppoe 40:EE:15:C8:67:5D 10.10.21.10 2h15m"
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			// Skip header row if "# NAME SERVICE ..."
			if strings.EqualFold(fields[0], "#") || strings.EqualFold(fields[1], "NAME") {
				continue
			}

			offset := 0
			id := fields[0]
			// Account for flags like "0 R user1 ..."
			if len(fields) >= 6 && (fields[1] == "R" || fields[1] == "X") {
				offset = 1
			}

			if len(fields) > 4+offset {
				sess := PPPoEActiveSession{
					ID:       id,
					Name:     fields[1+offset],
					Service:  fields[2+offset],
					CallerID: fields[3+offset],
					Address:  fields[4+offset],
					Status:   "up",
				}
				if len(fields) > 5+offset {
					sess.Uptime = fields[5+offset]
				}
				sessions = append(sessions, sess)
			}
		}
	}

	return sessions
}

// matchSessionsWithCustomers maps active sessions to CRM customers via IP, MAC, or username
func (s *MikroTikService) matchSessionsWithCustomers(sessions []PPPoEActiveSession, db *gorm.DB) []PPPoEActiveSession {
	var networkDevices []entities.NetworkDevice
	if err := db.Preload("Customer").Preload("Customer.Area").Find(&networkDevices).Error; err != nil {
		log.Printf("[MikroTik] Warning loading network devices for PPPoE matching: %v", err)
		return sessions
	}

	byIP := make(map[string]*entities.NetworkDevice)
	byMAC := make(map[string]*entities.NetworkDevice)
	for i := range networkDevices {
		dev := &networkDevices[i]
		if dev.IPStatic != nil && *dev.IPStatic != "" {
			cleanIP := strings.TrimSpace(strings.Split(*dev.IPStatic, "/")[0])
			byIP[cleanIP] = dev
		}
		if dev.MacAddress != nil && *dev.MacAddress != "" {
			cleanMAC := strings.ToLower(strings.TrimSpace(*dev.MacAddress))
			byMAC[cleanMAC] = dev
		}
	}

	var customers []entities.Customer
	if err := db.Preload("Area").Find(&customers).Error; err == nil {
		byUsername := make(map[string]*entities.Customer)
		for i := range customers {
			cust := &customers[i]
			if cust.Alias != "" {
				byUsername[strings.ToLower(strings.TrimSpace(cust.Alias))] = cust
			}
			if cust.Phone != "" {
				byUsername[strings.TrimSpace(cust.Phone)] = cust
				normalizedPhone := strings.Map(func(r rune) rune {
					if r >= '0' && r <= '9' {
						return r
					}
					return -1
				}, cust.Phone)
				if normalizedPhone != "" {
					byUsername[normalizedPhone] = cust
				}
			}
			if cust.Name != "" {
				byUsername[strings.ToLower(strings.TrimSpace(cust.Name))] = cust
			}
			byUsername[cust.ID] = cust
		}

		for i := range sessions {
			sess := &sessions[i]
			sess.Status = "up"

			// Match by IP
			sessionIP := strings.TrimSpace(strings.Split(sess.Address, "/")[0])
			if dev, found := byIP[sessionIP]; found && dev.Customer != nil {
				sess.CustomerID = dev.Customer.ID
				sess.CustomerName = dev.Customer.Name
				sess.CustomerPhone = dev.Customer.Phone
				sess.CustomerAddress = dev.Customer.Address
				if dev.Customer.Area != nil {
					sess.AreaName = dev.Customer.Area.NameVillage
				}
				sess.IsMatched = true
				continue
			}

			// Match by MAC
			cleanCaller := strings.ToLower(strings.TrimSpace(sess.CallerID))
			if dev, found := byMAC[cleanCaller]; found && dev.Customer != nil {
				sess.CustomerID = dev.Customer.ID
				sess.CustomerName = dev.Customer.Name
				sess.CustomerPhone = dev.Customer.Phone
				sess.CustomerAddress = dev.Customer.Address
				if dev.Customer.Area != nil {
					sess.AreaName = dev.Customer.Area.NameVillage
				}
				sess.IsMatched = true
				continue
			}

			// Match by Username
			cleanName := strings.ToLower(strings.TrimSpace(sess.Name))
			if cust, found := byUsername[cleanName]; found {
				sess.CustomerID = cust.ID
				sess.CustomerName = cust.Name
				sess.CustomerPhone = cust.Phone
				sess.CustomerAddress = cust.Address
				if cust.Area != nil {
					sess.AreaName = cust.Area.NameVillage
				}
				sess.IsMatched = true
				continue
			}
			normalizedSessionName := strings.Map(func(r rune) rune {
				if r >= '0' && r <= '9' {
					return r
				}
				return -1
			}, sess.Name)
			if normalizedSessionName != "" {
				if cust, found := byUsername[normalizedSessionName]; found {
					sess.CustomerID = cust.ID
					sess.CustomerName = cust.Name
					sess.CustomerPhone = cust.Phone
					sess.CustomerAddress = cust.Address
					if cust.Area != nil {
						sess.AreaName = cust.Area.NameVillage
					}
					sess.IsMatched = true
				}
			}
		}
	}

	return sessions
}
