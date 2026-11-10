package dashboard

type CreatePaymentCustomerDashboardRequest struct {
	InvoiceId string `json:"invoice_id" validate:"required"`
}

type idUser struct {
	UserId string `json:"user_id" `
}

type SearchInvoice struct {
	idUser
	InvoiceId string `json:"invoice_id" `
}

// CustomerBillItem is one row of the customer bill table.
// "month" is the billing period derived from invoice_date (fallback created_at).
type CustomerBillItem struct {
	InvoiceID       string `json:"invoice_id"`
	InvoiceNumber   string `json:"invoice_number"`
	Month           string `json:"month"`         // e.g. "Oktober 2025"
	MonthSort       string `json:"month_sort"`    // e.g. "2025-10" (for sorting)
	Amount          int64  `json:"amount"`
	Status          string `json:"status"`        // paid | pending | unpaid
	StatusLabel     string `json:"status_label"`  // Lunas | Pending | Belum Bayar
	InvoiceDate     string `json:"invoice_date"`  // YYYY-MM-DD
	DueDate         string `json:"due_date"`      // YYYY-MM-DD
	IsPayable       bool   `json:"is_payable"`    // false for paid invoices
	RedirectURL     string `json:"redirect_url"`  // existing Midtrans link (may be empty)
	PdfViewed       bool   `json:"pdf_viewed"`
}
