package main

import (
	"ajbell.co.uk/app"
	"ajbell.co.uk/migrations"
	"ajbell.co.uk/rest/routes"
	"flag"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/storage/memory/v2"
	"time"
)

func main() {

	configFile := flag.String("config", "config.yml", "User Config file from user")

	app.Load(*configFile)

	app.Http.Server.Setup()

	migrations.Migrate(app.Http.Database.DB)

	// ensures idempotency
	store := memory.New(memory.Config{
		GCInterval: 30 * time.Minute,
	})
	app.Http.Server.App.Use(idempotency.New(idempotency.Config{
		Lifetime:  30 * time.Minute,
		KeyHeader: "X-Idempotency-Key",
		Storage:   store,
	}))

	routes.LoadRoutes(app.Http.Server.App)

	app.Http.Route404()

	err := app.Http.Server.Listen(":3000")
	if err != nil {
		panic(err)
	}

}
