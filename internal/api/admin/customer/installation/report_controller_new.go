package customerinstallation

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"
	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/services"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ReportInstallationController struct {
	service ReportInstallationServiceInterface
}

func NewReportInstallationController(service ReportInstallationServiceInterface) *ReportInstallationController {
	return &ReportInstallationController{service: service}
}

// CreateReportInstallation - Create installation report with multipart form data
func (c *ReportInstallationController) CreateReportInstallation(ctx *fiber.Ctx) error {
	var request CreateReportInstallationRequest

	// Parse multipart form
	_, err := ctx.MultipartForm()
	if err != nil {
		return helpers.ResponseUtils(ctx, 400, false, "Invalid multipart form data", err.Error())
	}

	// Log all form values for debugging
	log.Printf("=== FORM VALUES DEBUG ===")
	form, _ := ctx.MultipartForm()
	if form != nil {
		for key, values := range form.Value {
			log.Printf("Form field '%s': %v", key, values)
		}
	}
	log.Printf("=== END FORM VALUES DEBUG ===")

	// Debug asset-related fields specifically
	log.Printf("=== ASSET FIELDS DEBUG ===")
	log.Printf("mac_address from form: '%s'", ctx.FormValue("mac_address"))
	log.Printf("asset_item_id from form: '%s'", ctx.FormValue("asset_item_id"))
	log.Printf("assets_id from form: '%s'", ctx.FormValue("assets_id"))
	log.Printf("=== END ASSET FIELDS DEBUG ===")

	// Parse form values
	request.CustomerID = ctx.FormValue("customer_id")
	request.TechnicianID = ctx.FormValue("technician_id")
	request.Status = ctx.FormValue("status")
	request.Notes = ctx.FormValue("notes")
	request.InstallationType = ctx.FormValue("installation_type")
	request.OnAirDate = ctx.FormValue("on_air_date")
	request.TrialEndDate = ctx.FormValue("trial_end_date")
	request.ServiceReadyDate = ctx.FormValue("service_ready_date")
	request.InstallationCompletedAt = ctx.FormValue("installation_completed_at")
	request.IsTerminal = ctx.FormValue("is_terminal")
	request.TerminalCustomerInstallationId = ctx.FormValue("terminal_customer_installation_id")
	// Parse latitude and longitude from form
	if latStr := ctx.FormValue("latitude"); latStr != "" {
		if parsed, err := strconv.ParseFloat(latStr, 64); err == nil {
			request.Latitude = &parsed
		}
	}
	if lonStr := ctx.FormValue("longitude"); lonStr != "" {
		if parsed, err := strconv.ParseFloat(lonStr, 64); err == nil {
			request.Longitude = &parsed
		}
	}
	request.DocumentType = ctx.FormValue("document_type")
	request.SwitchID = ctx.FormValue("switch_id")
	request.PortNumber = ctx.FormValue("port_number")
	request.RemotePort = ctx.FormValue("remote_port")
	request.EthPort = ctx.FormValue("eth_port")
	request.MacAddress = ctx.FormValue("mac_address")
	request.AssetItemID = ctx.FormValue("asset_item_id")
	request.IPStatic = ctx.FormValue("ip_static")
	request.KepemilikanPerangkat = ctx.FormValue("kepemilikan_perangkat")
	request.AssetsID = ctx.FormValue("assets_id")
	request.CableType = ctx.FormValue("cable_type")
	// Parse cable_length from string to float64
	if cableLengthStr := ctx.FormValue("cable_length"); cableLengthStr != "" {
		if parsed, err := strconv.ParseFloat(cableLengthStr, 64); err == nil {
			request.CableLength = parsed
		}
	}
	request.EndPortType = ctx.FormValue("end_port_type")
	request.UserLogin = ctx.FormValue("user_login")
	request.Password = ctx.FormValue("password")
	request.UserStatus = ctx.FormValue("user_status")
	request.InstallationNotes = ctx.FormValue("installation_notes")

	// Parse additional fields
	request.CustomerCompanyID = ctx.FormValue("customer_company_id")
	request.CustomerSalesRepresentativeID = ctx.FormValue("customer_sales_representative_id")
	request.ProductID = ctx.FormValue("product_id")

	// Parse technicians from JSON string
	techniciansStr := ctx.FormValue("technicians")
	if techniciansStr != "" {
		var technicians []TechnicianAssignment
		if err := json.Unmarshal([]byte(techniciansStr), &technicians); err == nil {
			request.Technicians = technicians
			log.Printf("Parsed %d technicians from JSON", len(technicians))
		} else {
			log.Printf("Failed to parse technicians JSON: %s, error: %v", techniciansStr, err)
		}
	}

	// Parse provisioning fields
	request.MacAddress = ctx.FormValue("mac_address")
	request.MaxLimit = ctx.FormValue("max_limit")

	// Parse boolean fields
	if autoProvStr := ctx.FormValue("auto_provision"); autoProvStr != "" {
		request.AutoProvision = autoProvStr == "true"
	}
	if dryRunStr := ctx.FormValue("dry_run"); dryRunStr != "" {
		request.DryRun = dryRunStr == "true"
	}

	// Parse image_ids from JSON string
	imageIdsStr := ctx.FormValue("image_ids")
	if imageIdsStr != "" {
		// Parse JSON string to []string
		var imageIds []string
		if err := json.Unmarshal([]byte(imageIdsStr), &imageIds); err == nil {
			request.ImageIds = imageIds
			log.Printf("Parsed image_ids: %v", imageIds)
		} else {
			log.Printf("Failed to parse image_ids JSON: %s, error: %v", imageIdsStr, err)
		}
	}

	// Parse technician photos notes
	request.TechnicianPhotosNotes = ctx.FormValue("technician_photos_notes")
	if request.TechnicianPhotosNotes != "" {
		log.Printf("Parsed technician_photos_notes: %s", request.TechnicianPhotosNotes)
	}

	// Parse auto-populated fields
	request.DeviceID = ctx.FormValue("device_id")
	request.ServiceActivationDate = ctx.FormValue("service_activation_date")
	request.InstallationTeamPhone = ctx.FormValue("installation_team_phone")
	request.IsTrial = ctx.FormValue("is_trial") == "true" // Parse as boolean

	// Handle document photo upload
	var documentPhotoPath string

	// Check if document_photo_path is provided (from frontend upload)
	if docPhotoPath := ctx.FormValue("document_photo_path"); docPhotoPath != "" {
		documentPhotoPath = docPhotoPath
		log.Printf("Using document photo path from frontend: %s", docPhotoPath)
	} else {
		log.Printf("No document_photo_path found in form data")
		// Log all form files for debugging
		form, formErr := ctx.MultipartForm()
		if formErr == nil {
			log.Printf("Multipart form files found: %v", len(form.File))
			for fieldName, files := range form.File {
				log.Printf("Form field '%s' has %d files", fieldName, len(files))
				for i, file := range files {
					log.Printf("  File %d: %s (size: %d, type: %s)", i, file.Filename, file.Size, file.Header.Get("Content-Type"))
				}
			}
		} else {
			log.Printf("Error getting multipart form: %v", formErr)
		}

		if file, err := ctx.FormFile("document_photo"); err == nil {
			// Log file upload info
			log.Printf("Document photo upload started - filename: %s, size: %d", file.Filename, file.Size)

			// Validate file type
			if !isValidImageFile(file) {
				return helpers.ResponseUtils(ctx, 400, false, "Invalid file type. Only JPG and PNG are allowed", nil)
			}

			// Generate unique filename
			ext := filepath.Ext(file.Filename)
			filename := "document_" + time.Now().Format("20060102_150405") + ext

			// Create upload directory if not exists
			uploadDir := "uploads/installations/documents"
			log.Printf("Creating upload directory: %s", uploadDir)
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Printf("Failed to create upload directory: %v", err)
				return helpers.ResponseUtils(ctx, 500, false, "Failed to create upload directory", err.Error())
			}
			log.Printf("Upload directory created successfully: %s", uploadDir)

			// Save file
			filePath := filepath.Join(uploadDir, filename)
			log.Printf("Attempting to save file to: %s", filePath)
			if err := ctx.SaveFile(file, filePath); err != nil {
				log.Printf("Failed to save document photo: %v", err)
				return helpers.ResponseUtils(ctx, 500, false, "Failed to save document photo", err.Error())
			}

			// Store only the filename in document_photo field, not the full path
			documentPhotoPath = filename

			// Log successful upload
			log.Printf("✅ Document photo uploaded successfully - filePath: %s, filename: %s", filePath, filename)
			log.Printf("✅ Document photo field will store filename only: %s", documentPhotoPath)

			// Verify file exists
			if _, err := os.Stat(filePath); err == nil {
				log.Printf("✅ File verification successful - file exists at: %s", filePath)
			} else {
				log.Printf("❌ File verification failed - file does not exist at: %s, error: %v", filePath, err)
			}
		} else {
			// Log if no file was uploaded
			log.Printf("❌ No document photo uploaded - error: %s", err.Error())

			// Check if document_photo field exists in form values
			if docPhotoValue := ctx.FormValue("document_photo"); docPhotoValue != "" {
				log.Printf("document_photo form value (not file): %s", docPhotoValue)
			} else {
				log.Printf("No document_photo field found in form values")
			}

			// Set documentPhotoPath to empty string to indicate no file was uploaded
			documentPhotoPath = ""
		}
	}

	// Set document photo path
	request.DocumentPhoto = documentPhotoPath

	// Log request data for debugging
	log.Printf("=== DOCUMENT PHOTO DEBUG ===")
	log.Printf("Document photo path set to: '%s'", documentPhotoPath)
	log.Printf("Request DocumentPhoto field: '%s'", request.DocumentPhoto)
	log.Printf("Document photo path length: %d", len(documentPhotoPath))
	if documentPhotoPath == "" {
		log.Printf("⚠️  WARNING: Document photo path is EMPTY - file upload may have failed!")
	} else {
		log.Printf("✅ Document photo path is set successfully: %s", documentPhotoPath)
	}
	log.Printf("=== END DOCUMENT PHOTO DEBUG ===")

	// Handle technician photo files
	var technicianPhotoPaths []string
	technicianPhotosCountStr := ctx.FormValue("technician_photos_count")

	log.Printf("=== TECHNICIAN PHOTOS DEBUG START ===")
	log.Printf("technician_photos_count from form: '%s'", technicianPhotosCountStr)

	// Debug all form values related to technician photos
	form, formErr := ctx.MultipartForm()
	if formErr == nil {
		log.Printf("All form values:")
		for key, values := range form.Value {
			if strings.Contains(key, "technician") {
				log.Printf("  %s: %v", key, values)
			}
		}
		log.Printf("All form files:")
		for fieldName, files := range form.File {
			if strings.Contains(fieldName, "technician") {
				log.Printf("  %s: %d files", fieldName, len(files))
				for i, file := range files {
					log.Printf("    File %d: %s (size: %d, type: %s)", i, file.Filename, file.Size, file.Header.Get("Content-Type"))
				}
			}
		}
	} else {
		log.Printf("Error getting multipart form: %v", formErr)
	}

	if technicianPhotosCountStr != "" {
		if count, err := strconv.Atoi(technicianPhotosCountStr); err == nil && count > 0 {
			log.Printf("=== PROCESSING TECHNICIAN PHOTOS ===")
			log.Printf("Processing %d technician photos", count)

			// Create upload directory for technician photos
			uploadDir := "uploads/installations/technician_photos"
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Printf("Failed to create technician photos upload directory: %v", err)
				return helpers.ResponseUtils(ctx, 500, false, "Failed to create upload directory", err.Error())
			}

			for i := 0; i < count; i++ {
				fieldName := fmt.Sprintf("technician_photo_%d", i)
				log.Printf("Looking for field: %s", fieldName)

				if file, err := ctx.FormFile(fieldName); err == nil {
					log.Printf("✅ Found technician photo %d: %s (size: %d, type: %s)", i+1, file.Filename, file.Size, file.Header.Get("Content-Type"))

					// Validate file type
					if !isValidImageFile(file) {
						log.Printf("❌ Invalid file type for technician photo %d: %s (type: %s)", i+1, file.Filename, file.Header.Get("Content-Type"))
						continue
					}
					log.Printf("✅ File type validation passed for technician photo %d", i+1)

					// Generate unique filename
					ext := filepath.Ext(file.Filename)
					filename := fmt.Sprintf("technician_photo_%d_%s%s", i+1, time.Now().Format("20060102_150405"), ext)
					log.Printf("Generated filename: %s", filename)

					// Save file
					filePath := filepath.Join(uploadDir, filename)
					log.Printf("Attempting to save to: %s", filePath)

					if err := ctx.SaveFile(file, filePath); err != nil {
						log.Printf("❌ Failed to save technician photo %d: %v", i+1, err)
						continue
					}

					// Verify file was saved
					if _, err := os.Stat(filePath); err == nil {
						technicianPhotoPaths = append(technicianPhotoPaths, filePath)
						log.Printf("✅ Successfully saved technician photo %d: %s", i+1, filePath)
					} else {
						log.Printf("❌ File verification failed for technician photo %d: %v", i+1, err)
					}
				} else {
					log.Printf("❌ Failed to get technician photo %d from field '%s': %v", i+1, fieldName, err)
				}
			}

			log.Printf("Successfully processed %d technician photos", len(technicianPhotoPaths))
			log.Printf("=== END TECHNICIAN PHOTOS DEBUG ===")
		} else {
			log.Printf("❌ Failed to parse technician_photos_count: %v", err)
		}
	} else {
		log.Printf("No technician_photos_count found in form data")
	}

	// Set technician photos in request
	request.TechnicianPhotos = technicianPhotoPaths
	log.Printf("=== FINAL TECHNICIAN PHOTOS RESULT ===")
	log.Printf("Total technician photos processed: %d", len(technicianPhotoPaths))
	for i, path := range technicianPhotoPaths {
		log.Printf("  Photo %d: %s", i+1, path)
	}
	log.Printf("=== END FINAL TECHNICIAN PHOTOS RESULT ===")

	log.Printf("Installation report request data - customer_id: %s, technician_id: %s, document_type: %s, document_photo: %s, installation_type: %s, assets_id: %s, switch_id: %s, port_number: %s, mac_address: %s, ip_static: %s, cable_type: %s, cable_length: %f, user_login: %s, user_status: %s, installation_notes: %s, customer_company_id: %s, customer_sales_rep_id: %s, product_id: %s, technician_photos_count: %d",
		request.CustomerID, request.TechnicianID, request.DocumentType, request.DocumentPhoto, request.InstallationType, request.AssetsID, request.SwitchID, request.PortNumber, request.MacAddress, request.IPStatic, request.CableType, request.CableLength, request.UserLogin, request.UserStatus, request.InstallationNotes, request.CustomerCompanyID, request.CustomerSalesRepresentativeID, request.ProductID, len(technicianPhotoPaths))

	// Validate required fields
	if request.CustomerID == "" {
		return helpers.ResponseUtils(ctx, 400, false, "Customer ID is required", nil)
	}

	// Validate technicians - either legacy technician_id OR new technicians array
	if request.TechnicianID == "" && len(request.Technicians) == 0 {
		return helpers.ResponseUtils(ctx, 400, false, "At least one technician is required", nil)
	}

	// If using new technicians array, validate at least one senior
	if len(request.Technicians) > 0 {
		hasSenior := false
		for _, tech := range request.Technicians {
			if tech.Role == "senior" {
				hasSenior = true
				break
			}
		}
		if !hasSenior {
			return helpers.ResponseUtils(ctx, 400, false, "At least one senior technician is required", nil)
		}
	}

	if request.AssetsID == "" {
		return helpers.ResponseUtils(ctx, 400, false, "Assets ID is required", nil)
	}

	// Validate IP format if provided
	if request.IPStatic != "" && !isValidIP(request.IPStatic) {
		return helpers.ResponseUtils(ctx, 400, false, "Invalid IP address format", nil)
	}

	// Validate MAC address format if provided
	if request.MacAddress != "" && !isValidMAC(request.MacAddress) {
		return helpers.ResponseUtils(ctx, 400, false, "Invalid MAC address format", nil)
	}

	// Get the authenticated user ID from the request context
	var createdBy string
	if userID := ctx.Locals("user_id"); userID != nil {
		createdBy = fmt.Sprintf("%v", userID)
		log.Printf("=== USER ID DEBUG ===")
		log.Printf("Extracted user ID from context: %s", createdBy)
	} else {
		// Fallback to a known admin user ID to avoid foreign key constraint issues
		createdBy = "b6cbf8b6-6a2e-45f3-a9ab-e30e5941bf5a" // Admin user ID from JWT
		log.Printf("=== USER ID DEBUG ===")
		log.Printf("No user ID in context, using fallback admin ID: %s", createdBy)
	}

	// Get installation count BEFORE creating the installation record
	// This ensures correct R-numbering (R1, R2, R3, etc.)
	var installationCount int
	if request.AutoProvision && request.MacAddress != "" {
		// We need to get the count before creating the installation
		// so that the R-number is calculated correctly
		var count int64
		err := c.service.GetDB().Table("customer_installations").
			Where("customer_installations.customer_id = ?", request.CustomerID).
			Count(&count).Error
		if err != nil {
			log.Printf("Error counting installations: %v", err)
			installationCount = 1 // Default fallback
		} else {
			installationCount = int(count) + 1
			log.Printf("🔍 Pre-installation count for customer %s: found %d existing, will use R%d",
				request.CustomerID, count, installationCount)
		}
	}

	// Create installation report first
	installation, err := c.service.CreateReportInstallationService(request, createdBy)
	if err != nil {
		return helpers.ResponseUtils(ctx, 500, false, "Failed to create installation report", err.Error())
	}

	// Handle MikroTik provisioning if enabled
	var provisioningResult map[string]interface{}
	if request.AutoProvision && request.MacAddress != "" {
		log.Printf("=== MIKROTIK PROVISIONING START ===")
		log.Printf("Auto-provisioning enabled for installation: %s", installation.ID)
		log.Printf("MAC Address: %s", request.MacAddress)
		log.Printf("Max Limit: %s", request.MaxLimit)
		log.Printf("Dry Run: %v", request.DryRun)

		// Get database connection
		db := database.GetDB()

		// Get the shared MikroTik service
		mikrotikService := services.GetSharedMikroTikService()
		if mikrotikService == nil {
			log.Printf("❌ MikroTik service not available - provisioning will be skipped")
			provisioningResult = map[string]interface{}{
				"status": "failed",
				"error":  "MikroTik service not connected",
			}

			// If MikroTik is not available, delete the installation and return error
			log.Printf("🗑️ Deleting installation due to MikroTik service unavailable: %s", installation.ID)
			if deleteErr := c.service.DeleteInstallation(installation.ID); deleteErr != nil {
				log.Printf("❌ Failed to delete installation after MikroTik service unavailable: %v", deleteErr)
			}

			return helpers.ResponseUtils(ctx, 500, false, "Installation created but MikroTik service unavailable", map[string]interface{}{
				"installation_id":    installation.ID,
				"provisioning_error": "MikroTik service not connected",
			})
		}

		provisioningService := services.NewMikrotikProvisioningService(db, mikrotikService)

		// Get customer name and area name from database
		var customerName, areaName string
		if installation.CustomerID != nil {
			// Get customer data with area information
			var customer entities.Customer
			err := db.Preload("Area").Where("id = ?", *installation.CustomerID).First(&customer).Error
			if err != nil {
				log.Printf("❌ Failed to get customer data: %v", err)
				customerName = "Unknown Customer"
				areaName = "Unknown Area"
			} else {
				customerName = customer.Name
				if customer.Area != nil {
					areaName = customer.Area.CodeName
				} else {
					areaName = "Unknown Area"
				}
			}
		} else {
			customerName = "Unknown Customer"
			areaName = "Unknown Area"
		}

		// Parse installation_completed_at to extract date and time
		startDate := ""
		startTime := "00:00:00"

		// Try to parse installation_completed_at first
		if request.InstallationCompletedAt != "" {
			log.Printf("🔍 Parsing InstallationCompletedAt: %s", request.InstallationCompletedAt)
			if parsedTime, err := time.Parse("2006-01-02T15:04", request.InstallationCompletedAt); err == nil {
				startDate = parsedTime.Format("2006-01-02")
				startTime = parsedTime.Format("15:04:05")
				log.Printf("✅ Parsed InstallationCompletedAt: date=%s, time=%s", startDate, startTime)
			} else {
				log.Printf("❌ Failed to parse InstallationCompletedAt: %v", err)
				// Try alternative formats
				formats := []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"}
				for _, format := range formats {
					if parsedTime, err := time.Parse(format, request.InstallationCompletedAt); err == nil {
						startDate = parsedTime.Format("2006-01-02")
						startTime = parsedTime.Format("15:04:05")
						log.Printf("✅ Parsed with format %s: date=%s, time=%s", format, startDate, startTime)
						break
					}
				}
			}
		}

		// Fallback to on_air_date if installation_completed_at parsing failed or is empty
		if startDate == "" && request.OnAirDate != "" {
			log.Printf("🔍 Using OnAirDate as fallback: %s", request.OnAirDate)
			if parsedTime, err := time.Parse("2006-01-02", request.OnAirDate); err == nil {
				startDate = parsedTime.Format("2006-01-02")
				startTime = "00:00:00"
				log.Printf("✅ Parsed OnAirDate: date=%s, time=%s", startDate, startTime)
			}
		}

		// Final fallback to current date if both are empty
		if startDate == "" {
			now := time.Now()
			startDate = now.Format("2006-01-02")
			startTime = now.Format("15:04:05")
			log.Printf("⚠️ Using current date as final fallback: date=%s, time=%s", startDate, startTime)
		}

		log.Printf("📅 Final start_date: %s, start_time: %s", startDate, startTime)

		// Create provisioning request
		provReq := services.ProvisioningRequest{
			InstallationID: installation.ID,
			CustomerID:     request.CustomerID,
			CustomerName:   customerName,
			AreaName:       areaName,
			MACAddress:     request.MacAddress,
			IPAddress:      request.IPStatic, // Use IP address from the form
			StartDate:      startDate,
			StartTime:      startTime,
			MaxLimit:       request.MaxLimit,
			Comment:        fmt.Sprintf("%s/%s", customerName, request.MaxLimit),
			DryRun:         request.DryRun,
			CreatedBy:      createdBy,
		}

		// Execute provisioning with pre-calculated installation count
		provResult, provErr := provisioningService.ProvisionInstallationWithCount(provReq, installationCount)
		if provErr != nil {
			log.Printf("❌ Provisioning failed: %v", provErr)
			provisioningResult = map[string]interface{}{
				"status": "failed",
				"error":  provErr.Error(),
			}

			// If provisioning fails, delete the installation and return error
			log.Printf("🗑️ Deleting installation due to provisioning failure: %s", installation.ID)
			if deleteErr := c.service.DeleteInstallation(installation.ID); deleteErr != nil {
				log.Printf("❌ Failed to delete installation after provisioning failure: %v", deleteErr)
			}

			return helpers.ResponseUtils(ctx, 500, false, "Installation created but provisioning failed", map[string]interface{}{
				"installation_id":    installation.ID,
				"provisioning_error": provErr.Error(),
			})
		} else {
			log.Printf("✅ Provisioning completed: %+v", provResult)

			// Update installation record with code_name from provisioning
			if provResult.CodeName != "" {
				updateErr := c.service.UpdateInstallationCodeName(installation.ID, provResult.CodeName)
				if updateErr != nil {
					log.Printf("⚠️ Failed to update installation code_name: %v", updateErr)
				} else {
					log.Printf("✅ Updated installation code_name: %s", provResult.CodeName)
					// Update the installation object for response
					installation.CodeName = &provResult.CodeName
				}
			}

			provisioningResult = map[string]interface{}{
				"status":            "success",
				"success":           provResult.Success,
				"code_name":         provResult.CodeName,
				"ip_address":        provResult.IPAddress,
				"commands":          provResult.Commands,
				"dry_run":           provResult.DryRun,
				"execution_time_ms": provResult.ExecutionTimeMs,
			}
		}
		log.Printf("=== MIKROTIK PROVISIONING END ===")
	} else {
		log.Printf("MikroTik provisioning skipped - Auto-provision: %v, MAC: %s", request.AutoProvision, request.MacAddress)
		provisioningResult = map[string]interface{}{
			"status":  "skipped",
			"message": "Auto-provisioning disabled or MAC address not provided",
		}
	}

	// Prepare response with installation information including document photo and provisioning
	response := map[string]interface{}{
		"installation":   installation,
		"document_photo": installation.DocumentPhoto, // Explicitly include document photo in response
		"provisioning":   provisioningResult,         // Include provisioning result
	}

	// Log response for debugging
	log.Printf("=== RESPONSE DEBUG ===")
	log.Printf("Response data - Installation ID: %s, Document Photo: %v",
		installation.ID, installation.DocumentPhoto)
	if installation.DocumentPhoto != nil {
		log.Printf("✅ Document Photo in response: '%s'", *installation.DocumentPhoto)
	} else {
		log.Printf("❌ Document Photo is NULL in response")
	}
	if installation.DocumentType != nil {
		log.Printf("✅ Document Type in response: '%s'", *installation.DocumentType)
	} else {
		log.Printf("❌ Document Type is NULL in response")
	}
	log.Printf("=== END RESPONSE DEBUG ===")

	return helpers.ResponseUtils(ctx, 201, true, "Installation report created successfully", response)
}

// isValidImageFile - Validate if uploaded file is a valid image
func isValidImageFile(file *multipart.FileHeader) bool {
	allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/jpeg", "application/octet-stream"}
	contentType := file.Header.Get("Content-Type")
	filename := file.Filename
	ext := strings.ToLower(filepath.Ext(filename))

	// Log file info for debugging
	log.Printf("File validation - filename: %s, contentType: %s, ext: %s", filename, contentType, ext)

	// Check by content type
	for _, allowedType := range allowedTypes {
		if contentType == allowedType {
			log.Printf("File validation passed by content type: %s", contentType)
			return true
		}
	}

	// Check by file extension as fallback
	allowedExts := []string{".jpg", ".jpeg", ".png"}
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			log.Printf("File validation passed by extension: %s", ext)
			return true
		}
	}

	log.Printf("File validation failed - contentType: %s, ext: %s", contentType, ext)
	return false
}

// isValidIP - Basic IP address validation
func isValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		// Additional validation can be added here
	}
	return true
}

// isValidMAC - Basic MAC address validation
func isValidMAC(mac string) bool {
	// Remove common separators
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")

	// Check if it's 12 characters long and contains only hex characters
	if len(mac) != 12 {
		return false
	}

	for _, char := range mac {
		if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}
