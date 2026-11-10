package services

import (
	"bufio"
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"
)

type routerOSAPITestStep struct {
	request  []string
	response [][]string
}

func newRouterOSAPITestClient(t *testing.T, steps []routerOSAPITestStep) (*RouterOSAPI, <-chan error) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	client := &RouterOSAPI{conn: clientConn, reader: bufio.NewReader(clientConn)}
	serverResult := make(chan error, 1)

	go func() {
		defer serverConn.Close()
		reader := bufio.NewReader(serverConn)
		for i, step := range steps {
			request, err := readRouterOSSentence(reader)
			if err != nil {
				serverResult <- fmt.Errorf("read request %d: %w", i, err)
				return
			}
			if !reflect.DeepEqual(request, step.request) {
				serverResult <- fmt.Errorf("request %d = %v, want %v", i, request, step.request)
				return
			}
			for _, sentence := range step.response {
				if err := writeRouterOSSentence(serverConn, sentence); err != nil {
					serverResult <- fmt.Errorf("write response %d: %w", i, err)
					return
				}
			}
		}
		serverResult <- nil
	}()

	t.Cleanup(func() {
		_ = client.Close()
	})
	return client, serverResult
}

func TestSetHotspotIPBindingTypeUpdatesMatchingBindings(t *testing.T) {
	mac := "AA:BB:CC:DD:EE:FF"
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{
			request: []string{"/ip/hotspot/ip-binding/print"},
			response: [][]string{
				{"!re", "=.id=*1", "=mac-address=" + mac, "=type=regular"},
				{"!re", "=.id=*2", "=mac-address=11:22:33:44:55:66", "=type=regular"},
				{"!done"},
			},
		},
		{
			request:  []string{"/ip/hotspot/ip-binding/set", "=.id=*1", "=type=bypassed"},
			response: [][]string{{"!done"}},
		},
	})

	if err := client.SetHotspotIPBindingType(strings.ToLower(mac), "bypassed"); err != nil {
		t.Fatalf("SetHotspotIPBindingType() error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestSetHotspotIPBindingTypeReturnsNotFound(t *testing.T) {
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{
			request: []string{"/ip/hotspot/ip-binding/print"},
			response: [][]string{
				{"!re", "=.id=*1", "=mac-address=11:22:33:44:55:66"},
				{"!done"},
			},
		},
	})

	err := client.SetHotspotIPBindingType("AA:BB:CC:DD:EE:FF", "regular")
	if err == nil {
		t.Fatal("SetHotspotIPBindingType() error = nil, want not-found error")
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestHotspotIPBindingValidation(t *testing.T) {
	client := &RouterOSAPI{}
	if err := client.SetHotspotIPBindingType("AA:BB:CC:DD:EE:FF", "disabled"); err == nil {
		t.Fatal("SetHotspotIPBindingType() accepted unsupported type")
	}
	if err := client.SetHotspotDeviceBindingType("192.0.2.10", "regular"); err == nil {
		t.Fatal("SetHotspotDeviceBindingType() accepted unsupported type")
	}
	if _, err := client.GetHotspotIPBindingByMAC("not-a-mac"); err == nil {
		t.Fatal("GetHotspotIPBindingByMAC() accepted invalid MAC")
	}
}

func TestSetHotspotDeviceBindingTypeUsesExactDeviceBinding(t *testing.T) {
	const address = "192.0.2.10"
	const mac = "AA:BB:CC:DD:EE:FF"
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{request: []string{"/ip/dhcp-server/lease/print"}, response: [][]string{
			{"!re", "=address=" + address, "=mac-address=" + mac},
			{"!done"},
		}},
		{request: []string{"/ip/arp/print"}, response: [][]string{{"!done"}}},
		{request: []string{"/ip/hotspot/ip-binding/print"}, response: [][]string{
			{"!re", "=.id=*7", "=address=" + address, "=mac-address=" + mac, "=type=regular", "=comment=CRM note"},
			{"!re", "=.id=*8", "=address=192.0.2.11", "=mac-address=11:22:33:44:55:66", "=type=regular"},
			{"!done"},
		}},
		{request: []string{"/ip/hotspot/ip-binding/set", "=.id=*7", "=type=blocked", "=comment=CRM note\n[CRM-DEVICE-ISOLATION original-type=regular]"}, response: [][]string{{"!done"}}},
	})

	if err := client.SetHotspotDeviceBindingType(address, "blocked"); err != nil {
		if serverErr := <-serverResult; serverErr != nil {
			t.Fatalf("SetHotspotDeviceBindingType() error = %v; fake RouterOS server: %v", err, serverErr)
		}
		t.Fatalf("SetHotspotDeviceBindingType() error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestSetHotspotDeviceBindingTypeRestoresOriginalTypeAndComment(t *testing.T) {
	const address = "192.0.2.10"
	const mac = "AA:BB:CC:DD:EE:FF"
	const comment = "CRM note\n[CRM-DEVICE-ISOLATION original-type=bypassed]"
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{request: []string{"/ip/dhcp-server/lease/print"}, response: [][]string{
			{"!re", "=address=" + address, "=mac-address=" + mac},
			{"!done"},
		}},
		{request: []string{"/ip/arp/print"}, response: [][]string{{"!done"}}},
		{request: []string{"/ip/hotspot/ip-binding/print"}, response: [][]string{
			{"!re", "=.id=*7", "=address=" + address, "=mac-address=" + mac, "=type=blocked", "=comment=" + comment},
			{"!done"},
		}},
		{request: []string{"/ip/hotspot/ip-binding/set", "=.id=*7", "=type=bypassed", "=comment=CRM note"}, response: [][]string{{"!done"}}},
	})

	if err := client.SetHotspotDeviceBindingType(address, "restore"); err != nil {
		t.Fatalf("SetHotspotDeviceBindingType(restore) error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestResolveHotspotDeviceBindingsRejectsConflictingMACs(t *testing.T) {
	const address = "192.0.2.10"
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{request: []string{"/ip/dhcp-server/lease/print"}, response: [][]string{
			{"!re", "=address=" + address, "=mac-address=AA:BB:CC:DD:EE:FF"},
			{"!done"},
		}},
		{request: []string{"/ip/arp/print"}, response: [][]string{
			{"!re", "=address=" + address, "=mac-address=11:22:33:44:55:66"},
			{"!done"},
		}},
		{request: []string{"/ip/hotspot/ip-binding/print"}, response: [][]string{{"!done"}}},
	})

	resolutions, err := client.ResolveHotspotDeviceBindings([]string{address})
	if err != nil {
		t.Fatalf("ResolveHotspotDeviceBindings() error = %v", err)
	}
	if len(resolutions) != 1 || resolutions[0].CanManage || resolutions[0].MAC != "" {
		t.Fatalf("ResolveHotspotDeviceBindings() = %#v, want one unresolved device", resolutions)
	}
	if resolutions[0].Reason != "IP address resolves to multiple MAC addresses" {
		t.Errorf("resolution reason = %q", resolutions[0].Reason)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestListHotspotIPBindingsMapsRouterOSFields(t *testing.T) {
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{
			request: []string{"/ip/hotspot/ip-binding/print"},
			response: [][]string{
				{"!re", "=.id=*1", "=mac-address=AA:BB:CC:DD:EE:FF", "=address=192.0.2.10", "=type=bypassed", "=server=all", "=comment=CRM", "=disabled=yes"},
				{"!done"},
			},
		},
	})

	bindings, err := client.ListHotspotIPBindings()
	if err != nil {
		t.Fatalf("ListHotspotIPBindings() error = %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("ListHotspotIPBindings() returned %d bindings, want 1", len(bindings))
	}
	want := HotspotIPBinding{
		ID:       "*1",
		MAC:      "AA:BB:CC:DD:EE:FF",
		Address:  "192.0.2.10",
		Type:     "bypassed",
		Server:   "all",
		Comment:  "CRM",
		Disabled: true,
	}
	if bindings[0] != want {
		t.Errorf("ListHotspotIPBindings() = %#v, want %#v", bindings[0], want)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestListLocalHotspotDataMapsRouterOSFields(t *testing.T) {
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{
			request: []string{"/ip/hotspot/user/print"},
			response: [][]string{
				{"!re", "=.id=*1", "=name=customer-1", "=profile=10M", "=address=192.0.2.10", "=limit-uptime=1d", "=comment=CRM", "=disabled=false"},
				{"!done"},
			},
		},
		{
			request: []string{"/ip/hotspot/user/profile/print"},
			response: [][]string{
				{"!re", "=name=10M", "=rate-limit=10M/10M", "=shared-users=1"},
				{"!done"},
			},
		},
		{
			request: []string{"/ip/hotspot/active/print"},
			response: [][]string{
				{"!re", "=.id=*2", "=user=customer-1", "=address=192.0.2.10", "=mac-address=AA:BB:CC:DD:EE:FF", "=uptime=2h", "=bytes-in=100", "=bytes-out=200"},
				{"!done"},
			},
		},
	})

	users, err := client.ListHotspotUsers()
	if err != nil {
		t.Fatalf("ListHotspotUsers() error = %v", err)
	}
	if len(users) != 1 || users[0].Name != "customer-1" || users[0].Profile != "10M" || users[0].Disabled {
		t.Errorf("ListHotspotUsers() = %#v", users)
	}

	profiles, err := client.ListHotspotUserProfiles()
	if err != nil {
		t.Fatalf("ListHotspotUserProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "10M" || profiles[0].RateLimit != "10M/10M" {
		t.Errorf("ListHotspotUserProfiles() = %#v", profiles)
	}

	sessions, err := client.ListHotspotActiveSessions()
	if err != nil {
		t.Fatalf("ListHotspotActiveSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].User != "customer-1" || sessions[0].MAC != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("ListHotspotActiveSessions() = %#v", sessions)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestCreateHotspotUser(t *testing.T) {
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{
			request:  []string{"/ip/hotspot/user/print"},
			response: [][]string{{"!done"}},
		},
		{
			request:  []string{"/ip/hotspot/user/add", "=name=test-user", "=password=test-pass", "=profile=10M", "=comment=CRM test"},
			response: [][]string{{"!done"}},
		},
	})

	if err := client.CreateHotspotUser("test-user", "test-pass", "10M", "CRM test"); err != nil {
		t.Fatalf("CreateHotspotUser() error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}

func TestKickHotspotUserRemovesMatchingActiveSessions(t *testing.T) {
	client, serverResult := newRouterOSAPITestClient(t, []routerOSAPITestStep{
		{
			request: []string{"/ip/hotspot/user/print"},
			response: [][]string{
				{"!re", "=.id=*1", "=name=test-user"},
				{"!done"},
			},
		},
		{
			request: []string{"/ip/hotspot/active/print"},
			response: [][]string{
				{"!re", "=.id=*2", "=user=test-user"},
				{"!re", "=.id=*3", "=user=other-user"},
				{"!done"},
			},
		},
		{
			request:  []string{"/ip/hotspot/active/remove", "=.id=*2"},
			response: [][]string{{"!done"}},
		},
	})

	kicked, err := client.KickHotspotUser("test-user")
	if err != nil {
		t.Fatalf("KickHotspotUser() error = %v", err)
	}
	if kicked != 1 {
		t.Fatalf("KickHotspotUser() = %d, want 1", kicked)
	}
	if err := <-serverResult; err != nil {
		t.Fatalf("fake RouterOS server: %v", err)
	}
}
