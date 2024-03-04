package config

import "github.com/gofiber/fiber/v2"

type ServerConfig struct {
	*fiber.App
}

func (s *ServerConfig) Setup() {
	s.App = fiber.New(fiber.Config{
		Concurrency: 256 * 1024 * 1024,
	})
}
