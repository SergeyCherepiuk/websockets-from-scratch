package main

import (
	"log"
	"strings"

	"github.com/SergeyCherepiuk/websockets-test/websockets"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	app := fiber.New()
	app.Use(logger.New())

	app.Get("/chat", func(c *fiber.Ctx) error {
		key := strings.TrimSpace(c.Get("Sec-WebSocket-Key", ""))
		if key == "" {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		conn := websockets.NewConnection(key)
		c.Set("Upgrade", "websocket")
		c.Set("Connection", "Upgrade")
		c.Set("Sec-WebSocket-Accept", conn.GenerateKey())

		c.Context().Hijack(conn.HandleCommunication)
		return c.SendStatus(fiber.StatusSwitchingProtocols)
	})

	app.Listen(":8080")
}
