// Package api provides the REST and WebSocket entry points for the trading platform.
package api

import (
	"encoding/json"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/matching"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
	"github.com/gofiber/contrib/v3/websocket" // This is the v3 version

	// "github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

// Handler encapsulates the dependencies required for API processing.
type Handler struct {
	registry *matching.Registry
}

// NewHandler initializes a new API handler with the provided matching registry.
func NewHandler(registry *matching.Registry) *Handler {
	return &Handler{registry: registry}
}

// RegisterRoutes defines the API surface area for the application.
func (h *Handler) RegisterRoutes(app *fiber.App) {
	v1 := app.Group("/api/v1")

	// REST Endpoints
	v1.Get("/markets", h.listMarkets)
	v1.Get("/markets/:symbol/orderbook", h.getOrderBook)
	v1.Post("/orders", h.submitOrder)
	v1.Delete("/orders/:symbol/:orderID", h.cancelOrder)

	// WebSocket Route with Fiber v3 Upgrade Middleware
	// We manually check the "Upgrade" header to ensure compatibility with v3's Ctx interface.
	app.Use("/ws/:symbol", func(c fiber.Ctx) error {
		if c.Get("Upgrade") == "websocket" {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket Connection Handler
	app.Get("/ws/:symbol", websocket.New(h.orderBookFeed))
}

// listMarkets returns all active trading pairs in the registry.
func (h *Handler) listMarkets(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"symbols": h.registry.Symbols()})
}

// getOrderBook retrieves a depth-limited snapshot of the order book for a symbol.
func (h *Handler) getOrderBook(c fiber.Ctx) error {
	symbol := c.Params("symbol")

	// Engineering Note: In Fiber v3, QueryInt is replaced by the Bind API.
	// We define a struct to capture query parameters with default values.
	query := struct {
		Depth int `query:"depth"`
	}{Depth: 20}

	if err := c.Bind().Query(&query); err != nil {
		log.Warn().Err(err).Msg("Failed to bind query parameters, using defaults")
	}

	snap, err := h.registry.Snapshot(symbol, query.Depth)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(snap)
}

// submitOrder handles incoming LIMIT and MARKET orders.
func (h *Handler) submitOrder(c fiber.Ctx) error {
	var req struct {
		BotID    string           `json:"bot_id"`
		Symbol   string           `json:"symbol"`
		Side     models.Side      `json:"side"`
		Type     models.OrderType `json:"type"`
		Price    string           `json:"price"`
		Quantity string           `json:"quantity"`
	}

	// Use v3 Bind().JSON() for request body parsing.
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Basic Validation
	if req.BotID == "" || req.Symbol == "" || req.Quantity == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bot_id, symbol, and quantity are required"})
	}

	quantity, err := decimal.FromString(req.Quantity)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid quantity: " + err.Error()})
	}

	var order *models.Order
	switch req.Type {
	case models.OrderTypeLimit:
		if req.Price == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "limit order requires price"})
		}
		price, err := decimal.FromString(req.Price)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid price: " + err.Error()})
		}
		order = models.NewLimitOrder(req.BotID, req.Symbol, req.Side, price, quantity)
	case models.OrderTypeMarket:
		order = models.NewMarketOrder(req.BotID, req.Symbol, req.Side, quantity)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "type must be LIMIT or MARKET"})
	}

	// Pass the order to the matching engine registry.
	result, err := h.registry.Submit(order)
	if err != nil {
		log.Error().Err(err).Str("bot_id", req.BotID).Msg("order submission failed")
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	// Success Logging
	log.Info().
		Str("order_id", order.ID).
		Str("symbol", order.Symbol).
		Str("side", string(order.Side)).
		Str("price", order.Price.String()).
		Str("quantity", order.Quantity.String()).
		Int("trades", len(result.Trades)).
		Msg("order processed")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"order":  order,
		"trades": result.Trades,
	})
}

// cancelOrder attempts to remove an active order from the book.
func (h *Handler) cancelOrder(c fiber.Ctx) error {
	symbol := c.Params("symbol")
	orderID := c.Params("orderID")

	cancelled, err := h.registry.Cancel(symbol, orderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cancelled)
}

// orderBookFeed streams live order book updates to a connected client.
func (h *Handler) orderBookFeed(c *websocket.Conn) {
	symbol := c.Params("symbol")
	log.Info().Str("symbol", symbol).Msg("WS client connected")

	// Engineering Note: We use a ticker to prevent overwhelming the client
	// with every single micro-change, batching updates every 500ms.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	defer c.Close()

	for range ticker.C {
		snap, err := h.registry.Snapshot(symbol, 10)
		if err != nil {
			// If symbol doesn't exist, stop the feed.
			log.Warn().Str("symbol", symbol).Msg("Market not found, closing feed")
			return
		}

		data, err := json.Marshal(snap)
		if err != nil {
			continue
		}

		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Info().Str("symbol", symbol).Msg("WS client disconnected")
			return
		}
	}
}
