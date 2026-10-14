package services

import (
	"bufio"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const routerOSAPITimeout = 10 * time.Second

type RouterOSAPIConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	UseTLS   bool
}

type RouterOSAPI struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
	config RouterOSAPIConfig
}

func ConnectRouterOSAPI(config RouterOSAPIConfig) (*RouterOSAPI, error) {
	if strings.TrimSpace(config.Host) == "" || config.Port < 1 || config.Port > 65535 {
		return nil, fmt.Errorf("valid RouterOS API host and port are required")
	}
	if config.Username == "" || config.Password == "" {
		return nil, fmt.Errorf("RouterOS API username and password are required")
	}

	address := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	dialer := &net.Dialer{Timeout: routerOSAPITimeout}
	var (
		conn net.Conn
		err  error
	)
	if config.UseTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: config.Host,
		})
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RouterOS API: %w", err)
	}

	client := &RouterOSAPI{
		conn:   conn,
		reader: bufio.NewReader(conn),
		config: config,
	}
	if err := client.login(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	client.config.Password = ""
	return client, nil
}

func (c *RouterOSAPI) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *RouterOSAPI) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *RouterOSAPI) Host() string {
	return c.config.Host
}

func (c *RouterOSAPI) login() error {
	reply, err := c.runLocked("/login", "=name="+c.config.Username, "=password="+c.config.Password)
	if err == nil {
		if len(reply) == 0 || reply[0]["ret"] == "" {
			return nil
		}
		return c.loginWithChallenge(reply[0]["ret"])
	}

	legacyReply, legacyErr := c.runLocked("/login")
	if legacyErr != nil {
		return fmt.Errorf("RouterOS API login failed: %w (legacy login attempt: %v)", err, legacyErr)
	}
	if len(legacyReply) == 0 || legacyReply[0]["ret"] == "" {
		return fmt.Errorf("RouterOS API login failed: %w", err)
	}
	return c.loginWithChallenge(legacyReply[0]["ret"])
}

func (c *RouterOSAPI) loginWithChallenge(challenge string) error {
	challengeBytes, err := hex.DecodeString(challenge)
	if err != nil {
		return fmt.Errorf("RouterOS API returned an invalid login challenge: %w", err)
	}
	hash := md5.New()
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(c.config.Password))
	_, _ = hash.Write(challengeBytes)
	response := "00" + hex.EncodeToString(hash.Sum(nil))
	_, err = c.runLocked("/login", "=name="+c.config.Username, "=response="+response)
	if err != nil {
		return fmt.Errorf("RouterOS API challenge login failed: %w", err)
	}
	return nil
}

func (c *RouterOSAPI) Command(command string, attributes ...string) ([]map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, fmt.Errorf("RouterOS API is not connected")
	}
	return c.runLocked(command, attributes...)
}

func (c *RouterOSAPI) runLocked(command string, attributes ...string) ([]map[string]string, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("RouterOS API is not connected")
	}
	if err := c.conn.SetDeadline(time.Now().Add(routerOSAPITimeout)); err != nil {
		c.invalidate()
		return nil, fmt.Errorf("failed to set RouterOS API timeout: %w", err)
	}
	defer c.conn.SetDeadline(time.Time{})

	if err := writeRouterOSSentence(c.conn, append([]string{command}, attributes...)); err != nil {
		c.invalidate()
		return nil, fmt.Errorf("failed to send RouterOS API command: %w", err)
	}

	var records []map[string]string
	for {
		sentence, err := readRouterOSSentence(c.reader)
		if err != nil {
			c.invalidate()
			return nil, fmt.Errorf("failed to read RouterOS API response: %w", err)
		}
		switch sentence[0] {
		case "!re":
			records = append(records, routerOSRecord(sentence[1:]))
		case "!done":
			if len(sentence) > 1 {
				records = append(records, routerOSRecord(sentence[1:]))
			}
			return records, nil
		case "!trap":
			message := routerOSRecord(sentence[1:])["message"]
			if message == "" {
				message = "router rejected the command"
			}
			return nil, fmt.Errorf("RouterOS API command failed: %s", message)
		case "!fatal":
			message := routerOSRecord(sentence[1:])["message"]
			c.invalidate()
			if message == "" {
				message = "RouterOS API connection was terminated"
			}
			return nil, fmt.Errorf("RouterOS API command failed: %s", message)
		}
	}
}

func (c *RouterOSAPI) invalidate() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func routerOSRecord(words []string) map[string]string {
	record := make(map[string]string, len(words))
	for _, word := range words {
		if !strings.HasPrefix(word, "=") {
			continue
		}
		key, value, ok := strings.Cut(word[1:], "=")
		if ok {
			record[key] = value
		}
	}
	return record
}

func writeRouterOSSentence(w io.Writer, words []string) error {
	for _, word := range words {
		if err := writeRouterOSWord(w, []byte(word)); err != nil {
			return err
		}
	}
	return writeRouterOSBytes(w, []byte{0})
}

func writeRouterOSWord(w io.Writer, word []byte) error {
	length := len(word)
	var prefix []byte
	switch {
	case length < 0x80:
		prefix = []byte{byte(length)}
	case length < 0x4000:
		prefix = []byte{byte(length>>8) | 0x80, byte(length)}
	case length < 0x200000:
		prefix = []byte{byte(length>>16) | 0xC0, byte(length >> 8), byte(length)}
	case length < 0x10000000:
		prefix = []byte{byte(length>>24) | 0xE0, byte(length >> 16), byte(length >> 8), byte(length)}
	default:
		prefix = []byte{0xF0, byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length)}
	}
	if err := writeRouterOSBytes(w, prefix); err != nil {
		return err
	}
	return writeRouterOSBytes(w, word)
}

func writeRouterOSBytes(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func readRouterOSSentence(r io.Reader) ([]string, error) {
	var words []string
	for {
		length, err := readRouterOSLength(r)
		if err != nil {
			return nil, err
		}
		if length == 0 {
			if len(words) == 0 {
				return nil, fmt.Errorf("RouterOS API returned an empty sentence")
			}
			return words, nil
		}
		if length > 16*1024*1024 {
			return nil, fmt.Errorf("RouterOS API response word exceeds the 16 MiB limit")
		}
		word := make([]byte, length)
		if _, err := io.ReadFull(r, word); err != nil {
			return nil, err
		}
		words = append(words, string(word))
	}
}

func readRouterOSLength(r io.Reader) (int, error) {
	var first [1]byte
	if _, err := io.ReadFull(r, first[:]); err != nil {
		return 0, err
	}
	b := first[0]
	switch {
	case b&0x80 == 0:
		return int(b), nil
	case b&0xC0 == 0x80:
		var rest [1]byte
		if _, err := io.ReadFull(r, rest[:]); err != nil {
			return 0, err
		}
		return int(b&0x3F)<<8 | int(rest[0]), nil
	case b&0xE0 == 0xC0:
		var rest [2]byte
		if _, err := io.ReadFull(r, rest[:]); err != nil {
			return 0, err
		}
		return int(b&0x1F)<<16 | int(rest[0])<<8 | int(rest[1]), nil
	case b&0xF0 == 0xE0:
		var rest [3]byte
		if _, err := io.ReadFull(r, rest[:]); err != nil {
			return 0, err
		}
		return int(b&0x0F)<<24 | int(rest[0])<<16 | int(rest[1])<<8 | int(rest[2]), nil
	case b == 0xF0:
		var rest [4]byte
		if _, err := io.ReadFull(r, rest[:]); err != nil {
			return 0, err
		}
		return int(rest[0])<<24 | int(rest[1])<<16 | int(rest[2])<<8 | int(rest[3]), nil
	default:
		return 0, fmt.Errorf("invalid RouterOS API word length prefix")
	}
}
