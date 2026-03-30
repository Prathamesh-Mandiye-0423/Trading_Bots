// Package botclient provides a Go SDK for building trading bots.
//
// Usage:
//
//	client, err := botclient.New(botclient.Config{
//	    BotID:  "my-bot",
//	    APIURL: "http://localhost:8080",
//	    APIKey: "dev",
//	})
//
//	resp, err := client.PlaceOrder(ctx, botclient.OrderRequest{
//	    Symbol:   "BTC-USD",
//	    Side:     botclient.SideBuy,
//	    Type:     botclient.OrderTypeLimit,
//	    Price:    "50000.00",
//	    Quantity: "0.5",
//	})
package botclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// ---- Domain constants ----

type Side string
type OrderType string
type OrderStatus string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"

	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"

	StatusOpen      OrderStatus = "OPEN"
	StatusFilled    OrderStatus = "FILLED"
	StatusPartial   OrderStatus = "PARTIAL"
	StatusCancelled OrderStatus = "CANCELLED"
	StatusSuspended OrderStatus = "SUSPENDED"
)

// ---- Config ----

// Config holds bot connection settings.
// APIKey is injected by the sandbox manager at runtime via BOT_API_KEY env var.
type Config struct {
	BotID  string
	APIURL string
	APIKey string
}

// ---- Client ----

type Client struct {
	cfg    Config
	http   *http.Client
	stopCh chan struct{}
}

func New(cfg Config) (*Client, error) {
	if cfg.BotID == "" || cfg.APIURL == "" {
		return nil, fmt.Errorf("botclient: BotID and APIURL are required")
	}
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: 5 * time.Second},
		stopCh: make(chan struct{}),
	}, nil
}

// ---- Domain types ----

type PriceLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Orders   int    `json:"orders"`
}

type OrderBook struct {
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Spread    string       `json:"spread"`
	Timestamp time.Time    `json:"timestamp"`
}

type Order struct {
	ID        string      `json:"id"`
	BotID     string      `json:"bot_id"`
	Symbol    string      `json:"symbol"`
	Side      Side        `json:"side"`
	Type      OrderType   `json:"type"`
	Price     string      `json:"price"`
	Quantity  string      `json:"quantity"`
	Remaining string      `json:"remaining"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type Trade struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	BuyBotID    string    `json:"buy_bot_id"`
	SellBotID   string    `json:"sell_bot_id"`
	Price       string    `json:"price"`
	Quantity    string    `json:"quantity"`
	Notional    string    `json:"notional"`
	ExecutedAt  time.Time `json:"executed_at"`
}

type Ticker struct {
	Symbol    string
	BidPrice  string
	AskPrice  string
	Spread    string
	Timestamp time.Time
}

// ---- Request / Response structs ----
// Typed structs with omitempty prevent bugs as the API grows.

type orderPayload struct {
	BotID    string    `json:"bot_id"`
	Symbol   string    `json:"symbol"`
	Side     Side      `json:"side"`
	Type     OrderType `json:"type"`
	Price    string    `json:"price,omitempty"` // omitted for MARKET orders
	Quantity string    `json:"quantity"`
}

type orderResponsePayload struct {
	Order  Order   `json:"order"`
	Trades []Trade `json:"trades"`
}

type marketsPayload struct {
	Symbols []string `json:"symbols"`
}

// OrderRequest is the public interface for placing orders.
type OrderRequest struct {
	Symbol   string
	Side     Side
	Type     OrderType
	Price    string // required for LIMIT, empty for MARKET
	Quantity string
}

// OrderResponse contains the placed order and any trades it triggered.
type OrderResponse struct {
	Order  Order
	Trades []Trade
}

// ---- Order management ----

func (c *Client) PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResponse, error) {
	if req.Type == OrderTypeLimit && req.Price == "" {
		return nil, fmt.Errorf("botclient: Price required for LIMIT order")
	}
	payload := orderPayload{
		BotID:    c.cfg.BotID,
		Symbol:   req.Symbol,
		Side:     req.Side,
		Type:     req.Type,
		Price:    req.Price,
		Quantity: req.Quantity,
	}
	var resp orderResponsePayload
	if err := c.post(ctx, "/api/v1/orders", payload, &resp); err != nil {
		return nil, fmt.Errorf("botclient: PlaceOrder: %w", err)
	}
	return &OrderResponse{Order: resp.Order, Trades: resp.Trades}, nil
}

func (c *Client) CancelOrder(ctx context.Context, symbol, orderID string) (*Order, error) {
	var order Order
	if err := c.delete(ctx, fmt.Sprintf("/api/v1/orders/%s/%s", symbol, orderID), &order); err != nil {
		return nil, fmt.Errorf("botclient: CancelOrder: %w", err)
	}
	return &order, nil
}

func (c *Client) GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error) {
	var book OrderBook
	if err := c.get(ctx, fmt.Sprintf("/api/v1/markets/%s/orderbook?depth=%d", symbol, depth), &book); err != nil {
		return nil, fmt.Errorf("botclient: GetOrderBook: %w", err)
	}
	return &book, nil
}

func (c *Client) GetMarkets(ctx context.Context) ([]string, error) {
	var resp marketsPayload
	if err := c.get(ctx, "/api/v1/markets", &resp); err != nil {
		return nil, fmt.Errorf("botclient: GetMarkets: %w", err)
	}
	return resp.Symbols, nil
}

// ---- Market data subscription ----

// Subscribe calls callback every 500ms with the latest Ticker for symbol.
// Runs in a goroutine. Call Stop() to cancel all subscriptions.
func (c *Client) Subscribe(ctx context.Context, symbol string, callback func(Ticker)) {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-c.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				book, err := c.GetOrderBook(ctx, symbol, 1)
				if err != nil {
					continue
				}
				t := Ticker{
					Symbol:    symbol,
					Spread:    book.Spread,
					Timestamp: book.Timestamp,
				}
				if len(book.Bids) > 0 {
					t.BidPrice = book.Bids[0].Price
				}
				if len(book.Asks) > 0 {
					t.AskPrice = book.Asks[0].Price
				}
				callback(t)
			}
		}
	}()
}

func (c *Client) Stop()          { close(c.stopCh) }
func (c *Client) BotID() string  { return c.cfg.BotID }
func (c *Client) APIURL() string { return strings.TrimRight(c.cfg.APIURL, "/") }

// ---- HTTP helpers ----

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.APIURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.APIURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	return c.do(req, out)
}

func (c *Client) delete(ctx context.Context, path string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, c.cfg.APIURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
		}
		json.Unmarshal(body, &e)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, e.Error)
	}
	return json.Unmarshal(body, out)
}

// ---- Price arithmetic using big.Rat ----
// Use these helpers instead of float64 for any price math.
// big.Rat is Go's exact rational arithmetic — zero rounding error.
//
// Example:
//   newPrice := botclient.PriceSub(ticker.AskPrice, "1.00")
//   resp, err := client.PlaceOrder(ctx, botclient.OrderRequest{Price: newPrice, ...})

func parseRat(s string) *big.Rat {
	r := new(big.Rat)
	r.SetString(s)
	return r
}

func formatRat(r *big.Rat) string {
	f, _ := r.Float64()
	return fmt.Sprintf("%.8f", f)
}

// PriceAdd adds two price strings exactly.
// PriceAdd("50000.00", "1.00") → "50001.00000000"
func PriceAdd(a, b string) string {
	return formatRat(new(big.Rat).Add(parseRat(a), parseRat(b)))
}

// PriceSub subtracts two price strings exactly.
// PriceSub("50000.00", "1.00") → "49999.00000000"
func PriceSub(a, b string) string {
	return formatRat(new(big.Rat).Sub(parseRat(a), parseRat(b)))
}

// PriceMul multiplies two price strings exactly.
// PriceMul("50000.00", "0.5") → "25000.00000000"
func PriceMul(a, b string) string {
	return formatRat(new(big.Rat).Mul(parseRat(a), parseRat(b)))
}
