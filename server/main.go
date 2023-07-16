package main

import (
	"log"
	"net"
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
		clientKey := strings.TrimSpace(c.Get("Sec-WebSocket-Key", ""))
		if clientKey == "" {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		c.Set("Upgrade", "websocket")
		c.Set("Connection", "Upgrade")
		c.Set("Sec-WebSocket-Accept", websockets.GenerateKey(clientKey))

		c.Context().Hijack(func(hijackedConn net.Conn) {
			conn := websockets.NewConnection(hijackedConn, clientKey)
			conn.HandleConnection()
		})

		return c.SendStatus(fiber.StatusSwitchingProtocols)
	})

	app.Listen(":8080")
}
