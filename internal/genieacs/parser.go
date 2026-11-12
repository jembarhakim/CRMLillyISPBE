package genieacs

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var numberRegex = regexp.MustCompile(`[-+]?[0-9]*\.?[0-9]+([eE][-+]?[0-9]+)?`)

// macRegex matches a colon or dash separated MAC address
var macRegex = regexp.MustCompile(`(?i)^[0-9a-f]{2}([:-][0-9a-f]{2}){5}$`)

// ExtractNestedValue retrieves a parameter from GenieACS NBI's nested map structure using dot notation.
// For example: "InternetGatewayDevice.WANDevice.1.X_GponInterfaceConfig.RXPower"
//
// A "*" segment acts as a wildcard that matches the first child carrying the
// requested key, so paths like "InternetGatewayDevice.WANDevice.*.X_GponInterfaceConfig.RXPower"
// keep working when the GPON interface is not at index 1.
// Returns value, timestamp, and boolean success.
func ExtractNestedValue(doc map[string]interface{}, dotPath string) (interface{}, time.Time, bool) {
	if doc == nil || dotPath == "" {
		return nil, time.Time{}, false
	}

	parts := strings.Split(dotPath, ".")
	var current interface{} = doc

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, time.Time{}, false
		}

		if part == "*" {
			// Wildcard is only meaningful when more segments follow
			current = firstChildOf(m)
			if current == nil {
				return nil, time.Time{}, false
			}
			continue
		}

		val, exists := m[part]
		if !exists {
			return nil, time.Time{}, false
		}
		current = val
	}

	// Leaf parameter in GenieACS NBI is typically a map containing "_value" and "_timestamp"
	if leafMap, ok := current.(map[string]interface{}); ok {
		val := leafMap["_value"]
		var ts time.Time
		if tsStr, ok := leafMap["_timestamp"].(string); ok && tsStr != "" {
			if parsed, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
				ts = parsed
			}
		}
		return val, ts, true
	}

	// Direct primitive value
	return current, time.Time{}, true
}

// ExtractNestedValueAny is like ExtractNestedValue but tries every children branch
// for wildcard segments and returns the first branch that resolves the full path.
func ExtractNestedValueAny(doc map[string]interface{}, dotPath string) (interface{}, time.Time, bool) {
	if doc == nil || dotPath == "" {
		return nil, time.Time{}, false
	}

	parts := strings.Split(dotPath, ".")
	return extractWithWildcard(doc, parts)
}

// extractWithWildcard walks the path recursively so wildcard segments can be
// resolved against every available branch instead of only the first one.
func extractWithWildcard(node interface{}, parts []string) (interface{}, time.Time, bool) {
	if len(parts) == 0 {
		if leafMap, ok := node.(map[string]interface{}); ok {
			if val, exists := leafMap["_value"]; exists {
				var ts time.Time
				if tsStr, ok := leafMap["_timestamp"].(string); ok && tsStr != "" {
					if parsed, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
						ts = parsed
					}
				}
				return val, ts, true
			}
			return node, time.Time{}, true
		}
		return node, time.Time{}, true
	}

	m, ok := node.(map[string]interface{})
	if !ok {
		return nil, time.Time{}, false
	}

	head := parts[0]
	rest := parts[1:]

	if head == "*" {
		for _, child := range m {
			if val, ts, ok := extractWithWildcard(child, rest); ok {
				return val, ts, true
			}
		}
		return nil, time.Time{}, false
	}

	next, exists := m[head]
	if !exists {
		return nil, time.Time{}, false
	}
	return extractWithWildcard(next, rest)
}

// firstChildOf returns the first map child of m (used for single-level wildcards)
func firstChildOf(m map[string]interface{}) interface{} {
	for _, child := range m {
		if _, ok := child.(map[string]interface{}); ok {
			return child
		}
	}
	return nil
}

// ParseFloatValue cleans and extracts a float64 from an interface{} value, applying the specified scale factor.
// Handles strings like "-19.50", "-26.14 dBm", raw numbers, and milli-dBm scaling.
func ParseFloatValue(raw interface{}, scale float64) (*float64, error) {
	if raw == nil {
		return nil, fmt.Errorf("value is nil")
	}

	if scale == 0 {
		scale = 1.0
	}

	switch v := raw.(type) {
	case float64:
		res := v * scale
		return &res, nil
	case float32:
		res := float64(v) * scale
		return &res, nil
	case int:
		res := float64(v) * scale
		return &res, nil
	case int64:
		res := float64(v) * scale
		return &res, nil
	case string:
		clean := strings.TrimSpace(v)
		if clean == "" {
			return nil, fmt.Errorf("empty string")
		}

		match := numberRegex.FindString(clean)
		if match == "" {
			return nil, fmt.Errorf("no numeric component in %q", clean)
		}

		parsed, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q: %w", match, err)
		}

		res := parsed * scale
		return &res, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", raw)
	}
}

// ExtractStringValue extracts string from nested path
func ExtractStringValue(doc map[string]interface{}, dotPath string) string {
	raw, _, ok := ExtractNestedValue(doc, dotPath)
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ExtractIntValue extracts integer from nested path
func ExtractIntValue(doc map[string]interface{}, dotPath string) int {
	raw, _, ok := ExtractNestedValue(doc, dotPath)
	if !ok || raw == nil {
		return 0
	}
	return toInt(raw)
}

// hostFromURL extracts the host (without port) from a URL such as
// "http://10.10.9.101:7547/cwmp". Returns "" when no host can be determined.
func hostFromURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	if host == "" {
		return ""
	}
	if !isValidIPv4(host) {
		return ""
	}
	return host
}

// isValidIPv4 reports whether value is a syntactically valid dotted-quad IPv4 address
// and is not the placeholder 0.0.0.0 that ONTs often report before getting a lease.
func isValidIPv4(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	parsed := net.ParseIP(value)
	if parsed == nil || parsed.To4() == nil {
		return false
	}
	return value != "0.0.0.0"
}

// looksLikeMac reports whether value looks like a MAC address (xx:xx:xx:xx:xx:xx or xx-xx-...)
func looksLikeMac(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	match := macRegex.FindString(value)
	return match != ""
}

// toInt converts an arbitrary GenieACS value into an integer
func toInt(raw interface{}) int {
	switch v := raw.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case bool:
		if v {
			return 1
		}
		return 0
	case string:
		match := numberRegex.FindString(v)
		if match != "" {
			if f, err := strconv.ParseFloat(match, 64); err == nil {
				return int(f)
			}
		}
	}
	return 0
}

// MatchModelProfile finds a matching vendor profile based on manufacturer and product class regex.
// Profiles are evaluated in declaration order; catch-all/generic profiles are always tried last
// so that a specific vendor profile cannot be shadowed by the generic fallback.
func MatchModelProfile(manufacturer, productClass string, cfg *ONTModelsConfig) *ModelProfile {
	if cfg == nil || len(cfg.Models) == 0 {
		return nil
	}

	var generic *ModelProfile

	for i := range cfg.Models {
		profile := &cfg.Models[i]

		// Remember the catch-all profile instead of returning it immediately
		if isGenericProfile(profile) {
			if generic == nil {
				generic = profile
			}
			continue
		}

		if profile.Match.Manufacturer != "" {
			reMan, err := regexp.Compile(profile.Match.Manufacturer)
			if err != nil || !reMan.MatchString(manufacturer) {
				continue
			}
		}
		if profile.Match.ProductClass != "" {
			rePC, err := regexp.Compile(profile.Match.ProductClass)
			if err != nil || !rePC.MatchString(productClass) {
				continue
			}
		}

		return profile
	}

	return generic
}

// isGenericProfile reports whether the profile is a catch-all entry
// (either keyed as generic_fallback or matching everything with ".*").
func isGenericProfile(profile *ModelProfile) bool {
	if profile == nil {
		return false
	}
	if strings.EqualFold(profile.Key, "generic_fallback") {
		return true
	}
	man := strings.TrimSpace(profile.Match.Manufacturer)
	pc := strings.TrimSpace(profile.Match.ProductClass)
	return (man == ".*" || man == "") && (pc == ".*" || pc == "")
}

// extractByPath resolves a dot path, automatically switching to the wildcard-aware
// traversal when the path contains a "*" segment.
func extractByPath(doc map[string]interface{}, dotPath string) (interface{}, time.Time, bool) {
	if strings.Contains(dotPath, "*") {
		return ExtractNestedValueAny(doc, dotPath)
	}
	return ExtractNestedValue(doc, dotPath)
}

// extractFloatByPath resolves a numeric parameter using scale, preferring wildcard traversal
func extractFloatByPath(doc map[string]interface{}, dotPath string, scale float64) *float64 {
	if dotPath == "" {
		return nil
	}
	val, _, ok := extractByPath(doc, dotPath)
	if !ok {
		return nil
	}
	parsed, err := ParseFloatValue(val, scale)
	if err != nil {
		return nil
	}
	return parsed
}

// extractStringByPath resolves a string parameter, preferring wildcard traversal
func extractStringByPath(doc map[string]interface{}, dotPath string) string {
	if dotPath == "" {
		return ""
	}
	val, _, ok := extractByPath(doc, dotPath)
	if !ok || val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return strings.TrimSpace(s)
	}
	return fmt.Sprintf("%v", val)
}

// extractIntByPath resolves an integer parameter, preferring wildcard traversal
func extractIntByPath(doc map[string]interface{}, dotPath string) int {
	if dotPath == "" {
		return 0
	}
	val, _, ok := extractByPath(doc, dotPath)
	if !ok || val == nil {
		return 0
	}
	return toInt(val)
}

// ExtractMetrics parses a GenieACS device document into a clean OntMetrics struct
func ExtractMetrics(doc map[string]interface{}, cfg *ONTModelsConfig) *OntMetrics {
	if doc == nil {
		return nil
	}
	if cfg == nil {
		cfg = DefaultConfig()
	}

	metrics := &OntMetrics{
		OpticalStatus: OpticalStatusUnknown,
	}

	// 1. Device ID & root properties
	if id, ok := doc["_id"].(string); ok {
		metrics.DeviceID = id
	}

	// Serial number and manufacturer from _deviceId or parameters
	if devId, ok := doc["_deviceId"].(map[string]interface{}); ok {
		if sn, ok := devId["_SerialNumber"].(string); ok {
			metrics.SerialNumber = sn
		}
		if pc, ok := devId["_ProductClass"].(string); ok {
			metrics.ModelName = pc
		}
		if man, ok := devId["_Manufacturer"].(string); ok {
			metrics.Manufacturer = man
		}
	}

	// Fallbacks if _deviceId was missing
	if metrics.SerialNumber == "" {
		metrics.SerialNumber = ExtractStringValue(doc, "InternetGatewayDevice.DeviceInfo.SerialNumber")
		if metrics.SerialNumber == "" {
			metrics.SerialNumber = ExtractStringValue(doc, "Device.DeviceInfo.SerialNumber")
		}
	}
	if metrics.ModelName == "" {
		metrics.ModelName = ExtractStringValue(doc, "InternetGatewayDevice.DeviceInfo.ModelName")
		if metrics.ModelName == "" {
			metrics.ModelName = ExtractStringValue(doc, "Device.DeviceInfo.ModelName")
		}
	}
	if metrics.Manufacturer == "" {
		metrics.Manufacturer = ExtractStringValue(doc, "InternetGatewayDevice.DeviceInfo.Manufacturer")
		if metrics.Manufacturer == "" {
			metrics.Manufacturer = ExtractStringValue(doc, "Device.DeviceInfo.Manufacturer")
		}
	}

	// Software version
	metrics.SoftwareVersion = ExtractStringValue(doc, "InternetGatewayDevice.DeviceInfo.SoftwareVersion")
	if metrics.SoftwareVersion == "" {
		metrics.SoftwareVersion = ExtractStringValue(doc, "Device.DeviceInfo.SoftwareVersion")
	}

	// 1b. IP address — cara paling andal (PBB dihasilkan oleh GenieACS/ACD)
	if ip := ExtractStringValue(doc, "_tags._ipAddress"); ip != "" {
		metrics.IPAddress = ip
	}
	// GenieACS juga menyimpan _ipAddress di root dokumen (diambil dari ConnectionRequestURL)
	if metrics.IPAddress == "" {
		if ip := ExtractStringValue(doc, "_ipAddress"); ip != "" {
			metrics.IPAddress = ip
		}
	}
	// ConnectionRequestURL: host-nya adalah IP publik perangkat yang dilihat ACS
	if metrics.IPAddress == "" {
		if url := extractStringByPath(doc, "InternetGatewayDevice.ManagementServer.ConnectionRequestURL"); url != "" {
			metrics.IPAddress = hostFromURL(url)
		}
	}
	if metrics.IPAddress == "" {
		if url := extractStringByPath(doc, "Device.ManagementServer.ConnectionRequestURL"); url != "" {
			metrics.IPAddress = hostFromURL(url)
		}
	}
	// TR-098: IP WAN yang diterima dari OLT / BRAS
	if metrics.IPAddress == "" {
		metrics.IPAddress = extractStringByPath(doc,
			"InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.ExternalIPAddress")
	}
	// TR-181: alamat IPv4 pada interface WAN
	if metrics.IPAddress == "" {
		metrics.IPAddress = extractStringByPath(doc, "Device.IP.Interface.1.IPv4Address.1.IPAddress")
	}
	// TR-098: IP LAN bila WAN tidak tersedia
	if metrics.IPAddress == "" {
		metrics.IPAddress = extractStringByPath(doc,
			"InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.IPInterface.1.IPInterfaceIPAddress")
	}
	// TR-181: IP LAN
	if metrics.IPAddress == "" {
		metrics.IPAddress = extractStringByPath(doc, "Device.IP.Interface.2.IPv4Address.1.IPAddress")
	}
	if metrics.IPAddress != "" && !isValidIPv4(metrics.IPAddress) {
		metrics.IPAddress = ""
	}

	// 1c. MAC address perangkat
	metrics.MacAddress = extractStringByPath(doc,
		"InternetGatewayDevice.LANDevice.1.LANDeviceInfo.1.LANEthernetInterfaceConfig.1.MACAddress")
	if metrics.MacAddress == "" {
		metrics.MacAddress = extractStringByPath(doc, "Device.Ethernet.Interface.1.MACAddress")
	}
	if metrics.MacAddress == "" {
		metrics.MacAddress = extractStringByPath(doc,
			"InternetGatewayDevice.LANDevice.*.LANEthernetInterfaceConfig.1.MACAddress")
	}
	if metrics.MacAddress != "" && !looksLikeMac(metrics.MacAddress) {
		metrics.MacAddress = ""
	}

	// Last inform timestamp
	if lastInfStr, ok := doc["_lastInform"].(string); ok && lastInfStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastInfStr); err == nil {
			metrics.LastInform = &t
		}
	}

	// 2. Determine Online Status
	// Offline if now - lastInform > (multiplier * inform_interval)
	informInterval := cfg.Thresholds.DefaultInformInterval
	if customInterval := ExtractIntValue(doc, "InternetGatewayDevice.ManagementServer.PeriodicInformInterval"); customInterval > 0 {
		informInterval = customInterval
	} else if customInterval := ExtractIntValue(doc, "Device.ManagementServer.PeriodicInformInterval"); customInterval > 0 {
		informInterval = customInterval
	}

	timeoutDuration := time.Duration(float64(informInterval)*cfg.Thresholds.OfflineTimeoutMultiplier) * time.Second
	if metrics.LastInform != nil {
		metrics.IsOnline = time.Since(*metrics.LastInform) <= timeoutDuration
	} else {
		metrics.IsOnline = false
	}

	// 3. Extract Optical & WiFi Parameters using matched model or candidates
	profile := MatchModelProfile(metrics.Manufacturer, metrics.ModelName, cfg)

	var rxExtracted, txExtracted *float64

	// Try profile parameters if matched
	if profile != nil && profile.Parameters != nil {
		// Rx Power
		if profile.HasParameter("rx_power") {
			rxCfg := profile.Parameter("rx_power")
			rxExtracted = extractFloatByPath(doc, rxCfg.Path, rxCfg.Scale)
			if rxExtracted == nil && rxCfg.FallbackPath != "" {
				rxExtracted = extractFloatByPath(doc, rxCfg.FallbackPath, rxCfg.Scale)
			}
		}

		// Tx Power
		if profile.HasParameter("tx_power") {
			txCfg := profile.Parameter("tx_power")
			txExtracted = extractFloatByPath(doc, txCfg.Path, txCfg.Scale)
			if txExtracted == nil && txCfg.FallbackPath != "" {
				txExtracted = extractFloatByPath(doc, txCfg.FallbackPath, txCfg.Scale)
			}
		}

		// SSID
		if profile.HasParameter("ssid") {
			ssidCfg := profile.Parameter("ssid")
			metrics.SSID = extractStringByPath(doc, ssidCfg.Path)
			if metrics.SSID == "" && ssidCfg.FallbackPath != "" {
				metrics.SSID = extractStringByPath(doc, ssidCfg.FallbackPath)
			}
		}

		// Client count
		if profile.HasParameter("client_count") {
			ccCfg := profile.Parameter("client_count")
			metrics.ClientCount = extractIntByPath(doc, ccCfg.Path)
			if metrics.ClientCount == 0 && ccCfg.FallbackPath != "" {
				metrics.ClientCount = extractIntByPath(doc, ccCfg.FallbackPath)
			}
		}

		// Software version (vendor specific path when declared)
		if profile.HasParameter("software_version") {
			svCfg := profile.Parameter("software_version")
			if sv := extractStringByPath(doc, svCfg.Path); sv != "" {
				metrics.SoftwareVersion = sv
			}
		}
	}

	// Fallback candidates if profile didn't extract Rx/Tx
	if rxExtracted == nil && profile != nil && len(profile.CandidatePaths("rx_power")) > 0 {
		for _, c := range profile.CandidatePaths("rx_power") {
			if parsed := extractFloatByPath(doc, c.Path, c.Scale); parsed != nil {
				rxExtracted = parsed
				break
			}
		}
	}

	if rxExtracted == nil {
		candidates := []MetricParam{
			{Path: "InternetGatewayDevice.WANDevice.1.X_GponInterfaceConfig.RXPower", Scale: 1.0},
			{Path: "InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.RXPower", Scale: 1.0},
			{Path: "Device.Optical.Interface.1.OpticalSignalLevel", Scale: 0.001},
			{Path: "InternetGatewayDevice.WANDevice.1.X_FH_PONInterfaceConfig.RXPower", Scale: 1.0},
			// Wildcard variants: interface index is not always 1
			{Path: "InternetGatewayDevice.WANDevice.*.X_GponInterfaceConfig.RXPower", Scale: 1.0},
			{Path: "InternetGatewayDevice.WANDevice.*.X_ZTE-COM_WANPONInterfaceConfig.RXPower", Scale: 1.0},
			{Path: "InternetGatewayDevice.WANDevice.*.X_FH_PONInterfaceConfig.RXPower", Scale: 1.0},
			{Path: "Device.Optical.Interface.*.OpticalSignalLevel", Scale: 0.001},
		}
		for _, c := range candidates {
			if parsed := extractFloatByPath(doc, c.Path, c.Scale); parsed != nil {
				rxExtracted = parsed
				break
			}
		}
	}

	if txExtracted == nil && profile != nil && len(profile.CandidatePaths("tx_power")) > 0 {
		for _, c := range profile.CandidatePaths("tx_power") {
			if parsed := extractFloatByPath(doc, c.Path, c.Scale); parsed != nil {
				txExtracted = parsed
				break
			}
		}
	}

	if txExtracted == nil {
		candidates := []MetricParam{
			{Path: "InternetGatewayDevice.WANDevice.1.X_GponInterfaceConfig.TXPower", Scale: 1.0},
			{Path: "InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.TXPower", Scale: 1.0},
			{Path: "Device.Optical.Interface.1.TransmitOpticalLevel", Scale: 0.001},
			{Path: "InternetGatewayDevice.WANDevice.1.X_FH_PONInterfaceConfig.TXPower", Scale: 1.0},
			// Wildcard variants: interface index is not always 1
			{Path: "InternetGatewayDevice.WANDevice.*.X_GponInterfaceConfig.TXPower", Scale: 1.0},
			{Path: "InternetGatewayDevice.WANDevice.*.X_ZTE-COM_WANPONInterfaceConfig.TXPower", Scale: 1.0},
			{Path: "InternetGatewayDevice.WANDevice.*.X_FH_PONInterfaceConfig.TXPower", Scale: 1.0},
			{Path: "Device.Optical.Interface.*.TransmitOpticalLevel", Scale: 0.001},
		}
		for _, c := range candidates {
			if parsed := extractFloatByPath(doc, c.Path, c.Scale); parsed != nil {
				txExtracted = parsed
				break
			}
		}
	}

	if metrics.SSID == "" {
		metrics.SSID = ExtractStringValue(doc, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID")
		if metrics.SSID == "" {
			metrics.SSID = ExtractStringValue(doc, "Device.WiFi.SSID.1.SSID")
		}
	}

	if metrics.ClientCount == 0 {
		metrics.ClientCount = ExtractIntValue(doc, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.AssociatedDeviceNumberOfEntries")
		if metrics.ClientCount == 0 {
			metrics.ClientCount = ExtractIntValue(doc, "Device.WiFi.AccessPoint.1.AssociatedDeviceNumberOfEntries")
		}
	}

	metrics.RxPower = rxExtracted
	metrics.TxPower = txExtracted

	// 4. Classify Optical Severity
	metrics.OpticalStatus = EvaluateOpticalStatus(rxExtracted, cfg.Thresholds.OpticalRx)

	// 5. Stale flag
	// If offline but has prior reading, mark as stale
	if !metrics.IsOnline && metrics.RxPower != nil {
		metrics.IsStale = true
	}

	return metrics
}

