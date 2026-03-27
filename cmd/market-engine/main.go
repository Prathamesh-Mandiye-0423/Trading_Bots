package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/api"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/events"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/matching"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg(".env file not found, using environment variables")
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Connect to Redpanda
	brokers := []string{"localhost:19092"}
	publisher, err := events.NewPublisher(brokers)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redpanda")
	}
	defer publisher.Close()

	// Build market registry
	registry := matching.NewRegistry(4096)
	for _, sym := range []string{"BTC-USD", "ETH-USD", "SOL-USD"} {
		if err := registry.AddMarket(sym); err != nil {
			log.Fatal().Err(err).Str("symbol", sym).Msg("failed to add market")
		}
		log.Info().Str("symbol", sym).Msg("market registered")
	}

	// Start trade event pipeline: matching engine → Redpanda
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pipeTradeEvents(ctx, registry, publisher)

	// Build Fiber app
	app := fiber.New(fiber.Config{AppName: "Market Engine v0.1"})
	app.Use(recover.New())
	app.Use(logger.New())

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	h := api.NewHandler(registry)
	h.RegisterRoutes(app)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		port := os.Getenv("SERVER_PORT")
		if port == "" {
			port = "8080"
		}
		log.Info().Str("port", port).Msg("market engine listening")
		if err := app.Listen(":" + port); err != nil {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-quit
	log.Info().Msg("shutting down...")
	cancel()
	app.Shutdown()
}

// pipeTradeEvents drains the matching engine's trade channel
// and publishes each trade + order events to Redpanda
func pipeTradeEvents(ctx context.Context, registry *matching.Registry, pub *events.Publisher) {
	for {
		select {
		case <-ctx.Done():
			return
		case trade, ok := <-registry.TradeChan():
			if !ok {
				return
			}
			publishTrade(ctx, pub, trade)
		}
	}
}

func publishTrade(ctx context.Context, pub *events.Publisher, trade *models.Trade) {
	// Publish the trade execution event
	tradeEvent := events.TradeEventFromModel(trade)
	if err := pub.PublishTrade(ctx, tradeEvent); err != nil {
		log.Error().Err(err).Str("trade_id", trade.ID).Msg("failed to publish trade event")
		return
	}

	log.Info().
		Str("trade_id", trade.ID).
		Str("symbol", trade.Symbol).
		Str("price", trade.Price.String()).
		Str("quantity", trade.Quantity.String()).
		Str("notional", trade.Notional.String()).
		Msg("TRADE PUBLISHED")
}
