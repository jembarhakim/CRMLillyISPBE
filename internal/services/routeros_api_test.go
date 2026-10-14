package services

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
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
