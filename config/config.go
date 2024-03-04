package config

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
)

type AppConfig struct {
	Database   DatabaseConfig `yaml:"db"`
	Server     ServerConfig
	ConfigFile string
}

func (cfg *AppConfig) Route404() {
	cfg.Server.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Page not found")
	})
}

func (cfg *AppConfig) Setup() {
	if err := cleanenv.ReadConfig(cfg.ConfigFile, cfg); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	cfg.Server.Setup()
	cfg.LoadComponents()

}

func (cfg *AppConfig) LoadComponents() {
	cfg.Database.Setup()
}
