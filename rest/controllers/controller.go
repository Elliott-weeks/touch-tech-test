package controllers

import (
	"ajbell.co.uk/app"
	"ajbell.co.uk/pkg/models"
	"ajbell.co.uk/pkg/service"
	"github.com/gofiber/fiber/v2"
)

type Dependencies struct {
	AllocationService service.Allocate
}

func GetDeposits(c *fiber.Ctx) error {

	id := c.Params("id")

	var result *models.Deposit

	app.Http.Database.DB.Preload("Receipts.Allocations").First(&result, "id = ?", id)

	if result.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "Deposit does not exist"})
	}

	// want an empty an array instead of null within the json
	if result.Receipts == nil {
		result.Receipts = make([]models.Receipt, 0)
	}
	return c.JSON(result)

}

/**
Example request:

{
	"client_id":1,
	"amount": 10000000,
	"proposed_allocation":[
		{
		"account_id":1,
		"split":0.56
	},
    {
		"account_id":2,
		"split":0.24
	},
		{
		"account_id":4,
		"split":0.20
	}
	]
}

*/

func CreateDeposit(c *fiber.Ctx) error {

	var payload *models.Deposit

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	errors := models.ValidateStruct(payload)

	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errors)
	}

	var count float32 = 0.00

	for _, allocation := range payload.ProposedAllocation {
		count += allocation.Split
		if count > 1 { // exceeded max percentage
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Allocation split exceeds 100%"})
		}
	}

	if count != 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Allocation split requires 100% allocation"})
	}

	err := app.Http.Database.Create(&payload).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"deposit_id": payload.ID})

}

/**
	Example request:
{
	"amount": 10000000
}


*/

func (d *Dependencies) ReceiptHandler(c *fiber.Ctx) error {

	var receipt *models.Receipt

	if err := c.BodyParser(&receipt); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	id := c.Params("id")

	var depo *models.Deposit

	app.Http.Database.DB.Preload("ProposedAllocation").First(&depo, id)

	if depo.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"status": "error", "message": "Deposit does not exist"})
	}

	receipt.DepositID = depo.ID

	err := d.AllocationService.AllocateReceipt(receipt, depo)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"receipt_id": receipt.ID})
}
