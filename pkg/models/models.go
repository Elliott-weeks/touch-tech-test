package models

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type Client struct {
	gorm.Model
	Name     string
	Pots     []Pot     `gorm:"foreignKey:ClientID"`
	Deposits []Deposit `gorm:"foreignKey:ClientID"`
}

type Pot struct {
	gorm.Model
	ClientID uint
	Name     string
	Accounts []Account `gorm:"foreignKey:PotID"`
}

type Account struct {
	gorm.Model
	PotID   uint
	Wrapper string // SIPP, GIA, ISA
}

type Deposit struct {
	gorm.Model
	ClientID           uint                 `json:"client_id" validate:"required"`
	Amount             uint                 `json:"amount" validate:"required"` // amount is always in pennies
	Receipts           []Receipt            `json:"receipts" gorm:"foreignKey:DepositID"`
	ProposedAllocation []ProposedAllocation `json:"proposed_allocation,omitempty" gorm:"foreignKey:DepositID" validate:"required,dive,required"`
}

type Receipt struct {
	gorm.Model
	DepositID   uint
	Amount      uint           `json:"amount" validate:"required"` // amount is always in pennies
	DeletedAt   gorm.DeletedAt `json:"-"`
	Allocations []Allocation   `gorm:"foreignKey:ReceiptID"`
}

type ProposedAllocation struct {
	gorm.Model
	AccountID uint    `json:"account_id" validate:"required"`
	Split     float32 `json:"split" validate:"required"`
	DepositID uint
}

type Allocation struct {
	gorm.Model
	ReceiptID uint
	AccountID uint
	Amount    uint
}

var validate = validator.New()

type ErrorResponse struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value,omitempty"`
}

func ValidateStruct[T any](payload T) []*ErrorResponse {
	var errors []*ErrorResponse
	err := validate.Struct(payload)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.Field = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errors = append(errors, &element)
		}
	}
	return errors
}
