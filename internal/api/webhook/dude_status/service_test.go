package dude_status

import (
	"errors"
	"testing"
	"time"

	"skripsi-be/internal/models/entities"
)

type fakeStatusRepository struct {
	status      DeviceStatus
	devices     []entities.DudeDeviceStatus
	err         error
	calls       int
	updateCalls int
}

func (r *fakeStatusRepository) Upsert(status DeviceStatus) error {
	r.status = status
	r.calls++
	return r.err
}

func (r *fakeStatusRepository) List() ([]entities.DudeDeviceStatus, error) {
	return r.devices, r.err
}

func (r *fakeStatusRepository) UpdateDetails(_ string, _ UpdateDeviceDetailsRequest) (entities.DudeDeviceStatus, error) {
	r.updateCalls++
	return entities.DudeDeviceStatus{}, r.err
}

type fakeStatusBroadcaster struct {
	status DeviceStatus
	err    error
	calls  int
}

func (b *fakeStatusBroadcaster) Broadcast(status DeviceStatus) error {
	b.status = status
	b.calls++
	return b.err
}

func TestReceiveStatusPersistsAndBroadcastsTrimmedPayload(t *testing.T) {
	repository := &fakeStatusRepository{}
	broadcaster := &fakeStatusBroadcaster{}
	service := NewService(repository, broadcaster)
	input := DeviceStatus{
		DeviceName: " Router Office ",
		IPAddress:  " 192.168.1.1 ",
		Status:     " up ",
	}

	if err := service.ReceiveStatus(input); err != nil {
		t.Fatalf("ReceiveStatus() error = %v", err)
	}

	want := DeviceStatus{
		DeviceName: "Router Office",
		IPAddress:  "192.168.1.1",
		Status:     "up",
	}
	if repository.calls != 1 || repository.status != want {
		t.Fatalf("repository got (%d calls, %#v), want (1 call, %#v)", repository.calls, repository.status, want)
	}
	if broadcaster.calls != 1 || broadcaster.status != want {
		t.Fatalf("broadcaster got (%d calls, %#v), want (1 call, %#v)", broadcaster.calls, broadcaster.status, want)
	}
}

func TestReceiveStatusRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name   string
		status DeviceStatus
	}{
		{name: "device name"},
		{name: "IP address", status: DeviceStatus{DeviceName: "Router", Status: "up"}},
		{name: "status", status: DeviceStatus{DeviceName: "Router", IPAddress: "192.168.1.1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeStatusRepository{}
			broadcaster := &fakeStatusBroadcaster{}
			service := NewService(repository, broadcaster)

			if err := service.ReceiveStatus(test.status); !errors.Is(err, ErrInvalidStatus) {
				t.Fatalf("ReceiveStatus() error = %v, want %v", err, ErrInvalidStatus)
			}
			if repository.calls != 0 || broadcaster.calls != 0 {
				t.Fatalf("invalid status caused side effects: repository=%d broadcaster=%d", repository.calls, broadcaster.calls)
			}
		})
	}
}

func TestReceiveStatusDoesNotBroadcastAfterPersistenceFailure(t *testing.T) {
	repository := &fakeStatusRepository{err: errors.New("database unavailable")}
	broadcaster := &fakeStatusBroadcaster{}
	service := NewService(repository, broadcaster)

	err := service.ReceiveStatus(DeviceStatus{
		DeviceName: "Router",
		IPAddress:  "192.168.1.1",
		Status:     "up",
	})
	if err == nil {
		t.Fatal("ReceiveStatus() error = nil, want persistence error")
	}
	if repository.calls != 1 || broadcaster.calls != 0 {
		t.Fatalf("got repository=%d broadcaster=%d calls, want repository=1 broadcaster=0", repository.calls, broadcaster.calls)
	}
}

func TestUpdateDeviceDetailsRejectsIncompleteCoordinates(t *testing.T) {
	repository := &fakeStatusRepository{}
	service := NewService(repository, &fakeStatusBroadcaster{})
	latitude := -6.2

	_, err := service.UpdateDeviceDetails("device-id", UpdateDeviceDetailsRequest{
		Latitude: &latitude,
	})
	if !errors.Is(err, ErrInvalidDeviceDetails) {
		t.Fatalf("UpdateDeviceDetails() error = %v, want %v", err, ErrInvalidDeviceDetails)
	}
	if repository.updateCalls != 0 {
		t.Fatalf("invalid coordinates caused %d repository updates, want 0", repository.updateCalls)
	}
}

func TestStartPingAllRejectsEmptyDeviceList(t *testing.T) {
	service := NewService(&fakeStatusRepository{}, &fakeStatusBroadcaster{})

	if _, err := service.StartPingAll(); !errors.Is(err, ErrNoPingTargets) {
		t.Fatalf("StartPingAll() error = %v, want %v", err, ErrNoPingTargets)
	}
}

func TestStartPingAllChecksAndBroadcastsDeviceStatus(t *testing.T) {
	repository := &fakeStatusRepository{
		devices: []entities.DudeDeviceStatus{{
			Name:      "Router Office",
			IPAddress: "192.0.2.1",
		}},
	}
	broadcaster := &fakeStatusBroadcaster{}
	service := NewService(repository, broadcaster)
	service.pingHost = func(ip string) (bool, error) {
		if ip != "192.0.2.1" {
			t.Errorf("ping IP = %q, want 192.0.2.1", ip)
		}
		return true, nil
	}

	job, err := service.StartPingAll()
	if err != nil {
		t.Fatalf("StartPingAll() error = %v", err)
	}
	if !job.Running || job.Total != 1 {
		t.Fatalf("initial ping job = %+v, want running with one target", job)
	}

	deadline := time.Now().Add(time.Second)
	for service.PingAllStatus().Running && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	job = service.PingAllStatus()
	if job.Running || job.Completed != 1 || job.Up != 1 || job.Down != 0 || job.Failed != 0 {
		t.Fatalf("completed ping job = %+v, want one successful UP result", job)
	}
	if repository.calls != 1 || repository.status.Status != "up" {
		t.Fatalf("persisted ping status = (%d calls, %+v), want one UP update", repository.calls, repository.status)
	}
	if broadcaster.calls != 1 || broadcaster.status.Status != "up" {
		t.Fatalf("broadcast ping status = (%d calls, %+v), want one UP update", broadcaster.calls, broadcaster.status)
	}
}
