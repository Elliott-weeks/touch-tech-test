package service

import (
	"ajbell.co.uk/app"
	"ajbell.co.uk/pkg/models"
	"fmt"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"log"
)

const yearlyIsaLimit = 2000000 // 20 thousand pounds in pence

const yearlyPensionLimit = 6000000 // 60 thousand pounds in pence

type AllocationService struct {
	GiaAllocations map[uint]*decimal.Decimal
	AlOps          AllocateOperations
	DbOps          DatabaseOperations
}

type AllocateOperations interface {
	processGiaAllocation(tx *gorm.DB, receipt *models.Receipt, amount *decimal.Decimal, account *models.Account, db DatabaseOperations) error
	processSIPPAllocation(tx *gorm.DB, receipt *models.Receipt, deposit *models.Deposit, account *models.Account, allocation decimal.Decimal, giaAllocationAmounts map[uint]*decimal.Decimal, db DatabaseOperations) error
	processIsaAllocation(tx *gorm.DB, receipt *models.Receipt, deposit *models.Deposit, account *models.Account, allocation decimal.Decimal, giaAllocationAmounts map[uint]*decimal.Decimal, db DatabaseOperations) error
}
type DatabaseOperations interface {
	getCurrentAmountAllocated(tx *gorm.DB, wrapper string, clientID uint) (int64, error)
	saveAllocation(tx *gorm.DB, allocation models.Allocation) error
}

type Allocate interface {
	AllocateReceipt(receipt *models.Receipt, deposit *models.Deposit) error
}

type DbOps struct {
}

type AllocateOps struct {
}

func NewAllocationService() *AllocationService {
	return &AllocationService{
		GiaAllocations: make(map[uint]*decimal.Decimal),
		AlOps:          &AllocateOps{},
		DbOps:          &DbOps{},
	}
}

func (c *AllocationService) AllocateReceipt(receipt *models.Receipt, deposit *models.Deposit) error {
	tx := app.Http.Database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered: %v\n", r)
			tx.Rollback()
		}
	}()
	if err := tx.Create(&receipt).Error; err != nil {
		log.Printf("Error creating receipt: %v\n", err)
		tx.Rollback()
		return errors.Wrap(err, "failed to create receipt")
	}

	giaAllocationAmounts := make(map[uint]*decimal.Decimal)

	leftOverFromRemainder := decimal.NewFromInt(0)

	for i, allocation := range deposit.ProposedAllocation {
		account := &models.Account{}
		if err := tx.First(&account, allocation.AccountID).Error; err != nil {
			log.Printf("Error fetching account: %v\n", err)
			tx.Rollback()
			return err
		}

		allocate, remainder := calculateAllocation(receipt.Amount, allocation.Split)

		leftOverFromRemainder = leftOverFromRemainder.Add(remainder)

		if i == len(deposit.ProposedAllocation)-1 && !leftOverFromRemainder.IsZero() {
			allocate = allocate.Add(leftOverFromRemainder.Round(0))
		}

		switch account.Wrapper {
		case "SIPP":
			err := c.AlOps.processSIPPAllocation(tx, receipt, deposit, account, allocate, giaAllocationAmounts, c.DbOps)
			if err != nil {
				log.Printf("Error processing SIPP allocations: %v\n", err)
				tx.Rollback()
				return errors.Wrap(err, "failed processing SIPP allocation")
			}
			break
		case "ISA":
			err := c.AlOps.processIsaAllocation(tx, receipt, deposit, account, allocate, giaAllocationAmounts, c.DbOps)
			if err != nil {
				log.Printf("Error processing ISA allocations: %v\n", err)
				tx.Rollback()
				return errors.Wrap(err, "failed processing ISA allocation")
			}
			break
		default:
			addOrCreateAllocationToGiaPot(allocate, account.PotID, giaAllocationAmounts)
		}
	}

	for k, v := range giaAllocationAmounts {
		gia, err := safeCreateGia(tx, k)
		if err != nil {
			log.Printf("Error creating GIA: %v\n", err)
			tx.Rollback()
			return errors.Wrap(err, "Error creating GIA")
		}

		log.Printf("Debug: GIA Allocation Amounts: %v\n", v)
		err = c.AlOps.processGiaAllocation(tx, receipt, v, &gia, c.DbOps)
		if err != nil {
			log.Printf("Error processing GIA allocations: %v\n", err)
			tx.Rollback()
			return errors.Wrap(err, "failed processing GIA allocations")
		}

	}

	if err := tx.Commit().Error; err != nil {
		fmt.Printf("Error committing transaction: %v\n", err)
		tx.Rollback()
		return errors.Wrap(err, "failed during committing db transaction")
	}

	return nil

}

func safeCreateGia(tx *gorm.DB, potId uint) (models.Account, error) {
	giaAccount := models.Account{}

	err := tx.First(&giaAccount, "pot_id = ? and wrapper = ?", potId, "GIA").Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			giaAccount.PotID = potId
			giaAccount.Wrapper = "GIA"
			if err := tx.Create(&giaAccount).Error; err != nil {
				return giaAccount, err
			}
		} else { // another error occurred
			return models.Account{}, err
		}
	}
	return giaAccount, nil
}

func (c *AllocateOps) processGiaAllocation(tx *gorm.DB, receipt *models.Receipt, amount *decimal.Decimal, account *models.Account, db DatabaseOperations) error {
	allocation := models.Allocation{Amount: uint(amount.IntPart()), ReceiptID: receipt.ID, AccountID: account.ID}
	return db.saveAllocation(tx, allocation)
}

func (c *AllocateOps) processSIPPAllocation(tx *gorm.DB, receipt *models.Receipt, deposit *models.Deposit, account *models.Account, allocation decimal.Decimal, giaAllocationAmounts map[uint]*decimal.Decimal, db DatabaseOperations) error {
	currentAmountAllocated, err := db.getCurrentAmountAllocated(tx, account.Wrapper, deposit.ClientID)
	if err != nil {
		return err
	}

	amountAfterAllocation := decimal.NewFromInt(currentAmountAllocated).Add(allocation)

	isUnderSubscribed := amountAfterAllocation.LessThanOrEqual(decimal.NewFromInt(yearlyPensionLimit))

	if isUnderSubscribed {
		allocation := models.Allocation{Amount: uint(allocation.IntPart()), ReceiptID: receipt.ID, AccountID: account.ID}
		return db.saveAllocation(tx, allocation)
	}
	toBeAllocatedToGia := amountAfterAllocation.Sub(decimal.NewFromInt(yearlyPensionLimit))
	addOrCreateAllocationToGiaPot(toBeAllocatedToGia, account.PotID, giaAllocationAmounts)
	allocate := allocation.Sub(toBeAllocatedToGia)
	if !allocate.IsZero() {
		allocation := models.Allocation{Amount: uint(allocate.IntPart()), ReceiptID: receipt.ID, AccountID: account.ID}
		return db.saveAllocation(tx, allocation)

	}
	return nil
}

func (c *DbOps) saveAllocation(tx *gorm.DB, allocation models.Allocation) error {
	if err := tx.Create(&allocation).Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func (c *AllocateOps) processIsaAllocation(tx *gorm.DB, receipt *models.Receipt, deposit *models.Deposit, account *models.Account, allocation decimal.Decimal, giaAllocationAmounts map[uint]*decimal.Decimal, db DatabaseOperations) error {

	currentAmountAllocated, err := db.getCurrentAmountAllocated(tx, account.Wrapper, deposit.ClientID)
	if err != nil {
		return err
	}

	amountAfterAllocation := decimal.NewFromInt(currentAmountAllocated).Add(allocation)

	isUnderSubscribed := amountAfterAllocation.LessThanOrEqual(decimal.NewFromInt(yearlyIsaLimit))

	if isUnderSubscribed {
		allocation := models.Allocation{Amount: uint(allocation.IntPart()), ReceiptID: receipt.ID, AccountID: account.ID}
		return db.saveAllocation(tx, allocation)
	}
	toBeAllocatedToGia := amountAfterAllocation.Sub(decimal.NewFromInt(yearlyIsaLimit))
	addOrCreateAllocationToGiaPot(toBeAllocatedToGia, account.PotID, giaAllocationAmounts)
	allocate := allocation.Sub(toBeAllocatedToGia)
	if !allocate.IsZero() {
		allocation := models.Allocation{Amount: uint(allocate.IntPart()), ReceiptID: receipt.ID, AccountID: account.ID}
		return db.saveAllocation(tx, allocation)
	}

	return nil

}

func (c *DbOps) getCurrentAmountAllocated(tx *gorm.DB, wrapper string, clientID uint) (int64, error) {
	var currentAmountAllocated int64
	result := tx.Raw("SELECT COALESCE(SUM(al.amount), 0) FROM clients c "+
		"LEFT JOIN pots p ON p.client_id = c.id "+
		"LEFT JOIN accounts a ON a.pot_id = p.id "+
		"LEFT JOIN allocations al ON al.account_id = a.id "+
		"WHERE a.wrapper = ? AND a.deleted_at IS NULL "+
		"AND al.deleted_at IS NULL AND p.deleted_at IS NULL"+
		" AND c.id = ?", wrapper, clientID)

	if err := result.Scan(&currentAmountAllocated).Error; err != nil {
		log.Printf("Error scanning current amount allocation result:%v\n", err)
		return 0, err
	}

	return currentAmountAllocated, nil

}

func addOrCreateAllocationToGiaPot(amount decimal.Decimal, potID uint, giaAllocations map[uint]*decimal.Decimal) {
	val, ok := giaAllocations[potID]
	// If the key exists
	if ok {
		sum := val.Add(amount)
		giaAllocations[potID] = &sum
	} else {
		giaAllocations[potID] = &amount
	}

}

func calculateAllocation(amount uint, split float32) (allocation decimal.Decimal, remainder decimal.Decimal) {
	amountDecimal := decimal.NewFromInt(int64(amount))
	splitDecimal := decimal.NewFromFloat(float64(split))

	toBeAllocated := amountDecimal.Mul(splitDecimal)
	roundedAllocation := toBeAllocated.Floor()
	remain := amountDecimal.Mul(splitDecimal).Sub(roundedAllocation).Abs()

	return roundedAllocation, remain
}
