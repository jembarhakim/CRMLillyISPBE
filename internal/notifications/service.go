package notifications

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"skripsi-be/internal/api/webhook/wa"
	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatusQueued    = "queued"
	StatusSending   = "sending"
	StatusSent      = "sent"
	StatusDelivered = "delivered"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

var variablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type TemplateInput struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Body     string `json:"body"`
	Enabled  *bool  `json:"enabled"`
}

type NotificationInput struct {
	To          string         `json:"to"`
	TemplateKey string         `json:"template_key"`
	Variables   map[string]any `json:"variables"`
}

type PreferenceInput struct {
	CustomerID              string `json:"customer_id"`
	InvoiceRemindersEnabled bool   `json:"invoice_reminders_enabled"`
}

type PreferenceView struct {
	CustomerID              string `json:"customer_id"`
	CustomerName            string `json:"customer_name"`
	Phone                   string `json:"phone"`
	InvoiceRemindersEnabled bool   `json:"invoice_reminders_enabled"`
}

func DefaultTemplates() []entities.NotificationTemplate {
	return []entities.NotificationTemplate{
		{Key: "ticket_created", Name: "Tiket gangguan dibuat", Category: "ticket", Body: "Halo {{nama_pelanggan}}, tiket {{no_tiket}} sudah kami catat: {{ringkasan}}. Tim kami akan segera menindaklanjuti."},
		{Key: "ticket_in_progress", Name: "Tiket sedang ditangani", Category: "ticket", Body: "Halo {{nama_pelanggan}}, tiket {{no_tiket}} sedang ditangani oleh teknisi. Kami akan mengabari Anda kembali."},
		{Key: "ticket_resolved", Name: "Tiket selesai", Category: "ticket", Body: "Halo {{nama_pelanggan}}, pekerjaan untuk tiket {{no_tiket}} telah selesai. Terima kasih telah menggunakan layanan kami. Evaluasi layanan: {{link_evaluasi}}"},
		{Key: "work_order_assigned", Name: "Penugasan teknisi", Category: "work_order", Body: "Tugas {{no_tiket}} untuk {{nama_pelanggan}} ditugaskan kepada Anda. Alamat: {{alamat}}. Telepon: {{telepon}}. Lokasi: {{link_maps}}"},
		{Key: "work_order_scheduled", Name: "Konfirmasi jadwal kunjungan", Category: "work_order", Body: "Halo {{nama_pelanggan}}, kunjungan teknisi {{nama_teknisi}} dijadwalkan pada {{jadwal_kunjungan}}. Alamat: {{alamat}}. Lokasi: {{link_maps}}"},
		{Key: "work_order_completed", Name: "Pekerjaan teknisi selesai", Category: "work_order", Body: "Halo {{nama_pelanggan}}, pekerjaan kunjungan {{no_tiket}} telah selesai. Terima kasih."},
		{Key: "invoice_reminder_3_days", Name: "Pengingat tagihan H-3", Category: "invoice", Body: "Halo {{nama_pelanggan}}, tagihan {{no_invoice}} untuk {{paket}} sebesar {{total_tagihan}} akan jatuh tempo pada {{tanggal_jatuh_tempo}}. Bayar di {{link_pembayaran}}."},
		{Key: "invoice_reminder_due_today", Name: "Pengingat tagihan jatuh tempo", Category: "invoice", Body: "Halo {{nama_pelanggan}}, tagihan {{no_invoice}} sebesar {{total_tagihan}} jatuh tempo hari ini. Rincian dan pembayaran: {{link_pembayaran}}."},
		{Key: "invoice_reminder_overdue", Name: "Peringatan tagihan terlambat", Category: "invoice", Body: "Halo {{nama_pelanggan}}, tagihan {{no_invoice}} sebesar {{total_tagihan}} belum tercatat lunas dan telah melewati jatuh tempo. Silakan bayar di {{link_pembayaran}}."},
	}
}

func (s *Service) SeedDefaultTemplates() error {
	for _, template := range DefaultTemplates() {
		if err := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&template).Error; err != nil {
			return fmt.Errorf("seed notification template %q: %w", template.Key, err)
		}
	}
	return nil
}

func Render(body string, values map[string]any) string {
	return variablePattern.ReplaceAllStringFunc(body, func(match string) string {
		parts := variablePattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		value, ok := values[parts[1]]
		if !ok || value == nil {
			return match
		}
		return fmt.Sprint(value)
	})
}

func (s *Service) Enqueue(to, templateKey string, variables map[string]any, deduplicationKey, invoiceID string) error {
	if strings.TrimSpace(templateKey) == "" {
		return errors.New("template_key is required")
	}

	var template entities.NotificationTemplate
	if err := s.db.First(&template, "key = ?", templateKey).Error; err != nil {
		return fmt.Errorf("notification template %q: %w", templateKey, err)
	}
	if !template.Enabled {
		return fmt.Errorf("notification template %q is disabled", templateKey)
	}
	message := Render(template.Body, variables)
	if strings.TrimSpace(message) == "" {
		return errors.New("rendered notification message is empty")
	}
	return s.enqueueMessage(to, templateKey, message, deduplicationKey, invoiceID, "")
}

func (s *Service) EnqueueRaw(to, message, templateKey, broadcastID string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("message is required")
	}
	return s.enqueueMessage(to, templateKey, message, "", "", broadcastID)
}

func (s *Service) enqueueMessage(to, templateKey, message, deduplicationKey, invoiceID, broadcastID string) error {
	to = normalizePhoneNumber(to)
	if !validPhoneNumber(to) {
		return fmt.Errorf("invalid WhatsApp phone number")
	}
	notification := entities.Notification{
		Recipient:     to,
		TemplateKey:   templateKey,
		Message:       message,
		Status:        StatusQueued,
		Attempts:      0,
		MaxAttempts:   configuredInt("WHATSAPP_MAX_RETRIES", 5),
		NextAttemptAt: time.Now(),
	}
	if deduplicationKey != "" {
		notification.DeduplicationKey = &deduplicationKey
	}
	if invoiceID != "" {
		notification.InvoiceID = &invoiceID
	}
	if broadcastID != "" {
		notification.BroadcastID = &broadcastID
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "deduplication_key"}}, DoNothing: true}).Create(&notification)
		if result.Error != nil {
			return fmt.Errorf("enqueue WhatsApp notification: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil
		}
		return tx.Create(&entities.NotificationLog{
			NotificationID: notification.ID,
			Status:         StatusQueued,
			Details:        "notification added to queue",
		}).Error
	})
}

func (s *Service) EnqueueCustomer(customerID, templateKey string, variables map[string]any, deduplicationKey, invoiceID string) error {
	var customer entities.Customer
	if err := s.db.First(&customer, "id = ?", customerID).Error; err != nil {
		return fmt.Errorf("find notification customer: %w", err)
	}
	if strings.TrimSpace(customer.Phone) == "" {
		return fmt.Errorf("customer %q has no phone number", customerID)
	}
	return s.Enqueue(customer.Phone, templateKey, variables, deduplicationKey, invoiceID)
}

func (s *Service) ListTemplates() ([]entities.NotificationTemplate, error) {
	var templates []entities.NotificationTemplate
	err := s.db.Order("category, name").Find(&templates).Error
	return templates, err
}

func (s *Service) SaveTemplate(input TemplateInput) (entities.NotificationTemplate, error) {
	input.Key = strings.TrimSpace(input.Key)
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	if input.Key == "" || input.Name == "" || input.Category == "" || strings.TrimSpace(input.Body) == "" {
		return entities.NotificationTemplate{}, errors.New("key, name, category, and body are required")
	}
	template := entities.NotificationTemplate{
		Key: input.Key, Name: input.Name, Category: input.Category, Body: input.Body, Enabled: true,
	}
	if input.Enabled != nil {
		template.Enabled = *input.Enabled
	}
	err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "category", "body", "enabled", "updated_at"}),
	}).Create(&template).Error
	if err != nil {
		return entities.NotificationTemplate{}, err
	}
	err = s.db.First(&template, "key = ?", input.Key).Error
	return template, err
}

func (s *Service) DeleteTemplate(key string) error {
	for _, template := range DefaultTemplates() {
		if template.Key == key {
			return errors.New("system templates cannot be deleted; disable or edit the template instead")
		}
	}
	result := s.db.Delete(&entities.NotificationTemplate{}, "key = ?", key)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Service) ListNotifications(status string, limit int) ([]entities.Notification, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	query := s.db.Order("created_at DESC").Limit(limit)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var notifications []entities.Notification
	err := query.Find(&notifications).Error
	return notifications, err
}

func (s *Service) ListLogs(notificationID string, limit int) ([]entities.NotificationLog, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var logs []entities.NotificationLog
	query := s.db.Order("created_at DESC").Limit(limit)
	if notificationID != "" {
		query = query.Where("notification_id = ?", notificationID)
	}
	err := query.Find(&logs).Error
	return logs, err
}

func (s *Service) ListPreferences(search string) ([]PreferenceView, error) {
	var preferences []PreferenceView
	query := s.db.Table("customer AS c").
		Select("c.id AS customer_id, c.name AS customer_name, c.phone, COALESCE(p.invoice_reminders_enabled, 1) AS invoice_reminders_enabled").
		Joins("LEFT JOIN notification_preferences AS p ON p.customer_id = c.id").
		Where("c.phone <> '' AND c.deleted_at IS NULL").
		Order("c.name")
	if search != "" {
		query = query.Where("c.name LIKE ? OR c.phone LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	err := query.Limit(500).Scan(&preferences).Error
	return preferences, err
}

func (s *Service) SetPreference(input PreferenceInput) (entities.NotificationPreference, error) {
	if strings.TrimSpace(input.CustomerID) == "" {
		return entities.NotificationPreference{}, errors.New("customer_id is required")
	}
	preference := entities.NotificationPreference{
		CustomerID: input.CustomerID, InvoiceRemindersEnabled: input.InvoiceRemindersEnabled,
	}
	var customerCount int64
	if err := s.db.Model(&entities.Customer{}).Where("id = ?", input.CustomerID).Count(&customerCount).Error; err != nil {
		return entities.NotificationPreference{}, err
	}
	if customerCount == 0 {
		return entities.NotificationPreference{}, gorm.ErrRecordNotFound
	}
	err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "customer_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"invoice_reminders_enabled", "updated_at"}),
	}).Create(&preference).Error
	return preference, err
}

func (s *Service) EnqueueInvoiceReminders(now time.Time) (int, error) {
	today := dateOnly(now)
	var invoices []entities.Invoice
	err := s.db.Preload("Customer").Preload("InvoiceItems").
		Where("status IN ? AND due_date BETWEEN ? AND ?", []entities.InvoiceStatus{entities.InvoiceStatusUnpaid, entities.InvoiceStatusPending}, today.AddDate(0, 0, -1), today.AddDate(0, 0, 3)).
		Find(&invoices).Error
	if err != nil {
		return 0, err
	}

	enqueued := 0
	var enqueueErr error
	for _, invoice := range invoices {
		if invoice.Customer.Phone == "" {
			log.Printf("[notifications] skipping invoice %s: customer %s has no phone number", invoice.ID, invoice.CustomerID)
			continue
		}
		if invoice.DueDate == nil {
			log.Printf("[notifications] skipping invoice %s: due date is missing", invoice.ID)
			continue
		}
		var preference entities.NotificationPreference
		err := s.db.First(&preference, "customer_id = ?", invoice.CustomerID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return enqueued, err
		}
		if err == nil && !preference.InvoiceRemindersEnabled {
			continue
		}

		daysUntilDue := int(dateOnly(*invoice.DueDate).Sub(today).Hours() / 24)
		templateKey, stage := "", ""
		switch daysUntilDue {
		case 3:
			templateKey, stage = "invoice_reminder_3_days", "h-3"
		case 0:
			templateKey, stage = "invoice_reminder_due_today", "h0"
		case -1:
			templateKey, stage = "invoice_reminder_overdue", "h+1"
		default:
			continue
		}
		items := make([]string, 0, len(invoice.InvoiceItems))
		for _, item := range invoice.InvoiceItems {
			if item.Name != "" {
				items = append(items, item.Name)
			}
		}
		paymentLink := invoicePaymentLink(invoice)
		variables := map[string]any{
			"nama_pelanggan":      invoice.Customer.Name,
			"no_invoice":          invoice.ID,
			"paket":               strings.Join(items, ", "),
			"total_tagihan":       formatIDR(invoice.Amount),
			"tanggal_jatuh_tempo": invoice.DueDate.Format("02-01-2006"),
			"link_pembayaran":     paymentLink,
		}
		deduplicationKey := fmt.Sprintf("invoice:%s:%s:%s", invoice.ID, stage, invoice.DueDate.Format("2006-01-02"))
		var existing entities.Notification
		err = s.db.Select("id").First(&existing, "deduplication_key = ?", deduplicationKey).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return enqueued, err
		}
		if err := s.Enqueue(invoice.Customer.Phone, templateKey, variables, deduplicationKey, invoice.ID); err != nil {
			if enqueueErr == nil {
				enqueueErr = fmt.Errorf("queue reminder for invoice %s: %w", invoice.ID, err)
			}
			log.Printf("[notifications] unable to queue reminder for invoice %s: %v", invoice.ID, err)
			continue
		}
		enqueued++
	}
	return enqueued, enqueueErr
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func invoicePaymentLink(invoice entities.Invoice) string {
	paymentLink := strings.TrimSpace(invoice.Link)
	if paymentLink == "" || strings.HasPrefix(paymentLink, "/invoice/") {
		paymentLink = fmt.Sprintf("/invoice/%s", invoice.ID)
	}
	if baseURL := strings.TrimRight(os.Getenv("APP_BASE_URL"), "/"); baseURL != "" && strings.HasPrefix(paymentLink, "/") {
		paymentLink = baseURL + paymentLink
	}
	return paymentLink
}

func formatIDR(amount int64) string {
	value := strconv.FormatInt(amount, 10)
	for i := len(value) - 3; i > 0; i -= 3 {
		value = value[:i] + "." + value[i:]
	}
	return "Rp" + value
}

func configuredInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func normalizePhoneNumber(phone string) string {
	return wa.NormalizePhoneNumber(phone)
}

func validPhoneNumber(phone string) bool {
	return wa.IsValidPhoneNumber(phone)
}

func (s *Service) ApplyDeliveryStatus(messageID, status, details string) error {
	switch strings.ToLower(status) {
	case "sent", "delivered", "read", "failed":
	default:
		return fmt.Errorf("unsupported delivery status %q", status)
	}
	var notification entities.Notification
	if err := s.db.First(&notification, "gateway_message_id = ?", messageID).Error; err != nil {
		return err
	}
	nextStatus := strings.ToLower(status)
	if nextStatus == "read" {
		nextStatus = StatusDelivered
	}
	if notification.Status == StatusDelivered && nextStatus == StatusSent {
		return nil
	}
	if err := s.db.Model(&notification).Updates(map[string]any{
		"status": nextStatus, "updated_at": time.Now(),
	}).Error; err != nil {
		return err
	}
	if err := s.db.Create(&entities.NotificationLog{
		NotificationID: notification.ID,
		Status:         nextStatus,
		Attempt:        notification.Attempts,
		Details:        details,
	}).Error; err != nil {
		return err
	}
	return syncBroadcastStatus(s.db, notification.BroadcastID)
}

func syncBroadcastStatus(db *gorm.DB, broadcastID *string) error {
	if broadcastID == nil || *broadcastID == "" {
		return nil
	}
	var result struct {
		Total     int64
		Active    int64
		Delivered int64
	}
	if err := db.Model(&entities.Notification{}).
		Select("COUNT(*) AS total, SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) AS active, SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) AS delivered",
			[]string{StatusQueued, StatusSending}, []string{StatusSent, StatusDelivered}).
		Where("broadcast_id = ?", *broadcastID).Scan(&result).Error; err != nil {
		return err
	}
	if result.Total == 0 || result.Active > 0 {
		return nil
	}
	status := "failed"
	if result.Delivered > 0 {
		status = "sent"
	}
	return db.Model(&entities.BroadcastHistory{}).Where("id = ? AND status = ?", *broadcastID, "pending").Update("status", status).Error
}

func (s *Service) UpdateDelivery(messageID, status string) error {
	return s.ApplyDeliveryStatus(messageID, status, "delivery status received from WhatsApp gateway")
}

func (s *Service) CountByStatus() (map[string]int64, error) {
	type countRow struct {
		Status string
		Count  int64
	}
	var rows []countRow
	if err := s.db.Model(&entities.Notification{}).Select("status, COUNT(*) AS count").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}

func (s *Service) CheckInvoiceStillUnpaid(invoiceID string) (bool, error) {
	var invoice entities.Invoice
	if err := s.db.Select("id", "status").First(&invoice, "id = ?", invoiceID).Error; err != nil {
		return false, err
	}
	return invoice.Status == entities.InvoiceStatusUnpaid || invoice.Status == entities.InvoiceStatusPending, nil
}

func (s *Service) Log(notificationID, status string, attempt int, details string) error {
	return s.db.Create(&entities.NotificationLog{
		NotificationID: notificationID,
		Status:         status,
		Attempt:        attempt,
		Details:        details,
	}).Error
}

func LogEnqueueError(err error, context string) {
	if err != nil {
		log.Printf("[notifications] %s: %v", context, err)
	}
}
