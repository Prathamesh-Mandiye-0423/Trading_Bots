package api

import (
	"strconv"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/db"
	"github.com/gofiber/fiber/v3"
)

// RegisterDBRoutes adds the historical data endpoints.
// Called from main.go alongside RegisterRoutes.
func RegisterDBRoutes(app *fiber.App) {
	v1 := app.Group("/api/v1")

	// Trade history
	v1.Get("/trades/:symbol", getRecentTrades)
	v1.Get("/bots/:botID/trades", getBotTrades)
	v1.Get("/bots/:botID/pnl", getBotPnL)

	// Candles
	v1.Get("/candles/:symbol", getCandles)

	// ML history
	v1.Get("/ml/:symbol/history", getMLHistory)
}

// GET /api/v1/trades/:symbol?limit=50
func getRecentTrades(c fiber.Ctx) error {
	symbol := c.Params("symbol")
	limit := fiber.Query[int](c, "limit", 50)

	trades, err := db.GetRecentTrades(c.Context(), symbol, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": err.Error()},
		)
	}
	return c.JSON(fiber.Map{"trades": trades, "symbol": symbol})
}

// GET /api/v1/bots/:botID/trades?limit=50
func getBotTrades(c fiber.Ctx) error {
	botID := c.Params("botID")
	limit := fiber.Query[int](c, "limit", 50)

	trades, err := db.GetBotTrades(c.Context(), botID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": err.Error()},
		)
	}
	return c.JSON(fiber.Map{"trades": trades, "bot_id": botID})
}

// GET /api/v1/bots/:botID/pnl
func getBotPnL(c fiber.Ctx) error {
	botID := c.Params("botID")

	pnl, err := db.GetBotPnL(c.Context(), botID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": err.Error()},
		)
	}
	return c.JSON(pnl)
}

// GET /api/v1/candles/:symbol?from=2024-01-01T00:00:00Z&to=2024-01-02T00:00:00Z
func getCandles(c fiber.Ctx) error {
	symbol := c.Params("symbol")

	from := time.Now().Add(-1 * time.Hour) // default: last 1 hour
	to := time.Now()

	if f := c.Query("from"); f != "" {
		if t, err := time.Parse(time.RFC3339, f); err == nil {
			from = t
		}
	}
	if t := c.Query("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = parsed
		}
	}

	candles, err := db.GetCandles(c.Context(), symbol, from, to)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": err.Error()},
		)
	}
	return c.JSON(fiber.Map{"candles": candles, "symbol": symbol})
}

// GET /api/v1/ml/:symbol/history?limit=100
func getMLHistory(c fiber.Ctx) error {
	symbol := c.Params("symbol")
	limit := fiber.Query[int](c, "limit", 100)

	history, err := db.GetMLHistory(c.Context(), symbol, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{"error": err.Error()},
		)
	}
	return c.JSON(fiber.Map{"history": history, "symbol": symbol})
}

// keep strconv import used
var _ = strconv.Itoa
