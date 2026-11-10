package dashboard

import (
	"fmt"
	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type CustomerDashboardHandlerStruct struct {
	service CustomerDashboardServiceInterface
}

func NewCustomerDashboardHandler(service CustomerDashboardServiceInterface) CustomerDashboardHandlerStruct {
	return CustomerDashboardHandlerStruct{service}
}

func (h CustomerDashboardHandlerStruct) MyUserCustomerDashboard(c *fiber.Ctx) error {
	request := c.Locals("user_id").(string)

	// Add debug logging
	fmt.Printf("CustomerDashboard - User ID: %s\n", request)

	myAccount, err := h.service.MyUserCustomerDashboard(request)
	if err != nil {
		fmt.Printf("CustomerDashboard - Error: %v\n", err)
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	// Add debug logging for response
	fmt.Printf("CustomerDashboard - Response: %+v\n", myAccount)

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "My Account", myAccount)
}

func (h CustomerDashboardHandlerStruct) CreatePaymentCustomerDashboard(c *fiber.Ctx) error {

	request := new(CreatePaymentCustomerDashboardRequest)
	if err := c.BodyParser(request); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	// Validate the request
	errorMessages := validation.ValidationRequest(request)
	if errorMessages != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errorMessages, ", "), nil)
	}

	SearchInvoice := SearchInvoice{}

	SearchInvoice.UserId = c.Locals("user_id").(string)
	SearchInvoice.InvoiceId = request.InvoiceId

	payment, err := h.service.CreatePaymentCustomerDashboard(SearchInvoice)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "My Account", payment)
}

func (h CustomerDashboardHandlerStruct) CheckDeviceStatus(c *fiber.Ctx) error {
	request := c.Locals("user_id").(string)

	// Add debug logging
	fmt.Printf("CheckDeviceStatus - User ID: %s\n", request)

	status, err := h.service.CheckDeviceStatus(request)
	if err != nil {
		fmt.Printf("CheckDeviceStatus - Error: %v\n", err)
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	// Add debug logging for response
	fmt.Printf("CheckDeviceStatus - Status: %s\n", status)

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Device Status", map[string]string{"status": status})
}

func (h CustomerDashboardHandlerStruct) TestEndpoint(c *fiber.Ctx) error {
	fmt.Printf("TestEndpoint - Called\n")
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Test endpoint working", map[string]string{"message": "Customer dashboard test endpoint is working"})
}

func (h CustomerDashboardHandlerStruct) GetAvailableProducts(c *fiber.Ctx) error {
	fmt.Printf("GetAvailableProducts - Called\n")

	products, err := h.service.GetAvailableProducts()
	if err != nil {
		fmt.Printf("GetAvailableProducts - Error: %v\n", err)
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	fmt.Printf("GetAvailableProducts - Found %d products\n", len(products))
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Available Products", products)
}

// MyBills returns the monthly bill table for the logged in customer.
func (h CustomerDashboardHandlerStruct) MyBills(c *fiber.Ctx) error {
	customerID := c.Locals("user_id").(string)

	bills, err := h.service.GetCustomerBills(customerID)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "My Bills", bills)
}

// CreateSnapToken returns a Midtrans Snap token so the portal can open the
// payment popup (QRIS / Virtual Account / e-wallet) in place.
func (h CustomerDashboardHandlerStruct) CreateSnapToken(c *fiber.Ctx) error {
	request := new(CreatePaymentCustomerDashboardRequest)
	if err := c.BodyParser(request); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	errorMessages := validation.ValidationRequest(request)
	if errorMessages != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errorMessages, ", "), nil)
	}

	customerID := c.Locals("user_id").(string)

	snap, err := h.service.CreateCustomerSnapToken(customerID, request.InvoiceId)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Snap token created", snap)
}

// MyInvoiceDetail returns one invoice (owned by the customer) with its items so
// the portal can render the official PDF receipt.
func (h CustomerDashboardHandlerStruct) MyInvoiceDetail(c *fiber.Ctx) error {
	customerID := c.Locals("user_id").(string)
	invoiceID := c.Params("id")

	invoice, err := h.service.GetCustomerInvoiceForPDF(customerID, invoiceID)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "tagihan tidak ditemukan", nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Invoice Detail", invoice)
}
