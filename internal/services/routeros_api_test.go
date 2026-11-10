package services

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"testing"
)

func TestRouterOSWordLengthEncoding(t *testing.T) {
	for _, size := range []int{127, 128, 0x3fff, 0x4000, 0x1fffff, 0x200000} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			word := bytes.Repeat([]byte{'x'}, size)
			var encoded bytes.Buffer
			if err := writeRouterOSWord(&encoded, word); err != nil {
				t.Fatalf("writeRouterOSWord() error = %v", err)
			}
			length, err := readRouterOSLength(&encoded)
			if err != nil {
				t.Fatalf("readRouterOSLength() error = %v", err)
			}
			if length != size {
				t.Fatalf("decoded length = %d, want %d", length, size)
			}
			decoded := make([]byte, length)
			if _, err := encoded.Read(decoded); err != nil {
				t.Fatalf("reading encoded word: %v", err)
			}
			if !bytes.Equal(decoded, word) {
				t.Fatal("decoded word differs from encoded word")
			}
		})
	}
}

func TestRouterOSSentenceRoundTrip(t *testing.T) {
	want := []string{"/ip/hotspot/user/add", "=name=guest", "=comment=trial user"}
	var encoded bytes.Buffer
	if err := writeRouterOSSentence(&encoded, want); err != nil {
		t.Fatalf("writeRouterOSSentence() error = %v", err)
	}
	got, err := readRouterOSSentence(&encoded)
	if err != nil {
		t.Fatalf("readRouterOSSentence() error = %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("sentence length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("word %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRouterOSCommandConnectionFailureDoesNotPanic(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	client := &RouterOSAPI{conn: clientConn, reader: bufio.NewReader(clientConn)}
	serverResult := make(chan error, 1)
	go func() {
		defer serverConn.Close()
		_, err := readRouterOSSentence(bufio.NewReader(serverConn))
		serverResult <- err
	}()

	if _, err := client.Command("/ip/hotspot/ip-binding/print"); err == nil {
		t.Fatal("Command() error = nil, want closed connection error")
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
	if client.conn != nil {
		t.Fatal("Command() did not invalidate the failed connection")
	}
}

func TestRouterOSCommandWithNilClientReturnsError(t *testing.T) {
	var client *RouterOSAPI
	if _, err := client.Command("/ip/hotspot/ip-binding/print"); err == nil {
		t.Fatal("Command() error = nil, want disconnected error")
	}
}

func TestRouterOSCommandReconnectsWithStoredConfig(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	serverResult := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverResult <- err
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		login, err := readRouterOSSentence(reader)
		if err != nil {
			serverResult <- fmt.Errorf("read login: %w", err)
			return
		}
		wantLogin := []string{"/login", "=name=api-user", "=password=test-password"}
		if !reflect.DeepEqual(login, wantLogin) {
			serverResult <- fmt.Errorf("login = %v, want %v", login, wantLogin)
			return
		}
		if err := writeRouterOSSentence(conn, []string{"!done"}); err != nil {
			serverResult <- fmt.Errorf("write login response: %w", err)
			return
		}
		command, err := readRouterOSSentence(reader)
		if err != nil {
			serverResult <- fmt.Errorf("read command: %w", err)
			return
		}
		if !reflect.DeepEqual(command, []string{"/ip/hotspot/ip-binding/print"}) {
			serverResult <- fmt.Errorf("command = %v, want IP Binding print", command)
			return
		}
		serverResult <- writeRouterOSSentence(conn, []string{"!done"})
	}()

	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("parse listener address: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}
	client := &RouterOSAPI{config: RouterOSAPIConfig{
		Host:     "127.0.0.1",
		Port:     port,
		Username: "api-user",
		Password: "test-password",
	}}
	if _, err := client.Command("/ip/hotspot/ip-binding/print"); err != nil {
		t.Fatalf("Command() after disconnect error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
	if !client.IsConnected() {
		t.Fatal("client is not connected after successful reconnect")
	}
	_ = client.Close()
}

func TestRouterOSRecordPreservesEqualsInValue(t *testing.T) {
	record := routerOSRecord([]string{"=.id=*1", "=name=user=one", "=disabled=false"})
	if record[".id"] != "*1" {
		t.Errorf(".id = %q, want *1", record[".id"])
	}
	if record["name"] != "user=one" {
		t.Errorf("name = %q, want user=one", record["name"])
	}
	if record["disabled"] != "false" {
		t.Errorf("disabled = %q, want false", record["disabled"])
	}
}

func TestParseRouterOSPingResult(t *testing.T) {
	tests := []struct {
		name    string
		records []map[string]string
		wantUp  bool
		wantErr bool
	}{
		{name: "reply time", records: []map[string]string{{"time": "12ms"}}, wantUp: true},
		{name: "received packets", records: []map[string]string{{"sent": "1", "received": "1", "packet-loss": "0%"}}, wantUp: true},
		{name: "no received packets", records: []map[string]string{{"sent": "1", "received": "0", "packet-loss": "100%"}}, wantUp: false},
		{name: "timeout status", records: []map[string]string{{"status": "timeout"}}, wantUp: false},
		{name: "missing result", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRouterOSPingResult(tt.records)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRouterOSPingResult() error = %v, wantErr %t", err, tt.wantErr)
			}
			if err == nil && got != tt.wantUp {
				t.Errorf("parseRouterOSPingResult() = %t, want %t", got, tt.wantUp)
			}
		})
	}
}

func TestRouterOSLoginWithCurrentCredentials(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := &RouterOSAPI{
		conn:   clientConn,
		reader: bufio.NewReader(clientConn),
		config: RouterOSAPIConfig{Username: "api-user", Password: "test-password"},
	}
	serverResult := make(chan error, 1)
	go func() {
		defer serverConn.Close()
		sentence, err := readRouterOSSentence(bufio.NewReader(serverConn))
		if err != nil {
			serverResult <- err
			return
		}
		want := []string{"/login", "=name=api-user", "=password=test-password"}
		if fmt.Sprint(sentence) != fmt.Sprint(want) {
			serverResult <- fmt.Errorf("login request = %v, want %v", sentence, want)
			return
		}
		serverResult <- writeRouterOSSentence(serverConn, []string{"!done"})
	}()

	if err := client.login(); err != nil {
		t.Fatalf("login() error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestRouterOSLoginWithLegacyChallenge(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	const password = "test-password"
	const challenge = "01020304"
	client := &RouterOSAPI{
		conn:   clientConn,
		reader: bufio.NewReader(clientConn),
		config: RouterOSAPIConfig{Username: "api-user", Password: password},
	}
	serverResult := make(chan error, 1)
	go func() {
		defer serverConn.Close()
		reader := bufio.NewReader(serverConn)
		if _, err := readRouterOSSentence(reader); err != nil {
			serverResult <- err
			return
		}
		if err := writeRouterOSSentence(serverConn, []string{"!trap", "=message=legacy challenge required"}); err != nil {
			serverResult <- err
			return
		}
		sentence, err := readRouterOSSentence(reader)
		if err != nil {
			serverResult <- err
			return
		}
		if len(sentence) != 1 || sentence[0] != "/login" {
			serverResult <- fmt.Errorf("legacy login request = %v, want [/login]", sentence)
			return
		}
		if err := writeRouterOSSentence(serverConn, []string{"!done", "=ret=" + challenge}); err != nil {
			serverResult <- err
			return
		}
		sentence, err = readRouterOSSentence(reader)
		if err != nil {
			serverResult <- err
			return
		}
		hash := md5.New()
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(password))
		challengeBytes, _ := hex.DecodeString(challenge)
		_, _ = hash.Write(challengeBytes)
		wantResponse := "00" + hex.EncodeToString(hash.Sum(nil))
		want := []string{"/login", "=name=api-user", "=response=" + wantResponse}
		if fmt.Sprint(sentence) != fmt.Sprint(want) {
			serverResult <- fmt.Errorf("challenge response = %v, want %v", sentence, want)
			return
		}
		serverResult <- writeRouterOSSentence(serverConn, []string{"!done"})
	}()

	if err := client.login(); err != nil {
		t.Fatalf("login() error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}
