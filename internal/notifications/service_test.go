package notifications

import (
	"testing"

	"skripsi-be/internal/models/entities"
)

func TestRenderReplacesKnownVariablesAndKeepsUnknownOnes(t *testing.T) {
	message := Render("Halo {{ nama_pelanggan }}, invoice {{no_invoice}} {{unknown}}", map[string]any{
		"nama_pelanggan": "Dymas",
		"no_invoice":     "INV-001",
	})
	expected := "Halo Dymas, invoice INV-001 {{unknown}}"
	if message != expected {
		t.Fatalf("Render() = %q, want %q", message, expected)
	}
}

func TestFormatIDR(t *testing.T) {
	if got := formatIDR(1234567); got != "Rp1.234.567" {
		t.Fatalf("formatIDR() = %q", got)
	}
}

func TestInvoicePaymentLinkUsesInvoiceIDForInternalRoutes(t *testing.T) {
	t.Setenv("APP_BASE_URL", "https://crm.example.com/")

	got := invoicePaymentLink(entities.Invoice{
		ID:   "invoice-id",
		Link: "/invoice/recurring-id",
	})
	want := "https://crm.example.com/invoice/invoice-id"
	if got != want {
		t.Fatalf("invoicePaymentLink() = %q, want %q", got, want)
	}
}

func TestInvoicePaymentLinkFallsBackWhenLinkIsEmpty(t *testing.T) {
	t.Setenv("APP_BASE_URL", "")

	got := invoicePaymentLink(entities.Invoice{ID: "invoice-id"})
	want := "/invoice/invoice-id"
	if got != want {
		t.Fatalf("invoicePaymentLink() = %q, want %q", got, want)
	}
}
