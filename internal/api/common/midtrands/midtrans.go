package midtrans

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type MidtransCreatePaymentLink struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
}

type MidtransResponsePaymentLink struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

func CreatePaymentLink(midtrans MidtransCreatePaymentLink) (MidtransResponsePaymentLink, error) {
	serverKey := strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY"))
	if serverKey == "" {
		return MidtransResponsePaymentLink{}, fmt.Errorf("MIDTRANS_SERVER_KEY is not configured")
	}

	baseURL := "https://app.sandbox.midtrans.com"
	if strings.EqualFold(os.Getenv("MIDTRANS_IS_PRODUCTION"), "true") {
		baseURL = "https://app.midtrans.com"
	}

	payload, err := json.Marshal(map[string]any{
		"transaction_details": map[string]any{
			"order_id":     midtrans.OrderID,
			"gross_amount": midtrans.GrossAmount,
		},
		"credit_card": map[string]bool{"secure": true},
	})
	if err != nil {
		return MidtransResponsePaymentLink{}, err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/snap/v1/transactions", strings.NewReader(string(payload)))
	if err != nil {
		return MidtransResponsePaymentLink{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(serverKey, "")

	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return MidtransResponsePaymentLink{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return MidtransResponsePaymentLink{}, err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return MidtransResponsePaymentLink{}, fmt.Errorf("Midtrans returned HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var response MidtransResponsePaymentLink
	if err := json.Unmarshal(body, &response); err != nil {
		return MidtransResponsePaymentLink{}, err
	}
	return response, nil
}
