//go:build ignore

// Example Go bot.
//
// Run:
//
//	BOT_ID=go-bot API_URL=http://localhost:8080 API_KEY=dev go run example_bot.go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/sdk/go/botclient"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	client, err := botclient.New(botclient.Config{
		BotID:  getenv("BOT_ID", "go-bot"),
		APIURL: getenv("API_URL", "http://localhost:8080"),
		APIKey: getenv("API_KEY", "dev"),
	})
	if err != nil {
		fmt.Println("failed to create client:", err)
		os.Exit(1)
	}
	defer client.Stop()

	ctx := context.Background()
	symbol := "BTC-USD"
	done := make(chan struct{})

	fmt.Printf("Bot %s running on %s...\n", client.BotID(), symbol)

	client.Subscribe(ctx, symbol, func(ticker botclient.Ticker) {
		fmt.Printf("[%s] bid=%s ask=%s spread=%s\n",
			symbol, ticker.BidPrice, ticker.AskPrice, ticker.Spread)

		if ticker.AskPrice == "" {
			return
		}

		// Use PriceSub — never float64 arithmetic
		price := botclient.PriceSub(ticker.AskPrice, "1.00")

		resp, err := client.PlaceOrder(ctx, botclient.OrderRequest{
			Symbol:   symbol,
			Side:     botclient.SideBuy,
			Type:     botclient.OrderTypeLimit,
			Price:    price,
			Quantity: "0.01",
		})
		if err != nil {
			fmt.Println("  → order failed:", err)
			return
		}
		fmt.Printf("  → placed buy at %s, order=%s\n", price, resp.Order.ID)
		client.Stop()
		close(done)
	})

	<-done
}
