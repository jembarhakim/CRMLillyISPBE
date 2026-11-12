package customerinstallation

import (
	"database/sql"
	"fmt"
	"log"
	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/services"
	"strings"
	"time"

	"gorm.io/gorm"
)

type AdminInstallationReportRepositoryInterface interface {
	FindCompleteInstallationReportRepository(installationId string) (entities.CustomerInstallation, error)
	FindCompleteInstallationReportWithTechnicianPhotosRepository(installationId string) (CompleteInstallationReportWithTechnicianPhotosResponse, error)
	FindCompleteInstallationReportByViewRepository(installationId string) (InstallationReportCompleteResponse, error)
	FindAllCompleteInstallationReportsRepository() ([]InstallationReportCompleteResponse, error)
	FindInstallationSummaryPerCustomerRepository() ([]InstallationSummaryResponse, error)
	FindInstallationAssetReportRepository(installationId string) (InstallationAssetReportResponse, error)
	FindInstallationTechnicianReportRepository() ([]InstallationTechnicianReportResponse, error)
	FindInstallationTechnicianTeamRepository(installationId string) ([]InstallationTechnicianTeamResponse, error)
	CreateCompleteInstallationReportRepository(request CreateCompleteInstallationReportRequest) (entities.CustomerInstallation, error)
	UpdateCompleteInstallationReportRepository(installationId string, request UpdateCompleteInstallationReportRequest) (entities.CustomerInstallation, error)
	UpdateInstallationCodeName(installationId string, codeName string) error
	DeleteInstallationReportRepository(installationId string) error
}

type AdminInstallationReportRepositoryStruct struct {
	db *gorm.DB
}

func NewAdminInstallationReportRepository(db *gorm.DB) AdminInstallationReportRepositoryStruct {
	return AdminInstallationReportRepositoryStruct{db}
}

// FindCompleteInstallationReportRepository - Get complete installation report with all related data
func (r AdminInstallationReportRepositoryStruct) FindCompleteInstallationReportRepository(installationId string) (entities.CustomerInstallation, error) {
	log.Printf("DEBUG: Finding installation report with ID: %s", installationId)

	var installation entities.CustomerInstallation

	err := r.db.Preload("Customer", "deleted_at IS NULL").
		Preload("Technician").
		Preload("Images").
		Preload("AssetTransactions").
		Preload("NetworkDevices").
		Preload("NetworkDevices.Assets").
		Preload("NetworkDevices.Product").
		Preload("CustomerServices").
		Preload("InstallationTechnicians").
		Preload("InstallationTechnicians.Technician").
		Where("id = ?", installationId).
		First(&installation).Error

	if err != nil {
		log.Printf("ERROR: Failed to find installation report: %v", err)
		return installation, err
	}

	// Log technician photos data from Images relationship
	log.Printf("DEBUG: Found installation report - ID: %s", installation.ID)
	log.Printf("DEBUG: Images relationship count: %d", len(installation.Images))

	// Log each image in the relationship
	for i, img := range installation.Images {
		log.Printf("DEBUG: Image %d - ID: %s, File: %s, FullPath: %s, ArchiveInstallationId: %s", i+1, img.ID, img.File, img.FullPath, img.ArchiveInstallationId)
	}

	return installation, err
}

// FindCompleteInstallationReportWithTechnicianPhotosRepository - Get complete installation report with computed technician photos
func (r AdminInstallationReportRepositoryStruct) FindCompleteInstallationReportWithTechnicianPhotosRepository(installationId string) (CompleteInstallationReportWithTechnicianPhotosResponse, error) {
	log.Printf("DEBUG: Finding installation report with technician photos for ID: %s", installationId)

	var installation entities.CustomerInstallation

	err := r.db.Preload("Customer", "deleted_at IS NULL").
		Preload("Technician").
		Preload("Images").
		Preload("AssetTransactions").
		Preload("NetworkDevices").
		Preload("NetworkDevices.Assets").
		Preload("NetworkDevices.Product").
		Preload("CustomerServices").
		Preload("InstallationTechnicians").
		Preload("InstallationTechnicians.Technician").
		Where("id = ?", installationId).
		First(&installation).Error

	if err != nil {
		log.Printf("ERROR: Failed to find installation report: %v", err)
		return CompleteInstallationReportWithTechnicianPhotosResponse{}, err
	}

	// Also fetch from the database view to get PSB computed fields
	var viewData InstallationReportCompleteResponse
	err = r.db.Table("installation_report_complete").
		Where("installation_id = ?", installationId).
		First(&viewData).Error

	if err != nil {
		log.Printf("WARNING: Failed to fetch from installation_report_complete view: %v", err)
		// Continue without view data - we'll calculate PSB fields manually
	}

	// Create response with computed technician photos
	response := CompleteInstallationReportWithTechnicianPhotosResponse{
		CustomerInstallation: installation,
	}

	// Populate computed fields from relationships for frontend compatibility
	if installation.Customer != nil {
		response.CustomerName = installation.Customer.Name
		response.CustomerAddress = installation.Customer.Address
		response.CustomerPhone = installation.Customer.Phone
		response.TglPermintaanPsb = installation.Customer.ServiceRequestDate
	}

	// Use PSB fields from view data if available, otherwise calculate manually
	if viewData.TglPermintaanPsb != nil {
		response.TglPermintaanPsb = viewData.TglPermintaanPsb.Format("2006-01-02")
	}
	if viewData.DurasiPsb != nil {
		response.DurasiPsb = viewData.DurasiPsb
	}
	if viewData.StatusPsb != "" {
		response.StatusPsb = viewData.StatusPsb
	}

	if installation.Technician != nil {
		response.TechnicianName = installation.Technician.Name
		response.TechnicianPhone = installation.Technician.Phone
	}

	// Populate network device information from the first network device
	if len(installation.NetworkDevices) > 0 {
		networkDevice := installation.NetworkDevices[0]
		response.NetworkDeviceId = networkDevice.ID
		if networkDevice.SwitchID != nil {
			response.SwitchId = *networkDevice.SwitchID
		}
		if networkDevice.PortNumber != nil {
			response.PortNumber = *networkDevice.PortNumber
		}
		if networkDevice.RemotePort != nil {
			response.RemotePort = *networkDevice.RemotePort
		}
		if networkDevice.EthPort != nil {
			response.EthPort = *networkDevice.EthPort
		}
		if networkDevice.MacAddress != nil {
			response.MacAddress = *networkDevice.MacAddress
		}
		if networkDevice.IPStatic != nil {
			response.IPStatic = *networkDevice.IPStatic
		}
		response.KepemilikanPerangkat = networkDevice.KepemilikanPerangkat

		// Get router information from assets relationship
		if networkDevice.Assets != nil {
			response.RouterBrand = networkDevice.Assets.Brand
			response.RouterType = networkDevice.Assets.Type
			response.RouterModel = networkDevice.Assets.Model
			response.RouterSerial = networkDevice.Assets.SerialNumber
		}

		// Get product information from network device
		if networkDevice.Product != nil {
			response.ProductId = networkDevice.Product.ID
			response.ProductName = networkDevice.Product.Name
			response.ProductDescription = networkDevice.Product.Description
			response.ProductPrice = float64(networkDevice.Product.Price)
			response.ProductDownloadSpeedMbps = networkDevice.Product.DownloadSpeedMbps
			response.ProductUploadSpeedMbps = networkDevice.Product.UploadSpeedMbps
		}
	}

	// Populate customer service information from the first customer service
	if len(installation.CustomerServices) > 0 {
		customerService := installation.CustomerServices[0]
		response.CustomerServiceId = customerService.ID
		if customerService.UserLogin != nil {
			response.UserLogin = *customerService.UserLogin
		}
		if customerService.Password != nil {
			response.Password = *customerService.Password
		}
		response.UserStatus = customerService.UserStatus
		if customerService.InstallationNotes != nil {
			response.ServiceNotes = *customerService.InstallationNotes
		}
		if customerService.EndPortType != nil {
			response.EndPortType = *customerService.EndPortType
		}
	}

	// Populate cable information from the first customer service (cable data is now stored in customer_services)
	if len(installation.CustomerServices) > 0 {
		service := installation.CustomerServices[0]
		// Cable ID is no longer separate, cable data is stored directly in customer_services
		// Cable name is not stored separately anymore
		if service.CableType != nil {
			response.CableType = *service.CableType
		}
		if service.CableLength != nil {
			response.CableLength = *service.CableLength
		}
		// Cable status is not tracked separately anymore
	}

	// Populate installation team information
	if len(installation.InstallationTechnicians) > 0 {
		primaryTechnician := installation.InstallationTechnicians[0]
		for _, tech := range installation.InstallationTechnicians {
			if tech.IsPrimary {
				primaryTechnician = tech
				break
			}
		}
		if primaryTechnician.Technician != nil {
			response.InstallationTeamName = primaryTechnician.Technician.Name
			response.InstallationTeamPhone = primaryTechnician.Technician.Phone
		}
	}

	// Set additional computed fields
	response.InstallationId = installation.ID
	response.InstallationStatus = installation.Status
	response.InstallationNotes = installation.Notes
	response.InstallationCreatedAt = installation.CreatedAt
	response.InstallationUpdatedAt = installation.UpdatedAt

	// Calculate PSB duration and status manually only if view data is not available
	if viewData.DurasiPsb == nil && installation.Customer != nil && installation.Customer.ServiceRequestDate != "" && installation.InstallationCompletedAt != nil {
		// Parse service request date
		serviceRequestDate, err := time.Parse("2006-01-02", installation.Customer.ServiceRequestDate)
		if err == nil {
			// Calculate duration in days
			duration := int(installation.InstallationCompletedAt.Sub(serviceRequestDate).Hours() / 24)
			response.DurasiPsb = &duration

			// Determine PSB status based on SLA (≤3 days = Tepat Waktu)
			if duration <= 3 {
				response.StatusPsb = "Tepat Waktu"
			} else {
				response.StatusPsb = "Terlambat"
			}
		}
	}

	// Debug: Log all populated fields
	log.Printf("=== DETAIL CUSTOMER INSTALLATION DEBUG ===")
	log.Printf("Installation ID: %s", response.InstallationId)
	log.Printf("Customer Info - Name: %s, Address: %s, Phone: %s", response.CustomerName, response.CustomerAddress, response.CustomerPhone)
	log.Printf("Technician Info - Name: %s, Phone: %s", response.TechnicianName, response.TechnicianPhone)
	log.Printf("Installation Status: %s, Notes: %s", response.InstallationStatus, response.InstallationNotes)
	log.Printf("PSB Info - Request Date: %s, Duration: %v, Status: %s", response.TglPermintaanPsb, response.DurasiPsb, response.StatusPsb)
	if viewData.DurasiPsb != nil {
		log.Printf("PSB data source: Database view (installation_report_complete)")
	} else {
		log.Printf("PSB data source: Manual calculation")
	}
	log.Printf("Network Device Info - ID: %s, Switch: %s, Port: %s, MAC: %s, IP: %s, Ownership: %s",
		response.NetworkDeviceId, response.SwitchId, response.PortNumber, response.MacAddress, response.IPStatic, response.KepemilikanPerangkat)
	log.Printf("Router Info - Brand: %s, Type: %s, Model: %s, Serial: %s",
		response.RouterBrand, response.RouterType, response.RouterModel, response.RouterSerial)
	log.Printf("Product Info - ID: %s, Name: %s, Description: %s, Price: %.2f",
		response.ProductId, response.ProductName, response.ProductDescription, response.ProductPrice)
	log.Printf("Customer Service Info - ID: %s, User: %s, Status: %s",
		response.CustomerServiceId, response.UserLogin, response.UserStatus)
	log.Printf("Cable Info - Type: %s, Length: %.2f",
		response.CableType, response.CableLength)
	log.Printf("Installation Team - Name: %s, Phone: %s", response.InstallationTeamName, response.InstallationTeamPhone)
	log.Printf("Images Count: %d", len(installation.Images))
	for i, img := range installation.Images {
		log.Printf("  Image %d: ID=%s, File=%s, ArchiveInstallationId=%s", i+1, img.ID, img.File, img.ArchiveInstallationId)
	}
	log.Printf("=== END DETAIL CUSTOMER INSTALLATION DEBUG ===")

	return response, nil
}

// FindCompleteInstallationReportByViewRepository - Get complete installation report using database view
func (r AdminInstallationReportRepositoryStruct) FindCompleteInstallationReportByViewRepository(installationId string) (InstallationReportCompleteResponse, error) {
	var report InstallationReportCompleteResponse

	query := `
		SELECT
			installation_id,
			customer_id,
			customer_name,
			customer_address,
			customer_phone,
			tgl_permintaan_psb,
			technician_id,
			technician_name,
			technician_phone,
			installation_status,
			installation_type,
			installation_notes,
			on_air_date,
			trial_end_date,
			service_ready_date,
			installation_completed_at,
			durasi_psb,
			status_psb,
			document_type,
			document_photo,
			network_device_id,
			switch_id,
			port_number,
			remote_port,
			eth_port,
			mac_address,
			ip_static,
			kepemilikan_perangkat,
			router_brand,
			router_type,
			router_model,
			router_serial,
			product_id,
			product_name,
			product_description,
			product_price,
			download_speed_mbps,
			upload_speed_mbps,
			customer_service_id,
			user_login,
			password,
			user_status,
			service_notes,
			cable_type,
			cable_length,
			end_port_type,
			installation_created_at,
			installation_updated_at
		FROM installation_report_complete
		WHERE installation_id = ?
		LIMIT 1
	`

	err := r.db.Raw(query, installationId).Scan(&report).Error
	if err != nil {
		log.Printf("Error querying installation report by view: %v", err)
		return report, fmt.Errorf("failed to retrieve installation report: %w", err)
	}

	return report, nil
}

// FindAllCompleteInstallationReportsRepository - Get all complete installation reports using database view
func (r AdminInstallationReportRepositoryStruct) FindAllCompleteInstallationReportsRepository() ([]InstallationReportCompleteResponse, error) {
	var reports []InstallationReportCompleteResponse

	// Use the installation_report_complete view which includes network device, customer service, and product data
	query := `
		SELECT
			installation_id,
			customer_id,
			customer_name,
			customer_address,
			customer_phone,
			tgl_permintaan_psb,
			technician_id,
			technician_name,
			technician_phone,
			installation_status,
			installation_type,
			installation_notes,
			on_air_date,
			trial_end_date,
			service_ready_date,
			installation_completed_at,
			durasi_psb,
			status_psb,
			document_type,
			document_photo,
			network_device_id,
			switch_id,
			port_number,
			remote_port,
			eth_port,
			mac_address,
			ip_static,
			kepemilikan_perangkat,
			router_brand,
			router_type,
			router_model,
			router_serial,
			product_id,
			product_name,
			product_description,
			product_price,
			download_speed_mbps,
			upload_speed_mbps,
			customer_service_id,
			user_login,
			password,
			user_status,
			service_notes,
			cable_type,
			cable_length,
			end_port_type,
			installation_created_at,
			installation_updated_at
		FROM installation_report_complete
		ORDER BY installation_created_at DESC
	`

	err := r.db.Raw(query).Scan(&reports).Error
	return reports, err
}

// FindInstallationSummaryPerCustomerRepository - Get installation summary grouped by customer
func (r AdminInstallationReportRepositoryStruct) FindInstallationSummaryPerCustomerRepository() ([]InstallationSummaryResponse, error) {
	var summaries []InstallationSummaryResponse

	query := `
        SELECT 
            c.id as customer_id,
            c.name as customer_name,
            c.address as customer_address,
            c.phone as customer_phone,
            c.service_request_date as tgl_permintaan_psb,
            c.deleted_at,  -- Added so frontend can detect dismantled customers
            COUNT(ci.id) as total_installations,
            COUNT(CASE WHEN ci.status = 'completed' THEN 1 END) as completed_installations,
            COUNT(CASE WHEN ci.status = 'pending' THEN 1 END) as pending_installations,
            COUNT(CASE WHEN ci.status = 'in_progress' THEN 1 END) as in_progress_installations,
            MAX(ci.on_air_date) as latest_on_air_date,
            MAX(ci.installation_completed_at) as latest_completion_date,
            AVG(CASE 
                WHEN c.service_request_date IS NOT NULL AND ci.installation_completed_at IS NOT NULL
                THEN DATEDIFF(ci.installation_completed_at, c.service_request_date)
                ELSE NULL
            END) as avg_durasi_psb,
            COUNT(CASE 
                WHEN c.service_request_date IS NOT NULL AND ci.installation_completed_at IS NOT NULL
                AND DATEDIFF(ci.installation_completed_at, c.service_request_date) <= 3
                THEN 1
            END) as tepat_waktu_count,
            COUNT(CASE 
                WHEN c.service_request_date IS NOT NULL AND ci.installation_completed_at IS NOT NULL
                AND DATEDIFF(ci.installation_completed_at, c.service_request_date) > 3
                THEN 1
            END) as terlambat_count
        FROM customer c
        -- Join installations but ignore deleted installations
        LEFT JOIN customer_installations ci ON c.id = ci.customer_id AND ci.deleted_at IS NULL
        
        -- REMOVED: WHERE c.deleted_at IS NULL (To allow fetching dismantled customers)
        
        GROUP BY c.id, c.name, c.address, c.phone, c.service_request_date, c.deleted_at
        ORDER BY total_installations DESC
    `

	err := r.db.Raw(query).Scan(&summaries).Error
	return summaries, err
}

// FindInstallationTechnicianTeamRepository - Get technician team for specific installation
func (r AdminInstallationReportRepositoryStruct) FindInstallationTechnicianTeamRepository(installationId string) ([]InstallationTechnicianTeamResponse, error) {
	var teams []InstallationTechnicianTeamResponse

	// I am adding a placeholder query here because this function was missing.
	// You should replace this query with your logic to fetch the team members.
	query := `
        SELECT 
            it.id,
            it.customer_installation_id,
            it.technician_id,
            u.name as technician_name,
            u.phone as technician_phone,
            u.email as technician_email,
            it.role,
            it.is_primary,
            it.notes,
            it.createdAt as created_at,
            it.updatedAt as updated_at
        FROM installation_report_technicians it
        JOIN users u ON it.technician_id = u.id
        WHERE it.customer_installation_id = ?
    `

	err := r.db.Raw(query, installationId).Scan(&teams).Error
	return teams, err
}

// FindInstallationAssetReportRepository - Get asset report for specific installation
func (r AdminInstallationReportRepositoryStruct) FindInstallationAssetReportRepository(installationId string) (InstallationAssetReportResponse, error) {
	var report InstallationAssetReportResponse

	query := `
		SELECT 
			ci.id as installation_id,
			c.name as customer_name,
			ci.installation_type,
			ci.status as installation_status,
			ci.on_air_date,
			ci.installation_completed_at,
			COALESCE(SUM(CASE WHEN at.transaction_type = 'out' THEN at.quantity ELSE 0 END), 0) as total_quantity_out,
			COALESCE(SUM(CASE WHEN at.transaction_type = 'in' THEN at.quantity ELSE 0 END), 0) as total_quantity_in,
			GROUP_CONCAT(DISTINCT CASE WHEN at.transaction_type = 'out' THEN CONCAT(a.brand, ' ', a.model, ' (', at.quantity, ' pcs)') END SEPARATOR ', ') as assets_out_details,
			GROUP_CONCAT(DISTINCT CASE WHEN at.transaction_type = 'in' THEN CONCAT(a.brand, ' ', a.model, ' (', at.quantity, ' pcs)') END SEPARATOR ', ') as assets_in_details
		FROM customer_installations ci
		LEFT JOIN customer c ON ci.customer_id = c.id AND c.deleted_at IS NULL
		LEFT JOIN asset_transactions at ON ci.id = at.customer_installation_id
		LEFT JOIN assets a ON at.asset_id = a.id
		WHERE ci.id = ?
		GROUP BY ci.id, c.name, ci.installation_type, ci.status, ci.on_air_date, ci.installation_completed_at
	`

	err := r.db.Raw(query, installationId).Scan(&report).Error
	return report, err
}

// FindInstallationTechnicianReportRepository - Get technician performance report
func (r AdminInstallationReportRepositoryStruct) FindInstallationTechnicianReportRepository() ([]InstallationTechnicianReportResponse, error) {
	var reports []InstallationTechnicianReportResponse

	query := `
		SELECT 
			u.id as technician_id,
			u.name as technician_name,
			u.phone as technician_phone,
			COUNT(ci.id) as total_installations,
			COUNT(CASE WHEN ci.status = 'completed' THEN 1 END) as completed_installations,
			COUNT(CASE WHEN ci.status = 'pending' THEN 1 END) as pending_installations,
			COUNT(CASE WHEN ci.status = 'in_progress' THEN 1 END) as in_progress_installations,
			AVG(TO_DAYS(ci.installation_completed_at) - TO_DAYS(ci.createdAt)) as avg_completion_days,
			MAX(ci.installation_completed_at) as latest_completion_date
		FROM users u
		LEFT JOIN customer_installations ci ON u.id = ci.technician_id
		WHERE u.role_id = (SELECT id FROM roles WHERE name = 'TECHNICIAN')
		GROUP BY u.id, u.name, u.phone
		ORDER BY total_installations DESC
	`

	err := r.db.Raw(query).Scan(&reports).Error
	return reports, err
}

// CreateCompleteInstallationReportRepository - Create complete installation report with all related data
func (r AdminInstallationReportRepositoryStruct) CreateCompleteInstallationReportRepository(request CreateCompleteInstallationReportRequest) (entities.CustomerInstallation, error) {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Parse dates
	var onAirDate, trialEndDate, serviceReadyDate, installationCompletedAt *time.Time

	if request.OnAirDate != "" {
		if parsed, err := time.Parse("2006-01-02", request.OnAirDate); err == nil {
			onAirDate = &parsed
		}
	}

	if request.TrialEndDate != "" {
		if parsed, err := time.Parse("2006-01-02", request.TrialEndDate); err == nil {
			trialEndDate = &parsed
		}
	}

	if request.ServiceReadyDate != "" {
		if parsed, err := time.Parse("2006-01-02", request.ServiceReadyDate); err == nil {
			serviceReadyDate = &parsed
		}
	}

	if request.InstallationCompletedAt != "" {
		// Try different date formats
		formats := []string{
			"2006-01-02T15:04",    // datetime-local format from frontend
			"2006-01-02 15:04:05", // standard datetime format
			"2006-01-02T15:04:05", // ISO format
		}

		for _, format := range formats {
			if parsed, err := time.Parse(format, request.InstallationCompletedAt); err == nil {
				installationCompletedAt = &parsed
				break
			}
		}

		// Log for debugging
		if installationCompletedAt != nil {
			fmt.Printf("Successfully parsed installation_completed_at: %s -> %v\n", request.InstallationCompletedAt, *installationCompletedAt)
		} else {
			fmt.Printf("Failed to parse installation_completed_at: %s\n", request.InstallationCompletedAt)
		}
	}

	// Log document photo for debugging
	fmt.Printf("Document Photo from request: '%s'\n", request.DocumentPhoto)
	fmt.Printf("Document Type from request: '%s'\n", request.DocumentType)

	// Handle terminal fields
	var isTerminal *string
	if request.IsTerminal != "" {
		isTerminal = &request.IsTerminal
	}

	var terminalCustomerInstallationId *string
	if request.TerminalCustomerInstallationId != "" {
		terminalCustomerInstallationId = &request.TerminalCustomerInstallationId
	}

	// Handle location fields
	latitude := request.Latitude
	longitude := request.Longitude

	// Create main installation record
	installation := entities.CustomerInstallation{
		ID:                             "",
		CustomerID:                     &request.CustomerId,
		TechnicianID:                   &request.TechnicianId,
		Status:                         request.Status,
		Notes:                          request.Notes,
		DocumentType:                   &request.DocumentType,
		DocumentPhoto:                  &request.DocumentPhoto,
		InstallationType:               request.InstallationType,
		InstallationCompletedAt:        installationCompletedAt,
		TrialEndDate:                   trialEndDate,
		ServiceReadyDate:               serviceReadyDate,
		OnAirDate:                      onAirDate,
		IsTerminal:                     isTerminal,
		TerminalCustomerInstallationID: terminalCustomerInstallationId,
		Latitude:                       latitude,
		Longitude:                      longitude,
	}

	if err := tx.Create(&installation).Error; err != nil {
		tx.Rollback()
		return installation, err
	}

	// Log what was actually saved to database
	fmt.Printf("Installation created with ID: %s\n", installation.ID)
	if installation.DocumentPhoto != nil {
		// Normalize the document photo path if needed
		normalizedPath := normalizeDocumentPhotoPath(*installation.DocumentPhoto)
		if normalizedPath != *installation.DocumentPhoto {
			// Update the database with the normalized path
			installation.DocumentPhoto = &normalizedPath
			tx.Save(&installation)
			fmt.Printf("Document Photo normalized from '%s' to '%s'\n", *installation.DocumentPhoto, normalizedPath)
		} else {
			fmt.Printf("Document Photo saved to DB: '%s'\n", *installation.DocumentPhoto)
		}
	} else {
		fmt.Printf("Document Photo is NULL in DB\n")
	}
	if installation.DocumentType != nil {
		fmt.Printf("Document Type saved to DB: '%s'\n", *installation.DocumentType)
	} else {
		fmt.Printf("Document Type is NULL in DB\n")
	}

	// Create asset transactions
	for _, assetTx := range request.AssetTransactions {
		var transactionDate time.Time
		if assetTx.TransactionDate != "" {
			if parsed, err := time.Parse("2006-01-02 15:04:05", assetTx.TransactionDate); err == nil {
				transactionDate = parsed
			} else {
				transactionDate = time.Now()
			}
		} else {
			transactionDate = time.Now()
		}

		assetTransaction := entities.AssetTransaction{
			ID:                     "",
			CustomerInstallationID: installation.ID,
			AssetID:                assetTx.AssetId,
			TransactionType:        assetTx.TransactionType,
			Quantity:               assetTx.Quantity,
			Notes:                  &assetTx.Notes,
			TransactionDate:        transactionDate,
			CreatedBy:              request.TechnicianId, // Assuming technician creates the transaction
		}

		if err := tx.Create(&assetTransaction).Error; err != nil {
			tx.Rollback()
			return installation, err
		}
	}

	// Create network devices
	for _, device := range request.NetworkDevices {
		// Debug: Log the product ID being set
		log.Printf("🔍 DEBUG: Setting ProductID for network device: %s", device.ProductID)

		networkDevice := entities.NetworkDevice{
			ID:         "",
			CustomerID: request.CustomerId,
			AssetsID: sql.NullString{
				String: func() string {
					if device.AssetsID != "" {
						return device.AssetsID
					}
					return ""
				}(),
				Valid: device.AssetsID != "",
			},
			AssetItemID:            &device.AssetItemID,
			CustomerInstallationID: &installation.ID,
			SwitchID:               &device.SwitchID,
			PortNumber:             &device.PortNumber,
			RemotePort:             &device.RemotePort,
			EthPort:                &device.EthPort,
			MacAddress:             &device.MacAddress,
			IPStatic:               &device.IPStatic,
			KepemilikanPerangkat:   device.KepemilikanPerangkat,
			ProductID:              &device.ProductID,
		}

		if err := tx.Create(&networkDevice).Error; err != nil {
			tx.Rollback()
			return installation, err
		}

		// Handle asset item tracking - update status from 'in_stock' to 'in_use'
		if device.AssetItemID != "" {
			log.Printf("🔧 DEBUG: Updating asset item %s status from 'in_stock' to 'in_use'", device.AssetItemID)

			// Check if asset item exists and is in stock
			var assetItem entities.AssetItem
			if err := tx.Where("id = ? AND status = 'in_stock'", device.AssetItemID).First(&assetItem).Error; err != nil {
				log.Printf("❌ ERROR: Asset item %s not found or not in stock: %v", device.AssetItemID, err)
				tx.Rollback()
				return installation, fmt.Errorf("asset item %s not found or not available (status must be 'in_stock')", device.AssetItemID)
			}

			// Update asset item status to 'in_use'
			if err := tx.Model(&entities.AssetItem{}).
				Where("id = ?", device.AssetItemID).
				Update("status", "in_use").Error; err != nil {
				log.Printf("❌ ERROR: Failed to update asset item %s status: %v", device.AssetItemID, err)
				tx.Rollback()
				return installation, fmt.Errorf("failed to update asset item status: %v", err)
			}

			log.Printf("✅ SUCCESS: Asset item %s status updated to 'in_use'", device.AssetItemID)

			// Create asset transaction record for tracking
			assetTransaction := entities.AssetTransaction{
				ID:                     "",
				CustomerInstallationID: installation.ID,
				AssetItemID:            &device.AssetItemID,
				AssetID:                assetItem.AssetID,
				TransactionType:        "out", // Asset going out for installation
				Quantity:               1,
				Notes:                  func() *string { s := fmt.Sprintf("Asset assigned to installation %s", installation.ID); return &s }(),
				TransactionDate:        time.Now(),
				CreatedBy:              request.TechnicianId,
			}

			if err := tx.Create(&assetTransaction).Error; err != nil {
				log.Printf("⚠️  WARNING: Failed to create asset transaction record: %v", err)
				// Don't rollback - transaction record is not critical
			} else {
				log.Printf("✅ SUCCESS: Asset transaction record created for item %s", device.AssetItemID)
			}
		}
	}

	// Create customer services
	for _, service := range request.CustomerServices {
		// Use service_ready_date from the main installation as service_activation_date
		// This eliminates the need for a separate service_activation_date field

		// Use the provided DeviceID and CableID values (or set to nil if empty)
		var deviceID *string
		if service.DeviceID != "" {
			deviceID = &service.DeviceID
		}

		// CableID no longer used - cable data is stored directly in customer_services

		customerService := entities.CustomerService{
			ID:                     "",
			CustomerID:             request.CustomerId,
			CustomerInstallationID: &installation.ID,
			DeviceID:               deviceID,
			CableType:              &service.CableType,
			CableLength:            &service.CableLength,
			EndPortType:            &service.EndPortType,
			UserLogin:              &service.UserLogin,
			Password:               &service.Password,
			UserStatus:             service.UserStatus,
			InstallationNotes:      &service.InstallationNotes,
		}

		if err := tx.Create(&customerService).Error; err != nil {
			tx.Rollback()
			return installation, err
		}
	}

	// Note: Cable table has been merged with customer_services table
	// Cable data is now stored in customer_services table (cable_type, cable_length, end_port_type)

	// Update images with installation ID
	if len(request.ImageIds) > 0 {
		if err := tx.Model(&entities.Image{}).
			Where("id IN ?", request.ImageIds).
			Update("archive_installation_id", installation.ID).Error; err != nil {
			tx.Rollback()
			return installation, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return installation, err
	}

	return installation, nil
}

// UpdateCompleteInstallationReportRepository - Update complete installation report with all related data
func (r AdminInstallationReportRepositoryStruct) UpdateCompleteInstallationReportRepository(installationId string, request UpdateCompleteInstallationReportRequest) (entities.CustomerInstallation, error) {
	var installation entities.CustomerInstallation

	// Start transaction
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find existing installation
	if err := tx.Preload("Customer", "deleted_at IS NULL").Preload("Technician").Preload("Images").First(&installation, "id = ?", installationId).Error; err != nil {
		tx.Rollback()
		return installation, err
	}

	// Update basic installation data
	installation.CustomerID = &request.CustomerId
	installation.TechnicianID = &request.TechnicianId
	installation.Status = request.Status
	installation.Notes = request.Notes
	if request.DocumentType != "" {
		installation.DocumentType = &request.DocumentType
	}
	if request.DocumentPhoto != "" {
		installation.DocumentPhoto = &request.DocumentPhoto
	}
	installation.InstallationType = request.InstallationType

	// Handle technician photos - they are already uploaded via file upload API
	// and linked to this installation via archive_installation_id
	// No need to delete/recreate them here
	log.Printf("DEBUG: Technician photos are already uploaded and linked via archive_installation_id")
	log.Printf("DEBUG: Skipping technician photo processing in update - photos persist in images table")

	// Note: Technician photos are now stored in images table with archive_installation_id
	// No need to store metadata in customer_installations table

	// Parse dates
	if request.OnAirDate != "" {
		if onAirDate, err := time.Parse("2006-01-02", request.OnAirDate); err == nil {
			installation.OnAirDate = &onAirDate
		}
	}
	if request.TrialEndDate != "" {
		if trialEndDate, err := time.Parse("2006-01-02", request.TrialEndDate); err == nil {
			installation.TrialEndDate = &trialEndDate
		}
	}
	if request.ServiceReadyDate != "" {
		if serviceReadyDate, err := time.Parse("2006-01-02", request.ServiceReadyDate); err == nil {
			installation.ServiceReadyDate = &serviceReadyDate
		}
	}
	if request.InstallationCompletedAt != "" {
		if completedAt, err := time.Parse("2006-01-02T15:04:05", request.InstallationCompletedAt); err == nil {
			installation.InstallationCompletedAt = &completedAt
		}
	}

	// Update terminal fields
	if request.IsTerminal != "" {
		installation.IsTerminal = &request.IsTerminal
	}
	if request.TerminalCustomerInstallationId != "" {
		installation.TerminalCustomerInstallationID = &request.TerminalCustomerInstallationId
	} else {
		installation.TerminalCustomerInstallationID = nil
	}

	// Update location fields
	installation.Latitude = request.Latitude
	installation.Longitude = request.Longitude

	// Save installation
	log.Printf("DEBUG: Saving installation with ID: %s", installation.ID)

	if err := tx.Save(&installation).Error; err != nil {
		log.Printf("ERROR: Failed to save installation: %v", err)
		tx.Rollback()
		return installation, err
	}

	log.Printf("DEBUG: Installation saved successfully")

	// Debug incoming network devices and enforce presence of asset_item_id when MAC is provided
	for i, d := range request.NetworkDevices {
		log.Printf("DEBUG: Incoming device[%d] AssetsID=%s AssetItemID=%s MAC=%s", i, d.AssetsID, d.AssetItemID, d.MacAddress)
		if d.MacAddress != "" && d.AssetItemID == "" {
			tx.Rollback()
			return installation, fmt.Errorf("device %d with MAC %s is missing asset_item_id", i, d.MacAddress)
		}
	}

	// Find existing network devices to release asset items
	var existingDevices []entities.NetworkDevice
	if err := tx.Where("customer_installation_id = ?", installation.ID).Find(&existingDevices).Error; err != nil {
		tx.Rollback()
		return installation, err
	}

	// Release asset items (set status to in_stock)
	for _, device := range existingDevices {
		if device.AssetItemID != nil && *device.AssetItemID != "" {
			if err := tx.Model(&entities.AssetItem{}).
				Where("id = ?", *device.AssetItemID).
				Update("status", "in_stock").Error; err != nil {
				log.Printf("WARNING: Failed to release asset item %s: %v", *device.AssetItemID, err)
				// Continue execution as this is non-critical
			} else {
				log.Printf("DEBUG: Released asset item %s (status set to in_stock)", *device.AssetItemID)
			}
		}
	}

	// Delete existing related data
	if err := tx.Where("customer_installation_id = ?", installation.ID).Delete(&entities.NetworkDevice{}).Error; err != nil {
		tx.Rollback()
		return installation, err
	}
	if err := tx.Where("customer_installation_id = ?", installation.ID).Delete(&entities.CustomerService{}).Error; err != nil {
		tx.Rollback()
		return installation, err
	}

	// Create network devices and store their new IDs
	newNetworkDeviceIDs := make([]string, 0, len(request.NetworkDevices))

	for _, device := range request.NetworkDevices {
		deviceEntity := entities.NetworkDevice{
			ID:                     "",
			CustomerID:             request.CustomerId,
			AssetsID:               sql.NullString{String: device.AssetsID, Valid: device.AssetsID != ""},
			AssetItemID:            nil,
			SwitchID:               &device.SwitchID,
			PortNumber:             &device.PortNumber,
			RemotePort:             &device.RemotePort,
			EthPort:                &device.EthPort,
			MacAddress:             &device.MacAddress,
			IPStatic:               &device.IPStatic,
			KepemilikanPerangkat:   device.KepemilikanPerangkat,
			ProductID:              &device.ProductID,
			CustomerInstallationID: &installation.ID,
		}

		// Persist selected asset item ID so it can be released on future updates
		if device.AssetItemID != "" {
			assetItemID := device.AssetItemID
			deviceEntity.AssetItemID = &assetItemID
		}

		if err := tx.Create(&deviceEntity).Error; err != nil {
			tx.Rollback()
			return installation, err
		}

		// Store the newly created network device ID
		newNetworkDeviceIDs = append(newNetworkDeviceIDs, deviceEntity.ID)
		log.Printf("DEBUG: Created network device with new ID: %s", deviceEntity.ID)

		// Handle asset item tracking if asset_item_id is provided
		if device.AssetItemID != "" {
			// Update asset item status to 'in_use'
			if err := tx.Model(&entities.AssetItem{}).
				Where("id = ?", device.AssetItemID).
				Update("status", "in_use").Error; err != nil {
				log.Printf("Failed to update asset item status: %v", err)
				// Don't rollback - this is not critical enough to fail the entire update
			}
		}
	}

	// Create customer services
	// Link each customer service to the corresponding newly created network device
	for i, service := range request.CustomerServices {
		// Use the newly created network device ID instead of the old device_id from the request
		// This prevents foreign key constraint errors since old device IDs were deleted
		var deviceID *string
		if i < len(newNetworkDeviceIDs) {
			deviceID = &newNetworkDeviceIDs[i]
			log.Printf("DEBUG: Linking customer service %d to new network device ID: %s", i, *deviceID)
		} else {
			log.Printf("WARNING: Customer service %d has no corresponding network device", i)
		}

		// CableID no longer used - cable data is stored directly in customer_services

		serviceEntity := entities.CustomerService{
			ID:                     "",
			CustomerID:             request.CustomerId,
			DeviceID:               deviceID,
			CableType:              &service.CableType,
			CableLength:            &service.CableLength,
			EndPortType:            &service.EndPortType,
			UserLogin:              &service.UserLogin,
			Password:               &service.Password,
			UserStatus:             service.UserStatus,
			InstallationNotes:      &service.InstallationNotes,
			CustomerInstallationID: &installation.ID,
		}

		if err := tx.Create(&serviceEntity).Error; err != nil {
			log.Printf("ERROR: Failed to create customer service: %v", err)
			tx.Rollback()
			return installation, err
		}
		log.Printf("DEBUG: Created customer service with ID: %s", serviceEntity.ID)
	}

	// Note: Cable table has been merged with customer_services table
	// Cable data is now stored in customer_services table (cable_type, cable_length, end_port_type)

	// Update images with installation ID
	if len(request.ImageIds) > 0 {
		if err := tx.Model(&entities.Image{}).
			Where("id IN ?", request.ImageIds).
			Update("archive_installation_id", installation.ID).Error; err != nil {
			tx.Rollback()
			return installation, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return installation, err
	}

	return installation, nil
}

// DeleteInstallationReportRepository - Perform soft delete and insert into history table
func (r AdminInstallationReportRepositoryStruct) DeleteInstallationReportRepository(installationId string) error {
	// Start transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// First, get the installation with all related data before soft delete
	var installation entities.CustomerInstallation
	if err := tx.Preload("NetworkDevices").Preload("Customer").Preload("Technician").Where("id = ?", installationId).First(&installation).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Perform Mikrotik cleanup (disable configurations instead of deleting)
	if err := r.performMikrotikCleanup(installation); err != nil {
		log.Printf("⚠️ Warning: Mikrotik cleanup failed for installation %s: %v", installationId, err)
		// Continue with soft delete even if Mikrotik cleanup fails
	}

	// Get current timestamp for the soft delete
	now := time.Now()

	// Loop through the NetworkDevices associated with the installation
	for _, networkDevice := range installation.NetworkDevices {
		// Try to locate the asset item by explicit asset_item_id first, fallback to MAC lookup
		var assetItem entities.AssetItem
		var assetItemFound bool

		if networkDevice.AssetItemID != nil && *networkDevice.AssetItemID != "" {
			if err := tx.Unscoped().Where("id = ?", *networkDevice.AssetItemID).First(&assetItem).Error; err == nil {
				assetItemFound = true
			}
		}

		if !assetItemFound && networkDevice.MacAddress != nil && *networkDevice.MacAddress != "" {
			if err := tx.Unscoped().Where("mac_address = ?", *networkDevice.MacAddress).First(&assetItem).Error; err == nil {
				assetItemFound = true
			}
		}

		if !assetItemFound {
			continue
		}

		// Prepare updates for the asset item
		updates := map[string]interface{}{
			"status": "in_stock",
		}

		// Reset MAC address back to the sticker value when available
		if assetItem.MacSticker != nil && *assetItem.MacSticker != "" {
			updates["mac_address"] = *assetItem.MacSticker

			// Keep network_devices table consistent with inventory MAC reset
			if err := tx.Model(&entities.NetworkDevice{}).
				Where("id = ?", networkDevice.ID).
				Update("mac_address", *assetItem.MacSticker).Error; err != nil {
				log.Printf("?? Warning: Failed to reset network device %s MAC to sticker: %v", networkDevice.ID, err)
			}
		}

		if err := tx.Unscoped().Model(&assetItem).Updates(updates).Error; err != nil {
			log.Printf("?? Warning: Failed to reset asset item %s: %v", assetItem.ID, err)
		}
	}

	// Find IP and MAC from network devices
	var oldIP, oldMac *string
	if len(installation.NetworkDevices) > 0 {
		networkDevice := installation.NetworkDevices[0] // Use first network device
		if networkDevice.IPStatic != nil && *networkDevice.IPStatic != "" {
			oldIP = networkDevice.IPStatic
		}
		if networkDevice.MacAddress != nil && *networkDevice.MacAddress != "" {
			oldMac = networkDevice.MacAddress
		}
	}

	// Get asset item ID from network devices
	var assetItemID *string
	if len(installation.NetworkDevices) > 0 && installation.NetworkDevices[0].AssetItemID != nil {
		assetItemID = installation.NetworkDevices[0].AssetItemID
	}

	// Create dismantle trouble ticket first
	dismantleTitle := fmt.Sprintf("Customer Dismantle: %s", installation.Customer.Name)
	dismantleDescription := fmt.Sprintf("Installation report for customer %s has been deleted. Customer has terminated service. Equipment needs to be collected and configurations need to be cleaned up.", installation.Customer.Name)

	// Look up TECHNICIAN role ID dynamically for assignment
	techRoleID, err := r.roleIDByName(tx, string(entities.AssignTech))
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to lookup technician role ID: %v", err)
	}

	typeStr := "7"
	dismantleTicket := entities.TroubleTicket{
		CustomerID:       *installation.CustomerID,
		Type:             &typeStr,
		Title:            dismantleTitle,
		Description:      &dismantleDescription,
		Status:           "unfinished",
		ClassificationID: "dismantle",
		CurrentAssignee:  techRoleID, // Assign to technician
		CustomerNote:     &dismantleDescription,
		AssetItemID:      assetItemID,
		Accumulation:     1,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}

	if err := tx.Create(&dismantleTicket).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create dismantle ticket: %v", err)
	}

	log.Printf("✅ Dissmantle ticket created with ID %d for customer %s", dismantleTicket.ID, installation.Customer.Name)

	// Insert into installation_history table
	historyRecord := entities.InstallationHistory{
		ID:             "", // Will be auto-generated
		InstallationID: installationId,
		OldIP:          oldIP,
		OldMac:         oldMac,
		ChangeReason:   "terminated", // Default reason for deletion
		TicketID:       uint(dismantleTicket.ID),
		CreatedAt:      now,
	}

	if err := tx.Create(&historyRecord).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create installation history record: %v", err)
	}

	log.Printf("✅ Installation history record created for installation %s", installationId)

	// Mark recurring invoices tied to this installation as completed
	if err := tx.Model(&entities.RecurringInvoice{}).
		Where("customer_installation = ?", installationId).
		Update("status", entities.RecurringInvoiceStatusCompleted).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update recurring invoices for installation %s: %v", installationId, err)
	}
	log.Printf("✅ Marked recurring invoices as completed for installation %s", installationId)

	// Keep only critical related records and soft delete main installation

	// Soft delete the installation by setting deleted_at timestamp
	if err := tx.Model(&entities.CustomerInstallation{}).
		Where("id = ?", installationId).
		Update("deleted_at", now).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to soft delete installation: %v", err)
	}

	log.Printf("✅ Installation %s soft deleted (set deleted_at = %v)", installationId, now)

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("✅ Installation deletion completed successfully for %s", installationId)
	return nil
}

// UpdateInstallationCodeName - Update code_name for an installation
func (r AdminInstallationReportRepositoryStruct) UpdateInstallationCodeName(installationId string, codeName string) error {
	return r.db.Model(&entities.CustomerInstallation{}).
		Where("id = ?", installationId).
		Update("code_name", codeName).Error
}

// roleIDByName finds role id by roles.name with fallback between space/underscore variants
func (r *AdminInstallationReportRepositoryStruct) roleIDByName(tx *gorm.DB, name string) (string, error) {
	log.Printf("roleIDByName: Looking for role with name: '%s'", name)
	var row struct{ ID string }
	if err := tx.Table("roles").Select("id").Where("name = ?", name).Scan(&row).Error; err != nil {
		log.Printf("roleIDByName: Error querying role '%s': %v", name, err)
		return "", err
		if alt := strings.Replace(name, " ", "_", -1); alt != name {
			log.Printf("roleIDByName: Not found, trying alternative name: '%s'", alt)
			if err := tx.Table("roles").Select("id").Where("name = ?", alt).Scan(&row).Error; err != nil {
				log.Printf("roleIDByName: Error querying role alt '%s': %v", alt, err)
				return "", err
			}
		}
		err := fmt.Errorf("role '%s' not found (after fallback)", name)
		log.Printf("roleIDByName: %v", err)
		return "", err
	}
	log.Printf("roleIDByName: Found role ID '%s' for name '%s'", row.ID, name)
	return row.ID, nil
}

// performMikrotikCleanup performs Mikrotik RouterOS disable for an installation
// Disables queues, netwatch, IP bindings, schedulers, and scripts based on customer_id and mac_address
func (r AdminInstallationReportRepositoryStruct) performMikrotikCleanup(installation entities.CustomerInstallation) error {
	// Get shared Mikrotik service
	mikrotikService := services.GetSharedMikroTikService()
	if mikrotikService == nil {
		log.Printf("⚠️ No shared Mikrotik service available, skipping disable")
		return nil
	}

	// Create provisioning service for disable
	provisioningService := services.NewMikrotikProvisioningService(r.db, mikrotikService)

	// Perform disable (not dry run)
	return provisioningService.CleanupInstallation(installation, false)
}
