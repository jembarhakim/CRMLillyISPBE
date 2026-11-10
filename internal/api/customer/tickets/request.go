package tickets

// CreateCustomerTicketRequest is the payload for POST /api/customer/tickets
type CreateCustomerTicketRequest struct {
	Title       string `json:"title" validate:"required,min=5,max=150"`
	Description string `json:"description" validate:"required,min=10"`
	// TroubleTypeID is one of the ids from GET /api/customer/tickets/trouble-types
	TroubleTypeID string `json:"trouble_type_id"`
	// ImageFilename is the file name returned by POST /api/customer/tickets/upload
	ImageFilename string `json:"image_filename"`
}

// CustomerTicketItem is one row of the complaint history list.
type CustomerTicketItem struct {
	ID              uint64  `json:"id"`
	TicketNumber    string  `json:"ticket_number"`
	Title           string  `json:"title"`
	Description     string  `json:"description,omitempty"`
	TroubleTypeName string  `json:"trouble_type_name"`
	Status          string  `json:"status"`        // raw DB status
	Stage           string  `json:"stage"`         // received | scheduled | handling | done | closed
	StageLabel      string  `json:"stage_label"`   // Tiket Diterima | Teknisi Dijadwalkan | ...
	StageStep       int     `json:"stage_step"`    // 1..4 for the progress tracker
	ImageURL        string  `json:"image_url,omitempty"`
	TechnicianNote  string  `json:"technician_note,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
}

// CustomerTicketDetail is the payload for the tracking detail view.
type CustomerTicketDetail struct {
	CustomerTicketItem
	Timeline []CustomerTicketTimelineEntry `json:"timeline"`
}

// CustomerTicketTimelineEntry is one step shown on the tracking timeline.
type CustomerTicketTimelineEntry struct {
	Stage      string `json:"stage"`
	StageLabel string `json:"stage_label"`
	Reached    bool   `json:"reached"`
	At         string `json:"at,omitempty"`
	Note       string `json:"note,omitempty"`
}
