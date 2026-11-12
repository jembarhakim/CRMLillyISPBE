package customerinstallation

import (
	"database/sql"
	"fmt"
	"log"
	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/services"
	"time"

	"gorm.io/gorm"
)

type ReportInstallationRepositoryInterface interface {
	CreateReportInstallationRepository(request CreateReportInstallationRequest, createdByUserID string) (entities.CustomerInstallation, error)
	CreateInstallationTechnicians(tx *gorm.DB, installationID string, technicians []TechnicianAssignment) error
	DeleteInstallation(installationId string) error
	UpdateInstallationCodeName(installationId string, codeName string) error
	GetDB() *gorm.DB
}

type ReportInstallationRepository struct {
	db *gorm.DB
}

func NewReportInstallationRepository(db *gorm.DB) ReportInstallationRepositoryInterface {
	return &ReportInstallationRepository{db: db}
}

// GetDB returns the database connection
func (r *ReportInstallationRepository) GetDB() *gorm.DB {
	return r.db
}

// CreateInstallationTechnicians creates multiple technician assignments for an installation
func (r *ReportInstallationRepository) CreateInstallationTechnicians(
	tx *gorm.DB,
	installationID string,
	technicians []TechnicianAssignment,
) error {
	if len(technicians) == 0 {
		return nil
	}

	// Validate at least one senior exists
	hasSenior := false
	for _, tech := range technicians {
		if tech.Role == "senior" {
			hasSenior = true
			break
		}
	}

	if !hasSenior {
		return fmt.Errorf("at least one senior technician is required")
	}

	// Ensure only one primary technician
	primaryCount := 0
	for i := range technicians {
		if technicians[i].IsPrimary {
			primaryCount++
		}
	}

	// If no primary is set, make the first senior primary
	if primaryCount == 0 {
		for i := range technicians {
			if technicians[i].Role == "senior" {
				technicians[i].IsPrimary = true
				break
			}
		}
	}

	// Create pivot records
	for _, tech := range technicians {
		pivot := entities.InstallationReportTechnician{
			CustomerInstallationID: installationID,
			TechnicianID:           tech.TechnicianID,
			Role:                   tech.Role,
			IsPrimary:              tech.IsPrimary,
			Notes:                  tech.Notes,
		}
		if err := tx.Create(&pivot).Error; err != nil {
			return fmt.Errorf("failed to create technician assignment: %w", err)
		}
	}

	log.Printf("✅ Created %d technician assignments for installation %s", len(technicians), installationID)
	return nil
}

// Helper function to convert empty string to nil
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Helper function to convert float64 to pointer
func stringToFloat(f float64) *float64 {
	return &f
}

func getStringValue(s *string) string {
	if s == nil {
		return "nil"
	}
	return *s
}

// CreateReportInstallationRepository - Create installation report with all related data
func (r *ReportInstallationRepository) CreateReportInstallationRepository(request CreateReportInstallationRequest, createdByUserID string) (entities.CustomerInstallation, error) {
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
			log.Printf("Successfully parsed installation_completed_at: %s -> %v", request.InstallationCompletedAt, *installationCompletedAt)
		} else {
			log.Printf("Failed to parse installation_completed_at: %s", request.InstallationCompletedAt)
		}
	}

	// Set default values
	if request.Status == "" {
		request.Status = "pending"
	}
	if request.InstallationType == "" {
		request.InstallationType = "new_installation"
	}
	if request.UserStatus == "" {
		request.UserStatus = "Active"
	}
	if request.KepemilikanPerangkat == "" {
		request.KepemilikanPerangkat = "owned"
	}

	// Determine technician_id for backward compatibility
	var technicianID *string
	if request.TechnicianID != "" {
		// Legacy: single technician provided
		technicianID = &request.TechnicianID
	} else if len(request.Technicians) > 0 {
		// New: multiple technicians - set primary technician as the legacy technician_id
		for _, tech := range request.Technicians {
			if tech.IsPrimary {
				technicianID = &tech.TechnicianID
				break
			}
		}
		// If no primary found, use first senior technician
		if technicianID == nil {
			for _, tech := range request.Technicians {
				if tech.Role == "senior" {
					technicianID = &tech.TechnicianID
					break
				}
			}
		}
	}

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
		CustomerID:                     stringToPtr(request.CustomerID),
		TechnicianID:                   technicianID,
		Status:                         request.Status,
		Notes:                          request.Notes,
		DocumentType:                   stringToPtr(request.DocumentType),
		DocumentPhoto:                  stringToPtr(request.DocumentPhoto),
		InstallationType:               request.InstallationType,
		IsTerminal:                     isTerminal,
		TerminalCustomerInstallationID: terminalCustomerInstallationId,
		Latitude:                       latitude,
		Longitude:                      longitude,
		InstallationCompletedAt:        installationCompletedAt,
		TrialEndDate:                   trialEndDate,
		ServiceReadyDate:               serviceReadyDate,
		OnAirDate:                      onAirDate,
	}

	// Normalize document photo path before saving
	if request.DocumentPhoto != "" {
		originalPath := request.DocumentPhoto
		request.DocumentPhoto = normalizeDocumentPhotoPath(request.DocumentPhoto)
		log.Printf("Repository: Normalized document photo path from '%s' to '%s'", originalPath, request.DocumentPhoto)
	}

	// Log installation data before creating
	var documentPhotoStr, documentTypeStr string
	if installation.DocumentPhoto != nil {
		documentPhotoStr = *installation.DocumentPhoto
	} else {
		documentPhotoStr = "nil"
	}
	if installation.DocumentType != nil {
		documentTypeStr = *installation.DocumentType
	} else {
		documentTypeStr = "nil"
	}
	log.Printf("=== REPOSITORY DEBUG ===")
	log.Printf("Creating installation record - DocumentPhoto: %s, DocumentType: %s",
		documentPhotoStr, documentTypeStr)
	log.Printf("Request DocumentPhoto field: '%s'", request.DocumentPhoto)
	log.Printf("Request DocumentType field: '%s'", request.DocumentType)
	log.Printf("=== END REPOSITORY DEBUG ===")

	if err := tx.Create(&installation).Error; err != nil {
		log.Printf("Failed to create installation record: %v", err)
		tx.Rollback()
		return installation, err
	}

	log.Printf("Installation record created successfully with ID: %s", installation.ID)

	// Log what was actually saved to database
	log.Printf("=== DATABASE SAVE DEBUG ===")
	if installation.DocumentPhoto != nil {
		log.Printf("✅ Document Photo saved to DB: '%s'", *installation.DocumentPhoto)
	} else {
		log.Printf("❌ Document Photo is NULL in DB")
	}
	if installation.DocumentType != nil {
		log.Printf("✅ Document Type saved to DB: '%s'", *installation.DocumentType)
	} else {
		log.Printf("❌ Document Type is NULL in DB")
	}
	log.Printf("=== END DATABASE SAVE DEBUG ===")

	// Update customer with company_id and sales_representative_id if provided
	if request.CustomerCompanyID != "" || request.CustomerSalesRepresentativeID != "" {
		updateData := make(map[string]interface{})
		if request.CustomerCompanyID != "" {
			updateData["company_id"] = request.CustomerCompanyID
		}
		if request.CustomerSalesRepresentativeID != "" {
			updateData["sales_representative_id"] = request.CustomerSalesRepresentativeID
		}

		if err := tx.Model(&entities.Customer{}).
			Where("id = ?", request.CustomerID).
			Updates(updateData).Error; err != nil {
			log.Printf("Failed to update customer with company/sales rep info: %v", err)
			tx.Rollback()
			return installation, err
		}
		log.Printf("Updated customer with company_id: %s, sales_representative_id: %s",
			request.CustomerCompanyID, request.CustomerSalesRepresentativeID)
	}

	// Create network device - only if essential fields are provided
	if request.AssetsID != "" && (request.MacAddress != "" || request.IPStatic != "" || request.SwitchID != "") {
		// Prepare ProductID - only set if not empty
		var productID *string
		if request.ProductID != "" {
			productID = &request.ProductID
		}

		networkDevice := entities.NetworkDevice{
			ID:         "",
			CustomerID: request.CustomerID,
			AssetsID: sql.NullString{
				String: request.AssetsID,
				Valid:  request.AssetsID != "",
			},
			AssetItemID:            stringToPtr(request.AssetItemID), // Set asset_items_id from form
			CustomerInstallationID: &installation.ID,
			SwitchID:               stringToPtr(request.SwitchID),
			PortNumber:             stringToPtr(request.PortNumber),
			RemotePort:             stringToPtr(request.RemotePort),
			EthPort:                stringToPtr(request.EthPort),
			MacAddress:             stringToPtr(request.MacAddress),
			IPStatic:               stringToPtr(request.IPStatic),
			KepemilikanPerangkat:   request.KepemilikanPerangkat,
			ProductID:              productID,
		}

		if err := tx.Create(&networkDevice).Error; err != nil {
			tx.Rollback()
			return installation, err
		}
		log.Printf("Created network device for installation %s", installation.ID)
		log.Printf("Network device AssetItemID: %s (from request.AssetItemID: %s)",
			getStringValue(networkDevice.AssetItemID), request.AssetItemID)

		// Create asset transaction to track asset item assignment
		log.Printf("=== ASSET TRANSACTION DEBUG ===")
		log.Printf("Request.MacAddress: '%s'", request.MacAddress)
		log.Printf("Request.AssetItemID: '%s'", request.AssetItemID)
		log.Printf("Condition check: MacAddress != '' (%t) || AssetItemID != '' (%t) = %t",
			request.MacAddress != "", request.AssetItemID != "",
			request.MacAddress != "" || request.AssetItemID != "")

		if request.MacAddress != "" || request.AssetItemID != "" {
			var assetItem entities.AssetItem
			var err error

			// Try to find asset item by asset_item_id first, then by MAC address
			if request.AssetItemID != "" {
				err = tx.Where("id = ? AND status = 'in_stock'", request.AssetItemID).First(&assetItem).Error
				if err == nil {
					log.Printf("Found asset item by ID: %s", request.AssetItemID)
				}
			}

			// If not found by ID, try by MAC address
			if err != nil && request.MacAddress != "" {
				err = tx.Where("mac_address = ? AND status = 'in_stock'", request.MacAddress).First(&assetItem).Error
				if err == nil {
					log.Printf("Found asset item by MAC address: %s", request.MacAddress)
				}
			}

			if err == nil {
				// Create asset transaction record
				assetTransaction := entities.AssetTransaction{
					ID:                     "",
					CustomerInstallationID: installation.ID,
					AssetItemID:            &assetItem.ID, // Link to specific asset item
					AssetID:                assetItem.AssetID,
					TransactionType:        "out", // Asset going out to customer
					Quantity:               1,
					Notes:                  stringToPtr(fmt.Sprintf("Asset assigned to installation - MAC: %s, Item ID: %s", assetItem.MacAddress, assetItem.ID)),
					TransactionDate:        time.Now(),
					CreatedBy:              createdByUserID, // Use the authenticated user ID
				}

				if err := tx.Create(&assetTransaction).Error; err != nil {
					log.Printf("Failed to create asset transaction: %v", err)
					// Don't rollback - this is not critical enough to fail the entire installation
				} else {
					// Update asset item status to 'in_use'
					err = tx.Model(&assetItem).Update("status", "in_use").Error
					if err != nil {
						log.Printf("Failed to update asset item status: %v", err)
					} else {
						log.Printf("✅ Created asset transaction and updated asset item %s (MAC: %s) status to 'in_use' for installation %s", assetItem.ID, assetItem.MacAddress, installation.ID)
					}
				}
			} else {
				log.Printf("❌ Asset item not found - AssetItemID: %s, MacAddress: %s, Error: %v", request.AssetItemID, request.MacAddress, err)
			}
		}
	} else {
		log.Printf("Skipping network device creation - insufficient network data provided")
	}

	// Create customer service - only if both login and password are provided
	if request.UserLogin != "" && request.Password != "" {
		// Auto-populate device_id from first available network device for this installation if not provided
		var deviceID *string

		// Try to get the most recently created network device for this installation
		var networkDevice entities.NetworkDevice
		if err := tx.Where("customer_installation_id = ?", installation.ID).Order("created_at DESC").First(&networkDevice).Error; err == nil {
			deviceID = &networkDevice.ID
			log.Printf("Auto-populated device_id with network device: %s for customer service", *deviceID)
		}

		customerService := entities.CustomerService{
			ID:                     "",
			CustomerID:             request.CustomerID,
			CustomerInstallationID: &installation.ID,
			DeviceID:               deviceID,
			CableType:              stringToPtr(request.CableType),
			CableLength:            stringToFloat(request.CableLength), // Use the provided cable length or nil
			EndPortType:            stringToPtr(request.EndPortType),
			UserLogin:              stringToPtr(request.UserLogin),
			Password:               stringToPtr(request.Password),
			UserStatus:             request.UserStatus,
			InstallationNotes:      stringToPtr(request.InstallationNotes),
		}

		if err := tx.Create(&customerService).Error; err != nil {
			tx.Rollback()
			return installation, err
		}
		log.Printf("Created customer service for installation %s with auto-populated device_id: %s",
			installation.ID, getStringValue(deviceID))
	} else {
		log.Printf("Skipping customer service creation - login and/or password not provided")
	}

	// Note: Cable table has been merged with customer_services table
	// Cable data is now stored in customer_services table (cable_type, cable_length, end_port_type)
	// No separate cable records are created anymore

	// Update images with installation ID
	if len(request.ImageIds) > 0 {
		log.Printf("Updating %d images with installation ID: %s", len(request.ImageIds), installation.ID)
		if err := tx.Model(&entities.Image{}).
			Where("id IN ?", request.ImageIds).
			Update("archive_installation_id", installation.ID).Error; err != nil {
			log.Printf("Failed to update images with installation ID: %v", err)
			tx.Rollback()
			return installation, err
		}
		log.Printf("Successfully updated images with installation ID")
	}

	// Create technician photos as Images with archive_installation_id
	log.Printf("=== REPOSITORY TECHNICIAN PHOTOS DEBUG ===")
	log.Printf("Request.TechnicianPhotos length: %d", len(request.TechnicianPhotos))
	for i, photoPath := range request.TechnicianPhotos {
		log.Printf("  Photo %d: %s", i+1, photoPath)
	}

	if len(request.TechnicianPhotos) > 0 {
		log.Printf("Creating %d technician photos for installation: %s", len(request.TechnicianPhotos), installation.ID)
		for i, photoPath := range request.TechnicianPhotos {
			log.Printf("Processing technician photo %d: %s", i+1, photoPath)

			now := time.Now()
			image := entities.Image{
				ID:                    "",
				File:                  photoPath,
				FullPath:              photoPath,
				ArchiveInstallationId: installation.ID,
				CreatedAt:             now,
				UpdatedAt:             &now,
			}

			log.Printf("Creating Image record for technician photo %d with archive_installation_id: %s", i+1, installation.ID)

			if err := tx.Create(&image).Error; err != nil {
				log.Printf("❌ Failed to create technician photo %d: %v", i+1, err)
				tx.Rollback()
				return installation, err
			}
			log.Printf("✅ Created technician photo %d: %s (Image ID: %s)", i+1, photoPath, image.ID)
		}
		log.Printf("✅ Successfully created all %d technician photos", len(request.TechnicianPhotos))
	} else {
		log.Printf("No technician photos to create")
	}
	log.Printf("=== END REPOSITORY TECHNICIAN PHOTOS DEBUG ===")

	// Create technician assignments
	if len(request.Technicians) > 0 {
		log.Printf("Creating technician assignments for installation: %s", installation.ID)
		if err := r.CreateInstallationTechnicians(tx, installation.ID, request.Technicians); err != nil {
			log.Printf("Failed to create technician assignments: %v", err)
			tx.Rollback()
			return installation, err
		}
		log.Printf("Successfully created technician assignments")
	} else if request.TechnicianID != "" {
		// Backward compatibility: if old TechnicianID is provided, create a single senior assignment
		log.Printf("Creating legacy technician assignment for installation: %s", installation.ID)
		legacyTech := []TechnicianAssignment{
			{
				TechnicianID: request.TechnicianID,
				Role:         "senior",
				IsPrimary:    true,
				Notes:        "",
			},
		}
		if err := r.CreateInstallationTechnicians(tx, installation.ID, legacyTech); err != nil {
			log.Printf("Failed to create legacy technician assignment: %v", err)
			tx.Rollback()
			return installation, err
		}
		log.Printf("Successfully created legacy technician assignment")
	}

	if err := tx.Commit().Error; err != nil {
		return installation, err
	}

	// Load the created installation with relations
	var result entities.CustomerInstallation
	err := r.db.Preload("Customer").
		Preload("Technician").
		Preload("InstallationTechnicians").
		Preload("InstallationTechnicians.Technician").
		Preload("Images").
		Preload("NetworkDevices").
		Preload("NetworkDevices.Assets").
		Preload("CustomerServices").
		Where("id = ?", installation.ID).
		First(&result).Error

	return result, err
}

// DeleteInstallation - Delete installation by ID and update asset status to in_stock
func (r *ReportInstallationRepository) DeleteInstallation(installationId string) error {
	// Start transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// First, get the installation to find associated MAC addresses
	var installation entities.CustomerInstallation
	if err := tx.Preload("NetworkDevices").Where("id = ?", installationId).First(&installation).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update MAC address status back to in_stock for each network device
	for _, networkDevice := range installation.NetworkDevices {
		if networkDevice.MacAddress != nil && *networkDevice.MacAddress != "" {
			// Find the asset item by MAC address and update status to in_stock
			if err := tx.Model(&entities.AssetItem{}).
				Where("mac_address = ?", *networkDevice.MacAddress).
				Update("status", "in_stock").Error; err != nil {
				tx.Rollback()
				return err
			}
			log.Printf("🔄 Updated asset status to 'in_stock' for MAC address: %s", *networkDevice.MacAddress)
		}
	}

	// Perform Mikrotik disable before deleting database records
	if err := r.performMikrotikCleanup(installation); err != nil {
		log.Printf("⚠️ Warning: Mikrotik disable failed for installation %s: %v", installationId, err)
		// Continue with deletion even if Mikrotik disable fails
	}

	// Delete related records first (to avoid foreign key constraints)

	// Delete installation provisioning logs (Mikrotik provisioning)
	if err := tx.Where("customer_installation_id = ?", installationId).
		Delete(&entities.InstallationProvisioningLog{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	log.Printf("🗑️ Deleted Mikrotik provisioning logs for installation %s", installationId)

	// Mark recurring invoices related to this installation as completed (do not delete)
	if err := tx.Model(&entities.RecurringInvoice{}).
		Where("customer_installation = ?", installationId).
		Update("status", entities.RecurringInvoiceStatusCompleted).Error; err != nil {
		tx.Rollback()
		return err
	}
	log.Printf("🗑️ Marked recurring invoices as 'completed' for installation %s", installationId)

	// Delete installation technicians
	if err := tx.Where("customer_installation_id = ?", installationId).
		Delete(&entities.InstallationReportTechnician{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete customer services
	if err := tx.Where("customer_installation_id = ?", installationId).
		Delete(&entities.CustomerService{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete network devices
	if err := tx.Where("customer_installation_id = ?", installationId).
		Delete(&entities.NetworkDevice{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete asset transactions
	if err := tx.Where("customer_installation_id = ?", installationId).
		Delete(&entities.AssetTransaction{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Note: Cable table no longer exists - data migrated to customer_services

	// Delete installation images (Images are linked via ArchiveInstallationId)
	if err := tx.Model(&entities.Image{}).
		Where("archive_installation_id = ?", installationId).
		Update("archive_installation_id", nil).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Finally, delete the installation itself
	if err := tx.Where("id = ?", installationId).
		Delete(&entities.CustomerInstallation{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return err
	}

	log.Printf("✅ Successfully deleted installation %s, disabled Mikrotik configurations, deleted provisioning logs, deleted recurring invoices, and updated asset statuses to 'in_stock'", installationId)
	return nil
}

// UpdateInstallationCodeName - Update code_name for an installation
func (r *ReportInstallationRepository) UpdateInstallationCodeName(installationId string, codeName string) error {
	return r.db.Model(&entities.CustomerInstallation{}).
		Where("id = ?", installationId).
		Update("code_name", codeName).Error
}

// performMikrotikCleanup performs Mikrotik RouterOS disable for an installation
// Disables queues, netwatch, IP bindings, schedulers, and scripts based on customer_id and mac_address
func (r *ReportInstallationRepository) performMikrotikCleanup(installation entities.CustomerInstallation) error {
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
