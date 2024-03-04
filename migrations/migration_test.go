package migrations

import (
	"gorm.io/driver/postgres"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
)

func TestMigrate(t *testing.T) {
	// Create a new GORM DB connected to the mock database
	testDB, _, _ := sqlmock.New()
	//handle error

	// uses "gorm.io/driver/postgres" library
	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 testDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("Error creating GORM DB: %v", err)
	}

	// ensure the panic is not run
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	// ideal world would check the table creation
	Migrate(db)
}
