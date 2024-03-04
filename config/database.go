package config

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
	"time"
)

type DatabaseDriver struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"5432"`
	Username string `yaml:"username" env:"DB_USER" env-default:"localhost"`
	Password string `yaml:"password" env:"DB_PASS" env-default:"postgres"`
	DBName   string `yaml:"db_name" env:"DB_NAME" env-default:"breezy"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"postgres"`
}

type DatabaseConfig struct {
	*gorm.DB
	Driver DatabaseDriver `yaml:"postgres"`
}

func (d *DatabaseConfig) Setup() {
	var err error

	connectionString := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s", d.Driver.Host, d.Driver.Port, d.Driver.Username, d.Driver.DBName, d.Driver.Password)
	d.DB, err = gorm.Open(postgres.Open(connectionString), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		fmt.Println(d.Driver)
		panic(err)
	}
	err = d.DB.Use(
		dbresolver.Register(dbresolver.Config{}).
			SetConnMaxLifetime(24 * time.Hour).
			SetMaxIdleConns(100).
			SetMaxOpenConns(100),
	)
	if err != nil {
		fmt.Println(d.Driver)
		panic(err)
	}
}
