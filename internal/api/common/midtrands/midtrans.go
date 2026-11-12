package midtrans

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	url := "https://app.sandbox.midtrans.com/snap/v1/transactions"
	method := "POST"

	// payload := strings.NewReader(`{
	// 	"transaction_details": {
	// 		"order_id": ` + midtrans.OrderID + `,
	// 		"gross_amount": ` + fmt.Sprintf("%d", midtrans.GrossAmount) + `
	// 	},
	// 	"item_details": [
	// 		{
	// 			"id": "'ITEM1'",
	// 			"price": 10000,
	// 			"quantity": 1,
	// 			"name": "Midtrans Bear",
	// 			"brand"/*  */: "Midtrans",
	// 			"category": "Toys",
	// 			"merchant_name": "Midtrans",
	// 			"url": "http://toko/toko1?item=abc"
	// 		}
	// 	],
	// 	"credit_card": {
	// 		"secure": true
	// 	}
	// }`)

	payload := strings.NewReader(fmt.Sprintf(`{
		"transaction_details": {
			"order_id": %q,
			"gross_amount": %d
		},
		"credit_card": {
			"secure": true
		}
	}`, midtrans.OrderID, midtrans.GrossAmount))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return MidtransResponsePaymentLink{}, err
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", "Basic U0ItTWlkLXNlcnZlci13TkZDY1J2bENUdmwwb0RaQkdaNXBHZUM6")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return MidtransResponsePaymentLink{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return MidtransResponsePaymentLink{}, err
	}
	fmt.Println(string(body))
	var response MidtransResponsePaymentLink
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println(err)
		return MidtransResponsePaymentLink{}, err
	}

	return response, nil
}
