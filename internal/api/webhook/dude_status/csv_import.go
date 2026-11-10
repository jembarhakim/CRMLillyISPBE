package dude_status

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"unicode"

	"skripsi-be/internal/models/entities"
)

var (
	ErrInvalidCSV         = errors.New("CSV must contain Name, Addresses, and Maps columns")
	ErrCSVMapRequired     = errors.New("map_name is required")
	ErrCSVMapHasNoDevices = errors.New("selected map contains no rows with valid IP addresses")
)

type CSVMapPreview struct {
	Name    string `json:"name"`
	Rows    int    `json:"rows"`
	Devices int    `json:"devices"`
	Skipped int    `json:"skipped"`
}

type CSVPreview struct {
	Rows    int             `json:"rows"`
	Skipped int             `json:"skipped"`
	Maps    []CSVMapPreview `json:"maps"`
}

type CSVImportResult struct {
	ImportResult
	MapName string `json:"map_name"`
	Skipped int    `json:"skipped"`
}

type parsedCSVRow struct {
	mapNames []string
	devices  []entities.DudeDeviceStatus
	skipped  int
}

func (r parsedCSVRow) hasMap(name string) bool {
	for _, mapName := range r.mapNames {
		if mapName == name {
			return true
		}
	}
	return false
}

func parseDudeCSV(reader io.Reader) ([]parsedCSVRow, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	csvReader.ReuseRecord = false

	header, err := csvReader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrInvalidCSV
		}
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for i, value := range header {
		columns[normalizeCSVHeader(value)] = i
	}
	nameColumn, hasName := columns["name"]
	addressColumn, hasAddresses := columns["addresses"]
	mapsColumn, hasMaps := columns["maps"]
	if !hasName || !hasAddresses || !hasMaps {
		return nil, ErrInvalidCSV
	}
	flagColumn, hasFlag := columns["flag"]

	var rows []parsedCSVRow
	line := 1
	for {
		record, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		line++
		if err != nil {
			return nil, fmt.Errorf("read CSV row %d: %w", line, err)
		}
		if isBlankCSVRecord(record) {
			continue
		}

		name := csvField(record, nameColumn)
		mapNames := parseMapNames(csvField(record, mapsColumn))
		if name == "" || len(mapNames) == 0 {
			rows = append(rows, parsedCSVRow{skipped: 1})
			continue
		}

		status := "unknown"
		if hasFlag {
			status = normalizeImportedStatus(csvField(record, flagColumn))
		}
		row := parsedCSVRow{mapNames: mapNames}
		for _, address := range parseAddresses(csvField(record, addressColumn)) {
			ip := net.ParseIP(address)
			if ip == nil {
				row.skipped++
				continue
			}
			row.devices = append(row.devices, entities.DudeDeviceStatus{
				Name:      name,
				IPAddress: ip.String(),
				Status:    status,
			})
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func normalizeCSVHeader(value string) string {
	var normalized strings.Builder
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			normalized.WriteRune(char)
		}
	}
	return normalized.String()
}

func csvField(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func isBlankCSVRecord(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}

func parseMapNames(value string) []string {
	seen := make(map[string]struct{})
	var names []string
	for _, name := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func parseAddresses(value string) []string {
	var addresses []string
	for _, entry := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			entry = network.IP.String()
		}
		addresses = append(addresses, entry)
	}
	return addresses
}

func normalizeImportedStatus(value string) string {
	switch strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " ")) {
	case "up", "online", "ok":
		return "up"
	case "down", "offline":
		return "down"
	case "partially down", "partial down":
		return "partially down"
	default:
		return "unknown"
	}
}
