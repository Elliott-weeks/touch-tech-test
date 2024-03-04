package controllers

import (
	"ajbell.co.uk/app"
	"ajbell.co.uk/config"
	"ajbell.co.uk/pkg/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetDeposits(t *testing.T) {

	// mock the db
	testDB, mock, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating mock db")
	}

	app.Http = &config.AppConfig{}
	app.Http.Database = config.DatabaseConfig{
		DB: db,
	}

	idRow := sqlmock.NewRows([]string{"id"}).
		AddRow("1")

	mock.ExpectQuery("SELECT \\* FROM \"deposits\"(.*)").WillReturnRows(idRow)

	mock.ExpectQuery("SELECT \\* FROM \"deposits\"(.*)").WithArgs("10", uint(1)).WillReturnError(gorm.ErrRecordNotFound)

	app := fiber.New()

	app.Get("/deposit/:id", GetDeposits)

	t.Run("Successful retrieval of deposit with receipts and allocations", func(t *testing.T) {

		req := httptest.NewRequest("GET", "/deposit/1", nil)

		resp, _ := app.Test(req)

		// Verify, if the status code is as expected
		assert.Equal(t, resp.StatusCode, 200)
	})

	t.Run("Cant find deposit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/deposit/10", nil)

		resp, _ := app.Test(req)

		// Verify, if the status code is as expected
		assert.Equal(t, resp.StatusCode, 404)

	})

}

func TestCreateDeposits(t *testing.T) {

	testDB, mock, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating mock db")
	}

	app.Http = &config.AppConfig{}
	app.Http.Database = config.DatabaseConfig{
		DB: db,
	}

	idRow := sqlmock.NewRows([]string{"id"}).
		AddRow("1")

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"deposits\"(.*)").WillReturnRows(idRow)
	mock.ExpectQuery("INSERT INTO \"proposed_allocations\"(.*)").WillReturnRows(idRow)
	mock.ExpectCommit()

	app := fiber.New()

	app.Post("/deposit", CreateDeposit)

	t.Run("Successful creation of deposit", func(t *testing.T) {

		body := `{"client_id":1,"amount": 10000000,"proposed_allocation":[{"account_id":1,"split":0.56},{"account_id":2,"split":0.24},{"account_id":4,"split":0.20}]}`

		req := httptest.NewRequest("POST", "/deposit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)

		assert.Equal(t, 201, resp.StatusCode)

	})
	t.Run("Test unprocessable entity", func(t *testing.T) {

		body := `{"client_id":1,"amount": 10000000,"proposed_allocation":[{"account_id":1,"split":0.56},{"account_id":2,"split":0.24},{"account_id":4,"split":0.20}]}` // purpose typo

		req := httptest.NewRequest("POST", "/deposit", strings.NewReader(body))

		resp, _ := app.Test(req)

		assert.Equal(t, 400, resp.StatusCode)

	})

	t.Run("Test validate structure failure", func(t *testing.T) {

		body := `{"client_id":1,"aount": 10000000,"proposed_allocation":[{"account_id":1,"split":0.56},{"account_id":2,"split":0.24},{"account_id":4,"split":0.20}]}`

		req := httptest.NewRequest("POST", "/deposit", strings.NewReader(body))

		resp, _ := app.Test(req)

		assert.Equal(t, 400, resp.StatusCode)

	})

}

type MockAllocationService struct {
}

func (s *MockAllocationService) AllocateReceipt(receipt *models.Receipt, deposit *models.Deposit) error {
	if throwError {
		return errors.New("Mock error")
	}

	return nil
}

var throwError = false

func TestCreateAllocation(t *testing.T) {

	// mock the db
	testDB, mock, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating mock db")
	}

	app.Http = &config.AppConfig{}
	app.Http.Database = config.DatabaseConfig{
		DB: db,
	}

	idRow := sqlmock.NewRows([]string{"id"}).
		AddRow("1")

	mock.ExpectQuery("SELECT \\* FROM \"deposits\"(.*)").WillReturnRows(idRow)

	app := fiber.New()

	deps := Dependencies{
		AllocationService: &MockAllocationService{},
	}

	app.Post("/deposit/:id/receipt", deps.ReceiptHandler)

	t.Run("Successful creation of receipt", func(t *testing.T) {

		body := `{"amount":100000}`

		req := httptest.NewRequest("POST", "/deposit/1/receipt", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)

		assert.Equal(t, 201, resp.StatusCode)

	})

	t.Run("unprocessable entity", func(t *testing.T) {

		body := `{"amount":100000}`

		req := httptest.NewRequest("POST", "/deposit/1/receipt", strings.NewReader(body))

		resp, _ := app.Test(req)

		assert.Equal(t, 400, resp.StatusCode)

	})

}
