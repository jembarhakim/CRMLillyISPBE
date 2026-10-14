package dude_status

type DeviceStatus struct {
	DeviceName string `json:"device_name"`
	IPAddress  string `json:"ip_address"`
	Status     string `json:"status"`
}

type UpdateDeviceDetailsRequest struct {
	UserName  string   `json:"user_name"`
	UserPhone string   `json:"user_phone"`
	Address   string   `json:"address"`
	Notes     string   `json:"notes"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}
