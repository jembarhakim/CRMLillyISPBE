package items_transaction

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

func ItemsTransactionRoute(router fiber.Router) {
	// Initialize repository, service, and handler
	repository := NewItemsTransactionRepository(database.GetDB())
	service := NewItemsTransactionService(repository)
	handler := NewItemsTransactionHandler(service)

	// Apply auth middleware if needed (assuming you have one)
	// router.Use(authMiddleware)

	// Routes
	router.Get("/", handler.GetAllItemsTransactionHandler)
	router.Get("/:id", handler.GetByIdItemsTransactionHandler)
	router.Post("/", handler.CreateItemsTransactionHandler)
	router.Put("/:id", handler.UpdateItemsTransactionHandler)
	router.Delete("/:id", handler.DeleteItemsTransactionHandler)

	// INV-02: stock transfer between holders. Auth is applied on this route only; the routes above
	// are left untouched on purpose.
	router.Post("/transfers", helpers.VerifyToken, handler.CreateTransferHandler)

	// INV-02 (read only): stock per holder and mutation history. Both paths have two segments, so
	// the "/:id" route above cannot capture them.
	router.Get("/stock/balance", helpers.VerifyToken, handler.GetStockBalanceHandler)
	router.Get("/stock/mutations", helpers.VerifyToken, handler.GetStockMutationsHandler)
}

