package service

import (
	"ajbell.co.uk/app"
	"ajbell.co.uk/config"
	"ajbell.co.uk/pkg/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestCalculateAllocationNiceEvenNum(t *testing.T) {
	split, remainder := calculateAllocation(100, 0.50)

	if !split.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("Calculating split is incorrect")
	}
	if !remainder.IsZero() {
		t.Fatalf("Error calculating remainder is incorrect %s %v", remainder, 0)
	}
}

func TestCalculateAllocationRoundUp(t *testing.T) {
	split, remainder := calculateAllocation(33, 0.30)

	if !split.Equal(decimal.NewFromInt(9)) {
		t.Fatalf("Calculating split is incorrect %s %v", split, 10)
	}
	if !remainder.Round(2).Equal(decimal.NewFromFloat(0.9)) {
		t.Fatalf("Calculating split is incorrect %s %v", remainder, 0.9)
	}
}

func TestCalculateAllocationRoundDown(t *testing.T) {
	split, remainder := calculateAllocation(11, 0.13)

	if !split.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("Error calculating split is incorrect %s %v", split, 1)
	}

	if !remainder.Round(2).Equal(decimal.NewFromFloat(0.43)) {
		t.Fatalf("Error calculating remainder is incorrect %s %v", remainder, 0.43)
	}
}

func TestAddOrCreateAllocationToGiaPot(t *testing.T) {
	mockMapData := decimal.NewFromFloat(10.0)

	tests := []struct {
		name        string
		initialData map[uint]*decimal.Decimal
		amount      decimal.Decimal
		potID       uint
		expected    decimal.Decimal
	}{
		{
			name: "KeyExists",
			initialData: map[uint]*decimal.Decimal{
				1: &mockMapData,
			},
			amount:   decimal.NewFromFloat(5.0),
			potID:    1,
			expected: decimal.NewFromFloat(15.0),
		},
		{
			name:        "KeyDoesNotExist",
			initialData: map[uint]*decimal.Decimal{},
			amount:      decimal.NewFromFloat(5.0),
			potID:       2,
			expected:    decimal.NewFromFloat(5.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			giaAllocations := cloneMap(tt.initialData)

			addOrCreateAllocationToGiaPot(tt.amount, tt.potID, giaAllocations)

			actual, ok := giaAllocations[tt.potID]
			if !ok {
				t.Error("Expected key to exist in the map")
			}

			if !tt.expected.Equal(*actual) {
				t.Errorf("Expected value: %s, Actual value: %s", tt.expected.String(), actual.String())
			}
		})
	}
}

func cloneMap(originalMap map[uint]*decimal.Decimal) map[uint]*decimal.Decimal {
	clonedMap := make(map[uint]*decimal.Decimal, len(originalMap))
	for k, v := range originalMap {
		clonedMap[k] = v
	}
	return clonedMap
}

func TestGetCurrentAmountAllocated(t *testing.T) {

	testDB, mock, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	if err != nil {
		t.Fatalf("Unable to create mock db: %v", err)
	}

	allocService := NewAllocationService()

	mock.ExpectBegin()

	mock.ExpectQuery("^SELECT COALESCE\\(SUM\\(al\\.amount\\), 0\\) FROM clients c .*").
		WithArgs("SIPP", 1).
		WillReturnRows(sqlmock.NewRows([]string{"COALESCE"}).AddRow(0))

	tx := db.Begin()

	result, err := allocService.DbOps.getCurrentAmountAllocated(tx, "SIPP", 1)

	assert.NoError(t, err)

	assert.Equal(t, int64(0), result)

	assert.NoError(t, mock.ExpectationsWereMet())

}

var getCurrentAmountAllocatedClientIdSipp uint
var getCurrentAmountAllocatedWrapperSipp string

var allocationInMockSipp models.Allocation

func TestProcessSIPPAllocation(t *testing.T) {
	testDB, _, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	if err != nil {
		t.Fatalf("Unable to create mock db: %v", err)
	}

	allocService := NewAllocationService()
	allocService.DbOps = &MockDBOperations{}

	receipt := models.Receipt{}
	receipt.ID = 1
	deposit := models.Deposit{}
	deposit.ID = 1
	deposit.ClientID = 2
	account := models.Account{}
	account.ID = 1
	account.Wrapper = "SIPP"
	account.PotID = 1

	tx := db.Begin()

	err = allocService.AlOps.processSIPPAllocation(
		tx,
		&receipt,
		&deposit,
		&account,
		decimal.NewFromInt(1000),
		make(map[uint]*decimal.Decimal),
		allocService.DbOps,
	)

	assert.NoError(t, err)
	assert.Equal(t, "SIPP", getCurrentAmountAllocatedWrapperSipp, "wrapper name does not match")
	assert.Equal(t, uint(2), getCurrentAmountAllocatedClientIdSipp, "client id does not match")
	amount := decimal.NewFromInt(int64(allocationInMockSipp.Amount))
	assert.Equal(t, decimal.NewFromInt(1000), amount, "Values do not match")

}

type MockDBOperations struct{}

func (m *MockDBOperations) getCurrentAmountAllocated(tx *gorm.DB, wrapper string, clientID uint) (int64, error) {
	getCurrentAmountAllocatedClientIdSipp = clientID
	getCurrentAmountAllocatedWrapperSipp = wrapper
	return 0, nil
}

func (m *MockDBOperations) saveAllocation(tx *gorm.DB, allocation models.Allocation) error {
	allocationInMockSipp = allocation
	return nil
}

var prevGetCurrentAmountAllocatedClientIdSipp uint
var prevGetCurrentAmountAllocatedWrapperSipp string

var prevallocationInMockSipp models.Allocation

func TestProcessSIPPAllocationWithPrevAllocationOverSub(t *testing.T) {
	testDB, _, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	if err != nil {
		t.Fatalf("Unable to create mock db: %v", err)
	}

	allocService := NewAllocationService()
	allocService.DbOps = &MockDBOperationsPrevAllocation{}

	receipt := models.Receipt{}
	receipt.ID = 1
	deposit := models.Deposit{}
	deposit.ID = 1
	deposit.ClientID = 2
	account := models.Account{}
	account.ID = 1
	account.Wrapper = "SIPP"
	account.PotID = 1

	tx := db.Begin()

	err = allocService.AlOps.processSIPPAllocation(
		tx,
		&receipt,
		&deposit,
		&account,
		decimal.NewFromInt(1000),
		make(map[uint]*decimal.Decimal),
		allocService.DbOps,
	)

	assert.NoError(t, err)
	assert.Equal(t, "SIPP", prevGetCurrentAmountAllocatedWrapperSipp, "wrapper name does not match")
	assert.Equal(t, uint(2), prevGetCurrentAmountAllocatedClientIdSipp, "client id does not match")

	assert.Equal(t, models.Allocation{}, prevallocationInMockSipp) // should be empty as there is nothing to allocate

}

// Mocked database operations for testing
type MockDBOperationsPrevAllocation struct{}

func (m *MockDBOperationsPrevAllocation) getCurrentAmountAllocated(tx *gorm.DB, wrapper string, clientID uint) (int64, error) {
	prevGetCurrentAmountAllocatedClientIdSipp = clientID
	prevGetCurrentAmountAllocatedWrapperSipp = wrapper
	return 6000000, nil // oversub
}

func (m *MockDBOperationsPrevAllocation) saveAllocation(tx *gorm.DB, allocation models.Allocation) error {
	prevallocationInMockSipp = allocation
	return nil
}

var prevGetCurrentAmountAllocatedClientIdIsa uint
var prevGetCurrentAmountAllocatedWrapperIsa string

var prevallocationInMockIsa models.Allocation

func TestProcessIsaAllocationWithPrevAllocationOverSub(t *testing.T) {
	testDB, _, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	if err != nil {
		t.Fatalf("Unable to create mock db: %v", err)
	}

	allocService := NewAllocationService()
	allocService.DbOps = &MockDBOperationsPrevAllocationIsaOverAllocate{}

	receipt := models.Receipt{}
	receipt.ID = 1
	deposit := models.Deposit{}
	deposit.ID = 1
	deposit.ClientID = 2
	account := models.Account{}
	account.ID = 1
	account.Wrapper = "ISA"
	account.PotID = 1

	tx := db.Begin()

	err = allocService.AlOps.processIsaAllocation(
		tx,
		&receipt,
		&deposit,
		&account,
		decimal.NewFromInt(1000),
		make(map[uint]*decimal.Decimal),
		allocService.DbOps,
	)

	assert.NoError(t, err)
	assert.Equal(t, "ISA", prevGetCurrentAmountAllocatedWrapperIsa, "wrapper name does not match")
	assert.Equal(t, uint(2), prevGetCurrentAmountAllocatedClientIdIsa, "client id does not match")

	assert.Equal(t, models.Allocation{}, prevallocationInMockIsa) // should be empty as there is nothing to allocate

}

// Mocked database operations for testing
type MockDBOperationsPrevAllocationIsaOverAllocate struct{}

func (m *MockDBOperationsPrevAllocationIsaOverAllocate) getCurrentAmountAllocated(tx *gorm.DB, wrapper string, clientID uint) (int64, error) {
	prevGetCurrentAmountAllocatedClientIdIsa = clientID
	prevGetCurrentAmountAllocatedWrapperIsa = wrapper
	return 2000000, nil // oversub
}

func (m *MockDBOperationsPrevAllocationIsaOverAllocate) saveAllocation(tx *gorm.DB, allocation models.Allocation) error {
	prevallocationInMockIsa = allocation
	return nil
}

// Test ISA with no previous allocation
var getCurrentAmountAllocatedClientIdIsa uint
var getCurrentAmountAllocatedWrapperIsa string

var allocationInMockIsa models.Allocation

func TestProcessIsaAllocationWithPrevAllocation(t *testing.T) {
	testDB, _, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	if err != nil {
		t.Fatalf("Unable to create mock db: %v", err)
	}

	allocService := NewAllocationService()
	allocService.DbOps = &MockDBOperationsIsa{}

	// Create sample data for testing
	receipt := models.Receipt{}
	receipt.ID = 1
	deposit := models.Deposit{}
	deposit.ID = 1
	deposit.ClientID = 2
	account := models.Account{}
	account.ID = 1
	account.Wrapper = "ISA"
	account.PotID = 1

	tx := db.Begin()

	err = allocService.AlOps.processIsaAllocation(
		tx,
		&receipt,
		&deposit,
		&account,
		decimal.NewFromInt(100000),
		make(map[uint]*decimal.Decimal),
		allocService.DbOps,
	)

	assert.NoError(t, err)
	assert.Equal(t, "ISA", getCurrentAmountAllocatedWrapperIsa, "wrapper name does not match")
	assert.Equal(t, uint(2), getCurrentAmountAllocatedClientIdIsa, "client id does not match")
	amount := decimal.NewFromInt(int64(allocationInMockIsa.Amount))
	assert.Equal(t, decimal.NewFromInt(100000), amount, "Values do not match")

}

// Mocked database operations for testing
type MockDBOperationsIsa struct{}

func (m *MockDBOperationsIsa) getCurrentAmountAllocated(tx *gorm.DB, wrapper string, clientID uint) (int64, error) {
	getCurrentAmountAllocatedClientIdIsa = clientID
	getCurrentAmountAllocatedWrapperIsa = wrapper
	return 0, nil
}

func (m *MockDBOperationsIsa) saveAllocation(tx *gorm.DB, allocation models.Allocation) error {
	allocationInMockIsa = allocation
	return nil
}

// test process allocation

var allocationInMockGia models.Allocation

func TestSaveGiaAllocation(t *testing.T) {
	testDB, _, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	if err != nil {
		t.Fatalf("Unable to create mock db: %v", err)
	}

	allocService := NewAllocationService()
	allocService.DbOps = &MockDBOperationsGia{}

	receipt := models.Receipt{}
	receipt.ID = 1
	deposit := models.Deposit{}
	deposit.ID = 1
	deposit.ClientID = 2
	account := models.Account{}
	account.ID = 1
	account.Wrapper = "ISA"
	account.PotID = 1

	tx := db.Begin()

	amount := decimal.NewFromInt(100000)

	// Perform the test
	err = allocService.AlOps.processGiaAllocation(
		tx,
		&receipt,
		&amount,
		&account,
		allocService.DbOps,
	)

	// Check for the expected errors or success
	assert.NoError(t, err)
	amountTest := decimal.NewFromInt(int64(allocationInMockGia.Amount))
	assert.Equal(t, decimal.NewFromInt(100000), amountTest, "Values do not match")

}

// Mocked database operations for testing
type MockDBOperationsGia struct{}

func (m *MockDBOperationsGia) getCurrentAmountAllocated(tx *gorm.DB, wrapper string, clientID uint) (int64, error) {
	return 0, nil
}

func (m *MockDBOperationsGia) saveAllocation(tx *gorm.DB, allocation models.Allocation) error {
	allocationInMockGia = allocation
	return nil
}

type MockAllocationService struct {
}

func (m MockAllocationService) processGiaAllocation(tx *gorm.DB, receipt *models.Receipt, amount *decimal.Decimal, account *models.Account, db DatabaseOperations) error {

	return nil
}

func (m MockAllocationService) processSIPPAllocation(tx *gorm.DB, receipt *models.Receipt, deposit *models.Deposit, account *models.Account, allocation decimal.Decimal, giaAllocationAmounts map[uint]*decimal.Decimal, db DatabaseOperations) error {
	return nil
}

func (m MockAllocationService) processIsaAllocation(tx *gorm.DB, receipt *models.Receipt, deposit *models.Deposit, account *models.Account, allocation decimal.Decimal, giaAllocationAmounts map[uint]*decimal.Decimal, db DatabaseOperations) error {
	return nil
}

// test happy path that all the allocation funcs are called
func TestAllocateReceipt(t *testing.T) {

	// mock the db
	testDB, mock, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	app.Http = &config.AppConfig{}
	app.Http.Database = config.DatabaseConfig{
		DB: db,
	}
	mock.ExpectBegin()
	idRow := sqlmock.NewRows([]string{"id"}).
		AddRow("1")

	mock.ExpectQuery("INSERT INTO \"receipts\"(.*)").WillReturnRows(idRow)

	now, _ := time.Parse(time.RFC3339, "2020-06-20T22:08:41Z")

	accountSipp := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "pot_id", "wrapper"}).AddRow("1", now, now, nil, "1", "SIPP")

	mock.ExpectQuery("SELECT \\* FROM \"accounts\"(.*)").WithArgs(int64(1), int64(1)).WillReturnRows(accountSipp)

	accountIsa := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "pot_id", "wrapper"}).AddRow("2", now, now, nil, "1", "ISA")

	mock.ExpectQuery("SELECT \\* FROM \"accounts\"(.*)").WithArgs(int64(2), int64(1)).WillReturnRows(accountIsa)

	accountGia := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "pot_id", "wrapper"}).AddRow("3", now, now, nil, "1", "ISA")

	mock.ExpectQuery("SELECT \\* FROM \"accounts\"(.*)").WithArgs(int64(3), int64(1)).WillReturnRows(accountGia)

	mock.ExpectCommit()

	// Create instances and dependencies needed for testing
	service := NewAllocationService()

	service.AlOps = MockAllocationService{}

	receipt := &models.Receipt{}
	receipt.ID = 1
	receipt.Amount = 5000000
	deposit := &models.Deposit{}
	deposit.Amount = 5000000
	deposit.ID = 1

	allocations := []models.ProposedAllocation{
		models.ProposedAllocation{AccountID: 1, Split: 0.25, DepositID: 1},
		models.ProposedAllocation{AccountID: 2, Split: 0.25, DepositID: 1},
		models.ProposedAllocation{AccountID: 3, Split: 0.50, DepositID: 1},
	}

	deposit.ProposedAllocation = allocations
	err = service.AllocateReceipt(receipt, deposit)
	if err != nil {
		t.Errorf("AllocateReceipt failed: %v", err)
	}

}

type MockAllocationServiceOverAllocate struct {
}

func (m MockAllocationServiceOverAllocate) processGiaAllocation(tx *gorm.DB, receipt *models.Receipt, amount *decimal.Decimal, account *models.Account, db DatabaseOperations) error {

	return nil
}

func (m MockAllocationServiceOverAllocate) processSIPPAllocation(tx *gorm.DB, receipt *models.Receipt, deposit *models.Deposit, account *models.Account, allocation decimal.Decimal, giaAllocationAmounts map[uint]*decimal.Decimal, db DatabaseOperations) error {
	amount := decimal.NewFromInt(20000)
	addOrCreateAllocationToGiaPot(amount, account.PotID, giaAllocationAmounts) // will cause the oversub block to be run
	return nil
}

func (m MockAllocationServiceOverAllocate) processIsaAllocation(tx *gorm.DB, receipt *models.Receipt, deposit *models.Deposit, account *models.Account, allocation decimal.Decimal, giaAllocationAmounts map[uint]*decimal.Decimal, db DatabaseOperations) error {
	return nil
}

// test over allocate so then a GIA needs to be created
func TestAllocateReceiptOverAllocate(t *testing.T) {

	testDB, mock, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	app.Http = &config.AppConfig{}
	app.Http.Database = config.DatabaseConfig{
		DB: db,
	}
	mock.ExpectBegin()
	idRow := sqlmock.NewRows([]string{"id"}).
		AddRow("1")

	mock.ExpectQuery("INSERT INTO \"receipts\"(.*)").WillReturnRows(idRow)

	now, _ := time.Parse(time.RFC3339, "2020-06-20T22:08:41Z")

	accountSipp := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "pot_id", "wrapper"}).AddRow("1", now, now, nil, "1", "SIPP")

	mock.ExpectQuery("SELECT \\* FROM \"accounts\"(.*)").WithArgs(int64(1), int64(1)).WillReturnRows(accountSipp)

	accountIsa := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "pot_id", "wrapper"}).AddRow("2", now, now, nil, "1", "ISA")

	mock.ExpectQuery("SELECT \\* FROM \"accounts\"(.*)").WithArgs(int64(2), int64(1)).WillReturnRows(accountIsa)

	mock.ExpectQuery("SELECT \\* FROM \"accounts\"(.*)").WithArgs(int64(1), "GIA", int64(1)).WillReturnError(gorm.ErrRecordNotFound)

	mock.ExpectQuery("INSERT INTO \"accounts\"(.*)").WillReturnRows(idRow)

	mock.ExpectCommit()

	// Create instances and dependencies needed for testing
	service := NewAllocationService()

	service.AlOps = MockAllocationServiceOverAllocate{}

	receipt := &models.Receipt{}
	receipt.ID = 1
	receipt.Amount = 5000000
	deposit := &models.Deposit{}
	deposit.Amount = 50000000
	deposit.ID = 1

	allocations := []models.ProposedAllocation{
		models.ProposedAllocation{AccountID: 1, Split: 0.50, DepositID: 1},
		models.ProposedAllocation{AccountID: 2, Split: 0.50, DepositID: 1},
	}

	deposit.ProposedAllocation = allocations
	err = service.AllocateReceipt(receipt, deposit)
	if err != nil {
		t.Errorf("AllocateReceipt failed: %v", err)
	}

}

func TestAllocateReceiptExceptionCreatingReceipt(t *testing.T) {

	// mock the db
	testDB, mock, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	app.Http = &config.AppConfig{}
	app.Http.Database = config.DatabaseConfig{
		DB: db,
	}
	mock.ExpectBegin()

	errMsg := "Error creating receipt"

	mock.ExpectQuery("INSERT INTO \"receipts\"(.*)").WillReturnError(errors.New(errMsg))

	// Create instances and dependencies needed for testing
	service := NewAllocationService()

	service.AlOps = MockAllocationService{}

	receipt := &models.Receipt{}
	receipt.ID = 1
	receipt.Amount = 5000000
	deposit := &models.Deposit{}
	deposit.Amount = 5000000
	deposit.ID = 1

	allocations := []models.ProposedAllocation{
		models.ProposedAllocation{AccountID: 1, Split: 0.25, DepositID: 1},
		models.ProposedAllocation{AccountID: 2, Split: 0.25, DepositID: 1},
		models.ProposedAllocation{AccountID: 3, Split: 0.50, DepositID: 1},
	}

	deposit.ProposedAllocation = allocations
	err = service.AllocateReceipt(receipt, deposit)

	assert.Error(t, err, errMsg)

}

func TestAllocateReceiptExceptionLoadingAccount(t *testing.T) {

	// mock the db
	testDB, mock, _ := sqlmock.New()

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})

	app.Http = &config.AppConfig{}
	app.Http.Database = config.DatabaseConfig{
		DB: db,
	}
	mock.ExpectBegin()
	idRow := sqlmock.NewRows([]string{"id"}).
		AddRow("1")

	mock.ExpectQuery("INSERT INTO \"receipts\"(.*)").WillReturnRows(idRow)

	errMsg := "Error loading account"

	mock.ExpectQuery("SELECT \\* FROM \"accounts\"(.*)").WithArgs(int64(1), int64(1)).WillReturnError(errors.New("Error loading account"))

	// Create instances and dependencies needed for testing
	service := NewAllocationService()

	service.AlOps = MockAllocationService{}

	receipt := &models.Receipt{}
	receipt.ID = 1
	receipt.Amount = 5000000
	deposit := &models.Deposit{}
	deposit.Amount = 5000000
	deposit.ID = 1

	allocations := []models.ProposedAllocation{
		models.ProposedAllocation{AccountID: 1, Split: 0.25, DepositID: 1},
		models.ProposedAllocation{AccountID: 2, Split: 0.25, DepositID: 1},
		models.ProposedAllocation{AccountID: 3, Split: 0.50, DepositID: 1},
	}

	deposit.ProposedAllocation = allocations
	err = service.AllocateReceipt(receipt, deposit)

	assert.Error(t, err, errMsg)

}
