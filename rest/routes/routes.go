package routes

import (
	"ajbell.co.uk/pkg/service"
	"ajbell.co.uk/rest/controllers"
	"github.com/gofiber/fiber/v2"
)

func LoadRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// DEPOSIT CREATION
	api.Post("/deposit", controllers.CreateDeposit)
	api.Get("/deposit/:id", controllers.GetDeposits)

	allocationService := service.NewAllocationService()

	deps := controllers.Dependencies{
		AllocationService: allocationService,
	}

	//// attach the receipt
	api.Post("/deposit/:id/receipt", deps.ReceiptHandler)

}
