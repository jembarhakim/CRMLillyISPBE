package dashboard

import (
	"time"

	"gorm.io/gorm"

	"fmt"
	"skripsi-be/internal/models/entities"
	"sort"
)

type AdminDashboardRepositoryInterface interface {
	CardCustomer() (map[string]interface{}, error)
	CardPacketPopular() (map[string]interface{}, error)
	CardAreaPopular() (map[string]interface{}, error)
	CardReportCash() (map[string]interface{}, error)
	GetTotalIncome() (int64, error)
	GetTotalExpenses() (int64, error)
	GetTotalCustomer() (int64, error)
	GetDashboardStats(start *time.Time, end *time.Time) (map[string]interface{}, error)
	GetRecentInvoices() ([]map[string]interface{}, error)
	GetRecentTransactions() ([]map[string]interface{}, error)
	GetCustomerGrowth(start *time.Time, end *time.Time) (map[string]interface{}, error)
	GetRevenueChart(start *time.Time, end *time.Time) (map[string]interface{}, error)
	GetExpensesChart(start *time.Time, end *time.Time) (map[string]interface{}, error)
	GetUnpaidCustomersChart(start *time.Time, end *time.Time) (map[string]interface{}, error)
	GetUnpaidCustomersList() ([]map[string]interface{}, error)
}

type AdminDashboardRepositoryStruct struct {
	db *gorm.DB
}

func NewAdminDashboradRepository(db *gorm.DB) AdminDashboardRepositoryStruct {
	return AdminDashboardRepositoryStruct{db}
}

type DailyCount struct {
	Date  time.Time
	Count int
}

type PacketCount struct {
	Name  string `json:"name" gorm:"column:name"`
	Count int    `json:"count" gorm:"column:count"`
}

type AreaCount struct {
	NameCity        string `json:"name_city"`
	NameSubdistrict string `json:"name_subdistrict"`
	NameVillage     string `json:"name_village"`
	Count           int    `json:"count"`
}

func (r AdminDashboardRepositoryStruct) CardCustomer() (map[string]interface{}, error) {
	var total int64
	if err := r.db.Model(&entities.Customer{}).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
		return nil, err
	}

	// Build a stable local-day window to avoid timezone drift
	now := time.Now().In(time.Local)
	startDay := time.Date(now.Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, time.Local)
	endDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)

	var results []DailyCount
	err := r.db.
		Model(&entities.Customer{}).
		Select("DATE(createdAt) as date, COUNT(*) as count").
		Where("DATE(createdAt) BETWEEN ? AND ?", startDay.Format("2006-01-02"), endDay.Format("2006-01-02")).
		Group("DATE(createdAt)").
		Order("date").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Map the results by date
	dateMap := make(map[string]int)
	for _, r := range results {
		dateMap[r.Date.Format("2006-01-02")] = r.Count
	}

	// Build 7-day graph data
	graphData := make([]map[string]interface{}, 0)
	for i := 6; i >= 0; i-- {
		date := time.Date(now.Year(), now.Month(), now.Day()-i, 0, 0, 0, 0, time.Local).Format("2006-01-02")
		graphData = append(graphData, map[string]interface{}{
			"date":  date,
			"count": dateMap[date],
		})
	}

	// Final result
	data := map[string]interface{}{
		"total_customer": total,
		"graph_customer": graphData,
	}

	return data, nil
}

func (r AdminDashboardRepositoryStruct) CardPacketPopular() (map[string]interface{}, error) {
	var total int64
	if err := r.db.Model(&entities.Customer{}).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
		return nil, err
	}

	var results []PacketCount
	// Join products through network_devices as product is linked to device, not directly on customer
	err := r.db.
		Model(&entities.NetworkDevice{}).
		Joins("JOIN products ON products.id = network_devices.product_id").
		Select("COUNT(*) as count, products.name").
		Group("products.id, products.name").
		Scan(&results).Error
	if err != nil {
		// If no data found, return empty results instead of error
		results = []PacketCount{}
	}

	// Final result
	data := map[string]interface{}{
		"total":                total,
		"graph_packet_popular": results,
	}

	return data, nil
}

func (r AdminDashboardRepositoryStruct) CardAreaPopular() (map[string]interface{}, error) {
	var total int64
	if err := r.db.Model(&entities.Customer{}).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
		return nil, err
	}

	var results []AreaCount
	err := r.db.
		Model(&entities.Customer{}).
		Where("deleted_at IS NULL").
		Joins("JOIN areas ON areas.id = customer.area_id").
		Select("COUNT(*) as count, areas.name_city,areas.name_subdistrict,areas.name_village").
		Group("areas.id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Final result
	data := map[string]interface{}{
		"total":              total,
		"graph_area_popular": results,
	}

	return data, nil
}

func (r AdminDashboardRepositoryStruct) CardReportCash() (map[string]interface{}, error) {
	// card report cash by date

	type ReportCash struct {
		Date   time.Time `json:"date"`
		Amount int       `json:"amount"`
	}

	// Limit to last 7 days with explicit local-day window to avoid timezone mismatches
	now := time.Now().In(time.Local)
	startDay := time.Date(now.Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, time.Local)

	var Transactions []ReportCash
	if err := r.db.Model(&entities.Transaction{}).
		Group("DATE(date)").
		Select("DATE(date) as date, COALESCE(SUM(amount), 0) as amount").
		Order("date DESC").
		Where("type_in_out = ? AND DATE(date) >= ?", entities.TransactionsTypeInOutIn, startDay.Format("2006-01-02")).
		Find(&Transactions).Error; err != nil {
		return nil, err
	}
	// Map the results by date
	dateMap := make(map[string]int)
	for _, r := range Transactions {
		dateMap[r.Date.Format("2006-01-02")] = int(r.Amount)
	}

	// Group transactions by date
	graphData := make([]map[string]interface{}, 0)
	for i := 6; i >= 0; i-- {
		date := time.Date(now.Year(), now.Month(), now.Day()-i, 0, 0, 0, 0, time.Local).Format("2006-01-02")
		graphData = append(graphData, map[string]interface{}{
			"date":  date,
			"count": dateMap[date],
		})
	}

	data := map[string]interface{}{
		"graph_report_cash": graphData,
		"total":             0,
	}
	return data, nil
}

func (r AdminDashboardRepositoryStruct) GetTotalIncome() (int64, error) {
	var totalIncome int64
	if err := r.db.Model(&entities.Transaction{}).
		Where("type_in_out = ?", entities.TransactionsTypeInOutIn).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalIncome).Error; err != nil {
		return 0, err
	}

	return totalIncome, nil
}

func (r AdminDashboardRepositoryStruct) GetTotalExpenses() (int64, error) {
	var totalIncome int64
	if err := r.db.Model(&entities.Transaction{}).
		Where("type_in_out = ?", entities.TransactionsTypeInOutOut).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalIncome).Error; err != nil {
		return 0, err
	}

	return totalIncome, nil
}

func (r AdminDashboardRepositoryStruct) GetTotalCustomer() (int64, error) {
	var total int64
	if err := r.db.Model(&entities.Customer{}).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}
func (r AdminDashboardRepositoryStruct) GetDashboardStats(start *time.Time, end *time.Time) (map[string]interface{}, error) {
	fmt.Printf("🔍 [DEBUG] GetDashboardStats called with start: %v, end: %v\n", start, end)

	// Get total customers (filtered by creation date if date range provided)
	var totalCustomers int64
	customerQuery := r.db.Model(&entities.Customer{}).Where("deleted_at IS NULL")
	if start != nil && end != nil {
		fmt.Printf("🔍 [DEBUG] Filtering customers by createdAt BETWEEN %v AND %v\n", start, end)
		customerQuery = customerQuery.Where("createdAt BETWEEN ? AND ?", start, end)
	}
	if err := customerQuery.Count(&totalCustomers).Error; err != nil {
		fmt.Printf("❌ [ERROR] Failed to count customers: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ [DEBUG] Total customers: %d\n", totalCustomers)

	// Get total income (filtered by transaction date if date range provided)
	var totalIncome int64
	incomeQuery := r.db.Model(&entities.Transaction{}).
		Where("type_in_out = ?", entities.TransactionsTypeInOutIn)
	if start != nil && end != nil {
		fmt.Printf("🔍 [DEBUG] Filtering income by date BETWEEN %v AND %v\n", start, end)
		incomeQuery = incomeQuery.Where("date BETWEEN ? AND ?", start, end)
	}
	if err := incomeQuery.Select("COALESCE(SUM(amount), 0)").Scan(&totalIncome).Error; err != nil {
		fmt.Printf("❌ [ERROR] Failed to get total income: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ [DEBUG] Total income: %d\n", totalIncome)

	// Get total expenses (filtered by transaction date if date range provided)
	var totalExpenses int64
	expenseQuery := r.db.Model(&entities.Transaction{}).
		Where("type_in_out = ?", entities.TransactionsTypeInOutOut)
	if start != nil && end != nil {
		fmt.Printf("🔍 [DEBUG] Filtering expenses by date BETWEEN %v AND %v\n", start, end)
		expenseQuery = expenseQuery.Where("date BETWEEN ? AND ?", start, end)
	}
	if err := expenseQuery.Select("COALESCE(SUM(amount), 0)").Scan(&totalExpenses).Error; err != nil {
		fmt.Printf("❌ [ERROR] Failed to get total expenses: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ [DEBUG] Total expenses: %d\n", totalExpenses)

	// Get total invoices (filtered by creation date if date range provided)
	var totalInvoices int64
	invoiceQuery := r.db.Model(&entities.Invoice{})
	if start != nil && end != nil {
		fmt.Printf("🔍 [DEBUG] Filtering invoices by createdAt BETWEEN %v AND %v\n", start, end)
		invoiceQuery = invoiceQuery.Where("createdAt BETWEEN ? AND ?", start, end)
	}
	if err := invoiceQuery.Count(&totalInvoices).Error; err != nil {
		fmt.Printf("❌ [ERROR] Failed to count invoices: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ [DEBUG] Total invoices: %d\n", totalInvoices)

	// Get total areas (not filtered by date as areas are static)
	var totalAreas int64
	if err := r.db.Model(&entities.Areas{}).Count(&totalAreas).Error; err != nil {
		fmt.Printf("❌ [ERROR] Failed to count areas: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ [DEBUG] Total areas: %d\n", totalAreas)

	// Get total products (not filtered by date as products are static)
	var totalProducts int64
	if err := r.db.Model(&entities.Products{}).Count(&totalProducts).Error; err != nil {
		fmt.Printf("❌ [ERROR] Failed to count products: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ [DEBUG] Total products: %d\n", totalProducts)

	// Get total trouble tickets (filtered by creation date if date range provided)
	var totalTickets int64
	ticketQuery := r.db.Model(&entities.TroubleTicket{})
	if start != nil && end != nil {
		fmt.Printf("🔍 [DEBUG] Filtering tickets by created_at BETWEEN %v AND %v\n", start, end)
		ticketQuery = ticketQuery.Where("created_at BETWEEN ? AND ?", start, end)
	}
	if err := ticketQuery.Count(&totalTickets).Error; err != nil {
		fmt.Printf("❌ [ERROR] Failed to count tickets: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ [DEBUG] Total tickets: %d\n", totalTickets)

	// Calculate net worth
	netWorth := totalIncome - totalExpenses

	data := map[string]interface{}{
		"total_customers": totalCustomers,
		"total_income":    totalIncome,
		"total_expenses":  totalExpenses,
		"net_worth":       netWorth,
		"total_invoices":  totalInvoices,
		"total_areas":     totalAreas,
		"total_products":  totalProducts,
		"total_tickets":   totalTickets,
	}

	fmt.Printf("✅ [DEBUG] GetDashboardStats completed successfully\n")
	fmt.Printf("📊 [DEBUG] Dashboard Stats: %+v\n", data)

	return data, nil
}

func (r AdminDashboardRepositoryStruct) GetRecentInvoices() ([]map[string]interface{}, error) {
	// First, let's check if invoices table exists and has data
	var totalInvoices int64
	if err := r.db.Model(&entities.Invoice{}).Count(&totalInvoices).Error; err != nil {
		return nil, err
	}

	// If no invoices, return empty array instead of error
	if totalInvoices == 0 {
		return []map[string]interface{}{}, nil
	}

	var invoices []entities.Invoice
	if err := r.db.Preload("Customer").
		Order("createdAt DESC").
		Limit(10).
		Find(&invoices).Error; err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, invoice := range invoices {
		customerName := ""
		if invoice.Customer.Name != "" {
			customerName = invoice.Customer.Name
		}

		result = append(result, map[string]interface{}{
			"id":         invoice.ID,
			"invoice_no": invoice.ID, // Using ID as invoice number since InvoiceNo field doesn't exist
			"amount":     invoice.Amount,
			"status":     invoice.Status,
			"customer":   customerName,
			"created_at": invoice.CreatedAt,
		})
	}

	return result, nil
}

func (r AdminDashboardRepositoryStruct) GetRecentTransactions() ([]map[string]interface{}, error) {
	// First, let's check if transactions table exists and has data
	var totalTransactions int64
	if err := r.db.Model(&entities.Transaction{}).Count(&totalTransactions).Error; err != nil {
		return nil, err
	}

	// If no transactions, return empty array instead of error
	if totalTransactions == 0 {
		return []map[string]interface{}{}, nil
	}

	var transactions []entities.Transaction
	if err := r.db.Order("date DESC").
		Limit(10).
		Find(&transactions).Error; err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, transaction := range transactions {
		result = append(result, map[string]interface{}{
			"id":          transaction.ID,
			"amount":      transaction.Amount,
			"type_in_out": transaction.TypeInOut,
			"description": transaction.Description,
			"date":        transaction.Date,
			"created_at":  transaction.CreatedAt,
		})
	}

	return result, nil
}

func (r AdminDashboardRepositoryStruct) GetCustomerGrowth(start *time.Time, end *time.Time) (map[string]interface{}, error) {
	// First, let's check if customers table exists and has data
	var totalCustomers int64
	if err := r.db.Model(&entities.Customer{}).Where("deleted_at IS NULL").Count(&totalCustomers).Error; err != nil {
		return nil, err
	}

	// If no customers, return empty data instead of error
	if totalCustomers == 0 {
		data := map[string]interface{}{
			"customer_growth": []map[string]interface{}{},
		}
		return data, nil
	}

	var results []DailyCount
	query := r.db.Model(&entities.Customer{}).
		Select("DATE(createdAt) as date, COUNT(*) as count")
	if start != nil && end != nil {
		// Expand to full local-day bounds
		s := time.Date(start.In(time.Local).Year(), start.In(time.Local).Month(), start.In(time.Local).Day(), 0, 0, 0, 0, time.Local)
		e := time.Date(end.In(time.Local).Year(), end.In(time.Local).Month(), end.In(time.Local).Day(), 23, 59, 59, 0, time.Local)
		query = query.Where("createdAt BETWEEN ? AND ?", s, e)
	}
	err := query.Group("DATE(createdAt)").Order("date").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Map the results by date
	dateMap := make(map[string]int)
	for _, r := range results {
		dateMap[r.Date.Format("2006-01-02")] = r.Count
	}

	// Build graph data across requested window (or full history by default)
	graphData := make([]map[string]interface{}, 0)
	if start != nil && end != nil {
		for d := start.Truncate(24 * time.Hour); !d.After(*end); d = d.AddDate(0, 0, 1) {
			ds := d.Format("2006-01-02")
			graphData = append(graphData, map[string]interface{}{
				"date":  ds,
				"count": dateMap[ds],
			})
		}
	} else {
		if len(results) > 0 {
			startDate := results[0].Date
			endDate := results[len(results)-1].Date
			for d := startDate.Truncate(24 * time.Hour); !d.After(endDate); d = d.AddDate(0, 0, 1) {
				ds := d.Format("2006-01-02")
				graphData = append(graphData, map[string]interface{}{
					"date":  ds,
					"count": dateMap[ds],
				})
			}
		}
	}

	data := map[string]interface{}{
		"customer_growth": graphData,
	}

	return data, nil
}

func (r AdminDashboardRepositoryStruct) GetRevenueChart(start *time.Time, end *time.Time) (map[string]interface{}, error) {
	// First, check if there are any transactions
	var totalTransactions int64
	if err := r.db.Model(&entities.Transaction{}).Where("type_in_out = ?", entities.TransactionsTypeInOutIn).Count(&totalTransactions).Error; err != nil {
		return nil, err
	}

	// If no transactions, return empty data
	if totalTransactions == 0 {
		data := map[string]interface{}{
			"revenue_chart": []map[string]interface{}{},
		}
		return data, nil
	}

	var results []DailyCount
	query := r.db.Model(&entities.Transaction{}).
		Select("DATE(date) as date, COALESCE(SUM(amount), 0) as count").
		Where("type_in_out = ?", entities.TransactionsTypeInOutIn)
	if start != nil && end != nil {
		query = query.Where("date BETWEEN ? AND ?", start, end)
	}
	err := query.Group("DATE(date)").Order("date").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Map the results by date
	dateMap := make(map[string]int)
	for _, r := range results {
		dateMap[r.Date.Format("2006-01-02")] = r.Count
	}

	// Build graph data across requested window (or full history by default)
	graphData := make([]map[string]interface{}, 0)
	if start != nil && end != nil {
		for d := start.Truncate(24 * time.Hour); !d.After(*end); d = d.AddDate(0, 0, 1) {
			ds := d.Format("2006-01-02")
			graphData = append(graphData, map[string]interface{}{
				"date":   ds,
				"amount": dateMap[ds],
			})
		}
	} else {
		if len(results) > 0 {
			// Normalize to local-day boundaries to avoid timezone off-by-one
			st := results[0].Date.In(time.Local)
			en := results[len(results)-1].Date.In(time.Local)
			startLocal := time.Date(st.Year(), st.Month(), st.Day(), 0, 0, 0, 0, time.Local)
			endLocal := time.Date(en.Year(), en.Month(), en.Day(), 0, 0, 0, 0, time.Local)
			for d := startLocal; !d.After(endLocal); d = d.AddDate(0, 0, 1) {
				ds := d.Format("2006-01-02")
				graphData = append(graphData, map[string]interface{}{
					"date":   ds,
					"amount": dateMap[ds],
				})
			}
		}
	}

	data := map[string]interface{}{
		"revenue_chart": graphData,
	}

	return data, nil
}

func (r AdminDashboardRepositoryStruct) GetExpensesChart(start *time.Time, end *time.Time) (map[string]interface{}, error) {
	// Check if there are any expense transactions
	var totalTransactions int64
	if err := r.db.Model(&entities.Transaction{}).Where("type_in_out = ?", entities.TransactionsTypeInOutOut).Count(&totalTransactions).Error; err != nil {
		return nil, err
	}
	if totalTransactions == 0 {
		data := map[string]interface{}{
			"expenses_chart": []map[string]interface{}{},
		}
		return data, nil
	}

	var results []DailyCount
	query := r.db.Model(&entities.Transaction{}).
		Select("DATE(date) as date, COALESCE(SUM(amount),0) as count").
		Where("type_in_out = ?", entities.TransactionsTypeInOutOut)
	if start != nil && end != nil {
		query = query.Where("date BETWEEN ? AND ?", start, end)
	}
	err := query.Group("DATE(date)").Order("date").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	dateMap := make(map[string]int)
	for _, r := range results {
		dateMap[r.Date.Format("2006-01-02")] = r.Count
	}

	graphData := make([]map[string]interface{}, 0)
	if start != nil && end != nil {
		for d := start.Truncate(24 * time.Hour); !d.After(*end); d = d.AddDate(0, 0, 1) {
			ds := d.Format("2006-01-02")
			graphData = append(graphData, map[string]interface{}{
				"date":   ds,
				"amount": dateMap[ds],
			})
		}
	} else if len(results) > 0 {
		st := results[0].Date.In(time.Local)
		en := results[len(results)-1].Date.In(time.Local)
		startLocal := time.Date(st.Year(), st.Month(), st.Day(), 0, 0, 0, 0, time.Local)
		endLocal := time.Date(en.Year(), en.Month(), en.Day(), 0, 0, 0, 0, time.Local)
		for d := startLocal; !d.After(endLocal); d = d.AddDate(0, 0, 1) {
			ds := d.Format("2006-01-02")
			graphData = append(graphData, map[string]interface{}{
				"date":   ds,
				"amount": dateMap[ds],
			})
		}
	}

	data := map[string]interface{}{
		"expenses_chart": graphData,
	}
	return data, nil
}

func (r AdminDashboardRepositoryStruct) GetUnpaidCustomersChart(start *time.Time, end *time.Time) (map[string]interface{}, error) {
	// If no invoices, return empty
	var totalInvoices int64
	if err := r.db.Model(&entities.Invoice{}).Count(&totalInvoices).Error; err != nil {
		return nil, err
	}
	if totalInvoices == 0 {
		return map[string]interface{}{"unpaid_customers_chart": []map[string]interface{}{}}, nil
	}

	type DailyDistinct struct {
		Date   time.Time
		Count  int
		Status string
	}

	// Normalize dates to local timezone before taking DATE() to avoid off-by-one
	offsetSeconds := time.Now().In(time.Local).Second() - time.Now().UTC().Second() // dummy to keep imports; replaced below
	_ = offsetSeconds

	var results []DailyDistinct
	// Use due_date to define the day for invoices with outstanding balances
	// This query finds invoices where the total paid is less than the invoice amount
	// and categorizes them by status (unpaid vs pending)

	// First, let's get all unpaid invoices with their status
	selectExpr := `
		DATE(i.due_date) as date,
		COUNT(DISTINCT i.customer_id) as count,
		CASE
			WHEN COALESCE(t.total_paid, 0) = 0 THEN 'unpaid'
			WHEN COALESCE(t.total_paid, 0) < i.amount THEN 'pending'
			ELSE 'paid'
		END as status
	`

	// Join with transactions to calculate total paid per invoice
	// Only include invoices where total_paid < amount (meaning they have outstanding balance)
	query := r.db.Table("invoices i").
		Select(selectExpr).
		Joins(`LEFT JOIN (
			SELECT invoice_id, COALESCE(SUM(amount), 0) as total_paid 
			FROM transactions 
			WHERE invoice_id IS NOT NULL 
			GROUP BY invoice_id
		) t ON i.id = t.invoice_id`).
		Where("COALESCE(t.total_paid, 0) < i.amount AND i.status != ?", entities.InvoiceStatusPaid).
		Group("DATE(i.due_date), CASE WHEN COALESCE(t.total_paid, 0) = 0 THEN 'unpaid' WHEN COALESCE(t.total_paid, 0) < i.amount THEN 'pending' ELSE 'paid' END")

	if start != nil && end != nil {
		// Compare against due_date window
		s := time.Date(start.In(time.Local).Year(), start.In(time.Local).Month(), start.In(time.Local).Day(), 0, 0, 0, 0, time.Local)
		e := time.Date(end.In(time.Local).Year(), end.In(time.Local).Month(), end.In(time.Local).Day(), 23, 59, 59, 0, time.Local)
		query = query.Where("i.due_date BETWEEN ? AND ?", s, e)
	}

	fmt.Println("🔧 [DEBUG] GetUnpaidCustomersChart: Executing database query...")
	if err := query.Order("DATE(i.due_date), status").Scan(&results).Error; err != nil {
		fmt.Printf("🔧 [DEBUG] GetUnpaidCustomersChart: Database query error - %v\n", err)
		return nil, err
	}
	fmt.Printf("🔧 [DEBUG] GetUnpaidCustomersChart: Database query successful - found %d records\n", len(results))

	// Map by date and status
	dateStatusMap := make(map[string]map[string]int)
	for _, r := range results {
		dateStr := r.Date.Format("2006-01-02")
		if dateStatusMap[dateStr] == nil {
			dateStatusMap[dateStr] = make(map[string]int)
		}
		dateStatusMap[dateStr][r.Status] = r.Count
	}

	// Create separate series for unpaid and pending
	var allDates []string
	dateSet := make(map[string]bool)
	for dateStr := range dateStatusMap {
		if !dateSet[dateStr] {
			allDates = append(allDates, dateStr)
			dateSet[dateStr] = true
		}
	}

	// Sort dates
	sort.Strings(allDates)

	unpaidData := make([]map[string]interface{}, 0)
	pendingData := make([]map[string]interface{}, 0)

	for _, ds := range allDates {
		unpaidCount := 0
		pendingCount := 0

		if dateStatusMap[ds] != nil {
			unpaidCount = dateStatusMap[ds]["unpaid"]
			pendingCount = dateStatusMap[ds]["pending"]
		}

		unpaidData = append(unpaidData, map[string]interface{}{
			"date":  ds,
			"count": unpaidCount,
		})

		pendingData = append(pendingData, map[string]interface{}{
			"date":  ds,
			"count": pendingCount,
		})
	}

	graphData := map[string]interface{}{
		"unpaid":  unpaidData,
		"pending": pendingData,
	}

	return map[string]interface{}{"unpaid_customers_chart": graphData}, nil
}

func (r AdminDashboardRepositoryStruct) GetUnpaidCustomersList() ([]map[string]interface{}, error) {
	fmt.Println("🔧 [DEBUG] GetUnpaidCustomersList: Repository method called")
	// Get invoices with outstanding balances (where total_paid < amount)
	var results []map[string]interface{}

	query := r.db.Table("invoices i").
		Select(`
			i.id,
			i.amount,
			i.status,
			i.due_date,
			c.name as customer_name,
			c.phone as customer_phone,
			c.id as customer_id,
			COALESCE(t.total_paid, 0) as total_paid,
			(i.amount - COALESCE(t.total_paid, 0)) as outstanding_amount
		`).
		Joins("LEFT JOIN customer c ON i.customer_id = c.id AND c.deleted_at IS NULL").
		Joins(`LEFT JOIN (
			SELECT invoice_id, COALESCE(SUM(amount), 0) as total_paid 
			FROM transactions 
			WHERE invoice_id IS NOT NULL 
			GROUP BY invoice_id
		) t ON i.id = t.invoice_id`).
		Where("COALESCE(t.total_paid, 0) < i.amount AND i.status != ?", entities.InvoiceStatusPaid).
		Order("i.due_date ASC, i.id DESC")

	fmt.Println("🔧 [DEBUG] GetUnpaidCustomersList: Executing database query...")
	if err := query.Scan(&results).Error; err != nil {
		fmt.Printf("🔧 [DEBUG] GetUnpaidCustomersList: Database query error - %v\n", err)
		return nil, err
	}

	fmt.Printf("🔧 [DEBUG] GetUnpaidCustomersList: Database query successful - found %d records\n", len(results))
	return results, nil
}
