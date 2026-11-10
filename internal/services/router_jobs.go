package services

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

type RouterAction string

const (
	RouterActionSetUnpaidScheduler RouterAction = "set_unpaid_scheduler"
	RouterActionRunOpenScript      RouterAction = "run_open_script"
)

// EnqueueRouterJob creates or upserts a job by unique key (invoiceID:action)
func EnqueueRouterJob(db *gorm.DB, invoiceID string, action RouterAction, delay time.Duration) (*entities.RouterJob, error) {
	if invoiceID == "" || action == "" {
		return nil, errors.New("invalid enqueue arguments")
	}
	unique := invoiceID + ":" + string(action)
	job := entities.RouterJob{}
	// Use Find instead of First to avoid "record not found" error logs when job doesn't exist
	result := db.Where("unique_key = ?", unique).Find(&job)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected > 0 {
		// Job exists - reset to pending and schedule soon
		job.Status = entities.RouterJobStatusPending
		job.LastError = nil
		job.NextRunAt = time.Now().Add(delay)
		job.UpdatedAt = time.Now()
		if err := db.Save(&job).Error; err != nil {
			return nil, err
		}
		log.Printf("[routerjobs] reset job unique=%s invoice=%s action=%s retry_count=%d next_run_at=%s", unique, invoiceID, string(action), job.RetryCount, job.NextRunAt.Format(time.RFC3339))
		return &job, nil
	}
	// Create new job (record not found is expected here)
	job = entities.RouterJob{
		InvoiceID:  invoiceID,
		Action:     string(action),
		UniqueKey:  unique,
		Status:     entities.RouterJobStatusPending,
		RetryCount: 0,
		NextRunAt:  time.Now().Add(delay),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := db.Create(&job).Error; err != nil {
		return nil, err
	}
	log.Printf("[routerjobs] enqueued unique=%s invoice=%s action=%s next_run_at=%s", unique, invoiceID, string(action), job.NextRunAt.Format(time.RFC3339))
	return &job, nil
}

// StartRouterJobWorker runs a background loop to process jobs
func StartRouterJobWorker(db *gorm.DB) {
	log.Printf("[routerjobs] ✅ Router jobs worker started - monitoring for pending MikroTik operations")

	// Start with 2 second polling
	ticker := time.NewTicker(2 * time.Second)
	noJobsCount := 0
	lastStatusLog := time.Now()

	for range ticker.C {
		hasJobs := processDueJobs(db)

		if hasJobs {
			// Reset to fast polling when jobs are found
			noJobsCount = 0
			ticker.Reset(2 * time.Second)
		} else {
			noJobsCount++
			// Gradually increase polling interval when no jobs
			if noJobsCount > 5 { // After 10 seconds of no jobs
				ticker.Reset(10 * time.Second)
			} else if noJobsCount > 2 { // After 4 seconds of no jobs
				ticker.Reset(5 * time.Second)
			}

			// Log status every 5 minutes when idle
			if time.Since(lastStatusLog) > 5*time.Minute {
				log.Printf("[routerjobs] 💤 Worker idle - no pending jobs")
				lastStatusLog = time.Now()
			}
		}
	}
}

func processDueJobs(db *gorm.DB) bool {
	// Fetch one due job to avoid stampede; simple approach
	var job entities.RouterJob
	// Use Find instead of First to avoid "record not found" logs
	result := db.Where("status = ? AND next_run_at <= ?", entities.RouterJobStatusPending, time.Now()).Order("next_run_at ASC").Limit(1).Find(&job)
	if result.Error != nil {
		log.Printf("[routerjobs] query error: %v", result.Error)
		return false
	}
	if result.RowsAffected == 0 {
		// No jobs found - this is normal, don't log
		return false
	}

	// Mark as pending (already), ensure UpdatedAt bumps
	job.UpdatedAt = time.Now()
	_ = db.Save(&job)

	// Load invoice and relations needed
	var invoice entities.Invoice
	if err := db.Preload("Customer").Preload("Customer.Area").First(&invoice, "id = ?", job.InvoiceID).Error; err != nil {
		setJobError(db, &job, fmt.Errorf("invoice not found: %w", err))
		return true // Job was processed (even if failed)
	}

	// Acquire RouterOS API connection
	router := GetSharedRouterOSAPI()
	if router == nil || !router.IsConnected() {
		setJobRetry(db, &job, fmt.Errorf("RouterOS API not connected"))
		return true // Job was processed (even if failed)
	}

	var err error
	switch RouterAction(job.Action) {
	case RouterActionSetUnpaidScheduler:
		err = executeSetUnpaidScheduler(router, &invoice)
	case RouterActionRunOpenScript:
		err = executeRunOpenScript(router, &invoice)
	default:
		err = fmt.Errorf("unknown action: %s", job.Action)
	}

	if err != nil {
		if isRetryableError(err) {
			setJobRetry(db, &job, err)
			log.Printf("[routerjobs] retry scheduled unique=%s invoice=%s action=%s retry_count=%d err=%v", job.UniqueKey, job.InvoiceID, job.Action, job.RetryCount, err)
		} else {
			setJobError(db, &job, err)
			log.Printf("[routerjobs] error unique=%s invoice=%s action=%s err=%v", job.UniqueKey, job.InvoiceID, job.Action, err)
		}
		return true // Job was processed (even if failed)
	}

	// Success
	job.Status = entities.RouterJobStatusSuccess
	job.LastError = nil
	job.UpdatedAt = time.Now()
	_ = db.Save(&job)
	log.Printf("[routerjobs] success unique=%s invoice=%s action=%s", job.UniqueKey, job.InvoiceID, job.Action)
	return true
}

func isRetryableError(err error) bool {
	msg := strings.ToLower(err.Error())
	// Treat transient connectivity or not-found as retryable for a few attempts
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "temporar") || strings.Contains(msg, "connect")
}

func setJobRetry(db *gorm.DB, job *entities.RouterJob, err error) {
	msg := err.Error()
	job.RetryCount += 1
	// exponential backoff with cap 30m
	backoff := time.Duration(1<<min(job.RetryCount-1, 5)) * time.Minute // 1m,2m,4m,8m,16m,32m
	if backoff > 30*time.Minute {
		backoff = 30 * time.Minute
	}
	job.NextRunAt = time.Now().Add(backoff)
	job.Status = entities.RouterJobStatusPending
	job.LastError = &msg
	job.UpdatedAt = time.Now()
	_ = db.Save(job)
}

func setJobError(db *gorm.DB, job *entities.RouterJob, err error) {
	msg := err.Error()
	job.Status = entities.RouterJobStatusError
	job.LastError = &msg
	job.UpdatedAt = time.Now()
	_ = db.Save(job)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helpers to execute actions
func executeSetUnpaidScheduler(router *RouterOSAPI, invoice *entities.Invoice) error {
	if invoice.DueDate == nil {
		return fmt.Errorf("invoice due date is nil")
	}
	name := buildSchedulerName(invoice)
	records, err := router.Command("/system/scheduler/print")
	if err != nil {
		return fmt.Errorf("RouterOS scheduler check failed: %w", err)
	}
	schedulerID := findRouterOSRecordID(records, name)
	if schedulerID == "" {
		return fmt.Errorf("scheduler not found: %s", name)
	}
	startDate := strings.ToLower(invoice.DueDate.Format("Jan/02/2006"))
	if _, err := router.Command("/system/scheduler/set", "=.id="+schedulerID, "=start-date="+startDate); err != nil {
		return fmt.Errorf("failed to set scheduler date: %w", err)
	}
	return nil
}

func executeRunOpenScript(router *RouterOSAPI, invoice *entities.Invoice) error {
	name := buildSchedulerName(invoice)
	scriptName := "open_" + name
	records, err := router.Command("/system/script/print")
	if err != nil {
		return fmt.Errorf("RouterOS script check failed: %w", err)
	}
	scriptID := findRouterOSRecordID(records, scriptName)
	if scriptID == "" {
		return fmt.Errorf("script not found: %s", scriptName)
	}
	if _, err := router.Command("/system/script/run", "=.id="+scriptID); err != nil {
		return fmt.Errorf("failed to run script %s: %w", scriptName, err)
	}
	return nil
}

func findRouterOSRecordID(records []map[string]string, name string) string {
	for _, record := range records {
		if record["name"] == name {
			return record[".id"]
		}
	}
	return ""
}

func buildSchedulerName(invoice *entities.Invoice) string {
	code := ""
	custName := ""
	if invoice.Customer.Name != "" {
		custName = invoice.Customer.Name
	}
	if invoice.Customer.Area.CodeName != "" {
		code = invoice.Customer.Area.CodeName
	}
	if code == "" {
		code = "N/A"
	}
	if custName == "" {
		custName = "Unknown"
	}
	return fmt.Sprintf("%s - %s", code, custName)
}
