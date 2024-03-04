package migrations

import (
	"ajbell.co.uk/pkg/models"
	"gorm.io/gorm"
	"log"
)

func Migrate(db *gorm.DB) {
	log.Println("Initiating migration...")
	err := db.Migrator().AutoMigrate(
		&models.Client{},
		&models.Pot{},
		&models.Account{},
		&models.Deposit{},
		&models.Receipt{},
		&models.ProposedAllocation{},
		&models.Allocation{},
	)
	if err != nil {
		panic(err)
	}
	log.Println("Migration Completed...")
}
