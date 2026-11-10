package dashboard

import (
	"fmt"
	midtrans "skripsi-be/internal/api/common/midtrands"
	"skripsi-be/internal/models/dto"
	"skripsi-be/internal/models/entities"
)

type CustomerDashboardServiceInterface interface {
	MyUserCustomerDashboard(request string) (dto.DashboardDTO, error)
	CreatePaymentCustomerDashboard(request SearchInvoice) (midtrans.MidtransResponsePaymentLink, error)
	CheckDeviceStatus(customerID string) (string, error)
	GetAvailableProducts() ([]entities.Products, error)
	GetCustomerBills(customerID string) ([]CustomerBillItem, error)
	GetCustomerInvoiceForPDF(customerID string, invoiceID string) (entities.Invoice, error)
	CreateCustomerSnapToken(customerID string, invoiceID string) (CustomerSnapToken, error)
}

// CustomerSnapToken is the payload used by the portal to open the Midtrans
// Snap popup (QRIS / VA / e-wallet) without leaving the page.
type CustomerSnapToken struct {
	InvoiceID   string `json:"invoice_id"`
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
	Amount      int64  `json:"amount"`
	ExpiresIn   int    `json:"expires_in"` // seconds the popup stays valid
}

type CustomerDashboardServiceStruct struct {
	repository CustomerDashboardRepositoryInterface
}

func NewCustomerDashboardService(repository CustomerDashboardRepositoryInterface) CustomerDashboardServiceStruct {
	return CustomerDashboardServiceStruct{repository}
}

func (s CustomerDashboardServiceStruct) MyUserCustomerDashboard(request string) (dto.DashboardDTO, error) {
	dashboard := dto.DashboardDTO{}
	myAccount, err := s.repository.MyUserCustomerDashboard(request)
	if err != nil {
		fmt.Printf("CustomerDashboard - Error getting customer: %v\n", err)
		return dashboard, err
	}

	fmt.Printf("CustomerDashboard - Customer ID: %s, Name: %s\n", myAccount.ID, myAccount.Name)

	// Get network device data with mac_address and product info
	networkDevice, err := s.repository.GetNetworkDeviceData(myAccount.ID)
	if err != nil {
		fmt.Printf("CustomerDashboard - Error getting network device: %v\n", err)
		return dashboard, err
	}

	fmt.Printf("CustomerDashboard - Network Device ID: %s, MAC: %v, ProductID: %v\n",
		networkDevice.ID, networkDevice.MacAddress, networkDevice.ProductID)

	// If no network device found, create empty product
	var MyProduct entities.Products
	if networkDevice.ID != "" && networkDevice.Product != nil {
		MyProduct = *networkDevice.Product
		fmt.Printf("CustomerDashboard - Product found: ID=%s, Name=%s, Price=%v\n",
			MyProduct.ID, MyProduct.Name, MyProduct.Price)
	} else {
		fmt.Printf("CustomerDashboard - No product found. NetworkDevice.ID=%s, Product=%v\n",
			networkDevice.ID, networkDevice.Product)

		// Fallback: Try to get product using the old method
		productID, err := s.repository.GetProductIDFromNetworkDevice(myAccount.ID)
		if err == nil && productID != "" {
			fmt.Printf("CustomerDashboard - Fallback: Found product_id=%s\n", productID)
			MyProduct, err = s.repository.MyProductCustomerDashboard(productID)
			if err != nil {
				fmt.Printf("CustomerDashboard - Fallback: Error getting product: %v\n", err)
			} else {
				fmt.Printf("CustomerDashboard - Fallback: Product found: ID=%s, Name=%s, Price=%v\n",
					MyProduct.ID, MyProduct.Name, MyProduct.Price)
			}
		} else {
			fmt.Printf("CustomerDashboard - Fallback: No product_id found\n")

			// Additional fallback: Try to get customer with product relationship
			_, err = s.repository.GetCustomerWithProduct(myAccount.ID)
			if err == nil {
				fmt.Printf("CustomerDashboard - Customer found but no product relationship available\n")
			} else {
				fmt.Printf("CustomerDashboard - No customer found: %v\n", err)
			}
		}
	}

	MyInvoice, err := s.repository.MyInvoiceCustomerDashboard(myAccount.ID)
	if err != nil {
		fmt.Printf("CustomerDashboard - Error getting invoices: %v\n", err)
		return dashboard, err
	}

	fmt.Printf("CustomerDashboard - Found %d invoices\n", len(MyInvoice))

	dashboard.Customer = myAccount
	dashboard.Product = MyProduct
	dashboard.Invoice = MyInvoice
	// Add network device data for frontend to use for status checking
	dashboard.NetworkDevice = networkDevice

	fmt.Printf("CustomerDashboard - Final dashboard: Customer=%s, Product=%s, NetworkDevice=%s\n",
		dashboard.Customer.Name, dashboard.Product.Name, dashboard.NetworkDevice.ID)

	return dashboard, nil
}

func (s CustomerDashboardServiceStruct) CreatePaymentCustomerDashboard(request SearchInvoice) (midtrans.MidtransResponsePaymentLink, error) {
	midtransResponse := midtrans.MidtransResponsePaymentLink{}
	invoice, err := s.repository.MyInvoiceIdCustomerDashboard(request)
	if err != nil {
		return midtransResponse, err
	}

	mindtransRequest := midtrans.MidtransCreatePaymentLink{
		OrderID:     invoice.ID,
		GrossAmount: invoice.Amount,
	}
	midtransResponse, err = midtrans.CreatePaymentLink(mindtransRequest)

	if err != nil {
		return midtransResponse, err
	}

	// Update invoice with payment link
	_, err = s.repository.MyInvoiceUpdatePaymentCustomerDashboard(request, midtransResponse.RedirectURL)
	if err != nil {
		return midtransResponse, err
	}

	return midtransResponse, nil

}

func (s CustomerDashboardServiceStruct) CheckDeviceStatus(customerID string) (string, error) {
	// Get network device data for the customer
	networkDevice, err := s.repository.GetNetworkDeviceData(customerID)
	if err != nil {
		return "off", err
	}

	// If no network device found, return "off"
	if networkDevice.ID == "" {
		return "off", nil
	}

	// If no mac_address, return "off"
	if networkDevice.MacAddress == nil || *networkDevice.MacAddress == "" {
		return "off", nil
	}

	// Check device status via Mikrotik
	status, err := s.repository.CheckDeviceStatus(*networkDevice.MacAddress)
	if err != nil {
		return "unknown", err
	}

	return status, nil
}

func (s CustomerDashboardServiceStruct) GetAvailableProducts() ([]entities.Products, error) {
	fmt.Printf("CustomerDashboard - Getting available products\n")

	products, err := s.repository.GetAvailableProducts()
	if err != nil {
		fmt.Printf("CustomerDashboard - Error getting products: %v\n", err)
		return products, err
	}

	fmt.Printf("CustomerDashboard - Found %d available products\n", len(products))
	return products, nil
}

// monthNames maps time.Month to Indonesian month names used on the bill table.
var monthNames = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// invoiceStatusLabel converts the raw invoice status into a label that is
// friendly for non technical customers.
func invoiceStatusLabel(status entities.InvoiceStatus) string {
	switch status {
	case entities.InvoiceStatusPaid:
		return "Lunas"
	case entities.InvoiceStatusPending:
		return "Pending"
	default:
		return "Belum Bayar"
	}
}

// toCustomerBillItem converts an invoice row into a table row for the portal.
func toCustomerBillItem(invoice entities.Invoice, index int) CustomerBillItem {
	// Prefer invoice_date, then due_date, then created_at for the period.
	period := invoice.CreatedAt
	if invoice.InvoiceDate != nil {
		period = *invoice.InvoiceDate
	} else if invoice.DueDate != nil {
		period = *invoice.DueDate
	}

	item := CustomerBillItem{
		InvoiceID:     invoice.ID,
		InvoiceNumber: fmt.Sprintf("INV-%03d", index+1),
		Month:         fmt.Sprintf("%s %d", monthNames[int(period.Month())-1], period.Year()),
		MonthSort:     period.Format("2006-01"),
		Amount:        invoice.Amount,
		Status:        string(invoice.Status),
		StatusLabel:   invoiceStatusLabel(invoice.Status),
		InvoiceDate:   period.Format("2006-01-02"),
		IsPayable:     invoice.Status != entities.InvoiceStatusPaid,
		RedirectURL:   invoice.Link,
		PdfViewed:     invoice.PdfViewed,
	}
	if invoice.DueDate != nil {
		item.DueDate = invoice.DueDate.Format("2006-01-02")
	}
	return item
}

// GetCustomerBills returns the monthly bill list shown on the customer portal.
// Invoices are ordered from the most recent billing period to the oldest.
func (s CustomerDashboardServiceStruct) GetCustomerBills(customerID string) ([]CustomerBillItem, error) {
	invoices, err := s.repository.MyInvoiceCustomerDashboard(customerID)
	if err != nil {
		return nil, err
	}

	bills := make([]CustomerBillItem, 0, len(invoices))
	for i, invoice := range invoices {
		bills = append(bills, toCustomerBillItem(invoice, i))
	}
	return bills, nil
}

// GetCustomerInvoiceForPDF returns one invoice owned by the customer so the
// portal can render the official receipt/invoice as a PDF client side.
func (s CustomerDashboardServiceStruct) GetCustomerInvoiceForPDF(customerID string, invoiceID string) (entities.Invoice, error) {
	if invoiceID == "" {
		return entities.Invoice{}, fmt.Errorf("invoice id is required")
	}
	return s.repository.MyInvoiceDetailCustomerDashboard(customerID, invoiceID)
}

// CreateCustomerSnapToken creates (or refreshes) a Midtrans Snap token for one
// invoice that belongs to the customer. The customer_id filter inside the
// repository makes sure a customer cannot pay/steal another customer's invoice.
func (s CustomerDashboardServiceStruct) CreateCustomerSnapToken(customerID string, invoiceID string) (CustomerSnapToken, error) {
	result := CustomerSnapToken{InvoiceID: invoiceID}

	invoice, err := s.repository.MyInvoiceIdCustomerDashboard(SearchInvoice{
		idUser:    idUser{UserId: customerID},
		InvoiceId: invoiceID,
	})
	if err != nil {
		return result, err
	}

	if invoice.Status == entities.InvoiceStatusPaid {
		return result, fmt.Errorf("tagihan ini sudah lunas")
	}

	link, err := midtrans.CreatePaymentLink(midtrans.MidtransCreatePaymentLink{
		OrderID:     invoice.ID,
		GrossAmount: invoice.Amount,
	})
	if err != nil {
		return result, err
	}

	// Persist the link and mark the invoice as pending payment so the admin
	// dashboard sees the same state as before.
	if _, err := s.repository.MyInvoiceUpdatePaymentCustomerDashboard(SearchInvoice{
		idUser:    idUser{UserId: customerID},
		InvoiceId: invoiceID,
	}, link.RedirectURL); err != nil {
		return result, err
	}

	result.Token = link.Token
	result.RedirectURL = link.RedirectURL
	result.Amount = invoice.Amount
	result.ExpiresIn = 24 * 60 * 60

	return result, nil
}
