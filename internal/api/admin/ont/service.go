package ont

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"skripsi-be/internal/genieacs"
	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/services"
)

type Service interface {
	GetMapStatuses(ctx context.Context) ([]MapOntStatusSummary, error)
	GetCustomerOnt(ctx context.Context, customerID string) (*entities.OntStatus, error)
	GetCustomerHistory(ctx context.Context, customerID string, limit int) ([]entities.OntStatusHistory, error)
	RefreshCustomerOnt(ctx context.Context, customerID string) (*entities.OntStatus, error)
}

type service struct {
	repo   Repository
	poller *services.OntPollerService
}

func NewService(repo Repository, poller *services.OntPollerService) Service {
	return &service{
		repo:   repo,
		poller: poller,
	}
}

func (s *service) GetMapStatuses(ctx context.Context) ([]MapOntStatusSummary, error) {
	return s.repo.GetMapStatuses(ctx)
}

func (s *service) GetCustomerOnt(ctx context.Context, customerID string) (*entities.OntStatus, error) {
	status, err := s.repo.GetCustomerOntStatus(ctx, customerID)
	if err == nil && status != nil {
		return status, nil
	}

	// If not in DB yet, check if customer has ONT serial registered and try direct lookup
	customer, err := s.repo.GetCustomerByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	if customer.OntSerialNumber == nil || *customer.OntSerialNumber == "" {
		return nil, fmt.Errorf("customer has no registered ONT serial number")
	}

	sn := strings.TrimSpace(*customer.OntSerialNumber)

	// Attempt on-demand lookup via NBI
	if s.poller != nil && s.poller.Client() != nil {
		doc, err := s.poller.Client().GetDeviceBySerial(ctx, sn)
		if err == nil && doc != nil {
			metrics := genieacs.ExtractMetrics(*doc, s.poller.Config())
			if metrics != nil {
				newStatus := &entities.OntStatus{
					CustomerID:    customerID,
					SerialNumber:  sn,
					Manufacturer:  metrics.Manufacturer,
					ModelName:     metrics.ModelName,
					IsOnline:      metrics.IsOnline,
					RxPower:       metrics.RxPower,
					TxPower:       metrics.TxPower,
					OpticalStatus: string(metrics.OpticalStatus),
					SSID:          metrics.SSID,
					ClientCount:   metrics.ClientCount,
					LastInform:    metrics.LastInform,
					IsStale:       metrics.IsStale,
					UpdatedAt:     time.Now(),
				}
				_ = s.repo.SaveOntStatus(ctx, newStatus)
				return newStatus, nil
			}
		}
	}

	return nil, fmt.Errorf("ONT data not available yet for serial number %s", sn)
}

func (s *service) GetCustomerHistory(ctx context.Context, customerID string, limit int) ([]entities.OntStatusHistory, error) {
	return s.repo.GetCustomerOntHistory(ctx, customerID, limit)
}

func (s *service) RefreshCustomerOnt(ctx context.Context, customerID string) (*entities.OntStatus, error) {
	customer, err := s.repo.GetCustomerByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	if customer.OntSerialNumber == nil || *customer.OntSerialNumber == "" {
		return nil, fmt.Errorf("customer has no registered ONT serial number to refresh")
	}

	if s.poller == nil || s.poller.Client() == nil {
		return nil, fmt.Errorf("GenieACS client is not initialized")
	}

	sn := strings.TrimSpace(*customer.OntSerialNumber)
	client := s.poller.Client()

	// 1. Locate device document in GenieACS to find full _id
	doc, err := client.GetDeviceBySerial(ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("ONT %s not found in GenieACS: %w", sn, err)
	}

	deviceID, ok := (*doc)["_id"].(string)
	if !ok || deviceID == "" {
		return nil, fmt.Errorf("device document missing _id")
	}

	// 2. Trigger task refreshObject with Connection Request
	refreshCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	if err := client.RefreshDevice(refreshCtx, deviceID); err != nil {
		log.Printf("[ONT Refresh] Refresh request warning for device %s: %v", deviceID, err)
		// Don't fail immediately, wait and check if fresh data arrived
	}

	// 3. Wait a moment for TR-069 session handshake to complete
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
	}

	// 4. Fetch the refreshed device document
	updatedDoc, err := client.GetDeviceByID(ctx, deviceID)
	if err != nil {
		updatedDoc = doc // fallback to previous doc
	}

	// 5. Parse refreshed metrics
	metrics := genieacs.ExtractMetrics(*updatedDoc, s.poller.Config())
	now := time.Now()

	status := &entities.OntStatus{
		CustomerID:    customerID,
		SerialNumber:  sn,
		Manufacturer:  metrics.Manufacturer,
		ModelName:     metrics.ModelName,
		IsOnline:      metrics.IsOnline,
		RxPower:       metrics.RxPower,
		TxPower:       metrics.TxPower,
		OpticalStatus: string(metrics.OpticalStatus),
		SSID:          metrics.SSID,
		ClientCount:   metrics.ClientCount,
		LastInform:    metrics.LastInform,
		IsStale:       metrics.IsStale,
		UpdatedAt:     now,
	}

	_ = s.repo.SaveOntStatus(ctx, status)

	if metrics.RxPower != nil {
		history := &entities.OntStatusHistory{
			CustomerID:    customerID,
			SerialNumber:  sn,
			IsOnline:      metrics.IsOnline,
			RxPower:       metrics.RxPower,
			TxPower:       metrics.TxPower,
			OpticalStatus: string(metrics.OpticalStatus),
			RecordedAt:    now,
		}
		_ = s.repo.SaveOntHistory(ctx, history)
	}

	return status, nil
}

