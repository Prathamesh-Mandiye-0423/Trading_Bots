package matching_test

import (
	"testing"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/matching"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func d(s string) decimal.Decimal { return decimal.MustFromString(s) }

func newTestRegistry() *matching.Registry {
	r := matching.NewRegistry(256)
	r.AddMarket("BTC-USD")
	return r
}

func TestMatching_FullFill(t *testing.T) {
	r := newTestRegistry()
	r.Submit(models.NewLimitOrder("seller", "BTC-USD", models.SideSell, d("50000.00"), d("1.0")))
	buy := models.NewLimitOrder("buyer", "BTC-USD", models.SideBuy, d("50000.00"), d("1.0"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	require.Len(t, res.Trades, 1)
	assert.True(t, res.Trades[0].Price.Equal(d("50000.00")))
	assert.Equal(t, models.OrderStatusFilled, buy.Status)
}

func TestMatching_Notional_IsExact(t *testing.T) {
	r := newTestRegistry()
	r.Submit(models.NewLimitOrder("seller", "BTC-USD", models.SideSell, d("50000.00"), d("0.5")))
	buy := models.NewLimitOrder("buyer", "BTC-USD", models.SideBuy, d("50000.00"), d("0.5"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	assert.True(t, res.Trades[0].Notional.Equal(d("25000.00")),
		"got %s", res.Trades[0].Notional.String())
}

func TestMatching_NoMatch_PriceTooLow(t *testing.T) {
	r := newTestRegistry()
	r.Submit(models.NewLimitOrder("seller", "BTC-USD", models.SideSell, d("50000.00"), d("1.0")))
	buy := models.NewLimitOrder("buyer", "BTC-USD", models.SideBuy, d("49000.00"), d("1.0"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	assert.Empty(t, res.Trades)
	assert.NotNil(t, res.RestingOrder)
}

func TestMatching_PartialFill(t *testing.T) {
	r := newTestRegistry()
	r.Submit(models.NewLimitOrder("seller", "BTC-USD", models.SideSell, d("50000.00"), d("0.5")))
	buy := models.NewLimitOrder("buyer", "BTC-USD", models.SideBuy, d("50000.00"), d("1.0"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	require.Len(t, res.Trades, 1)
	assert.True(t, buy.Remaining.Equal(d("0.5")))
	assert.Equal(t, models.OrderStatusPartial, buy.Status)
}

func TestMatching_PricePriority(t *testing.T) {
	r := newTestRegistry()
	r.Submit(models.NewLimitOrder("s1", "BTC-USD", models.SideSell, d("50200.00"), d("1.0")))
	r.Submit(models.NewLimitOrder("s2", "BTC-USD", models.SideSell, d("50000.00"), d("1.0")))
	buy := models.NewLimitOrder("buyer", "BTC-USD", models.SideBuy, d("50200.00"), d("1.0"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	assert.True(t, res.Trades[0].Price.Equal(d("50000.00")), "should fill at best ask")
}

func TestMatching_FIFO(t *testing.T) {
	r := newTestRegistry()
	sell1 := models.NewLimitOrder("s1", "BTC-USD", models.SideSell, d("50000.00"), d("0.5"))
	sell2 := models.NewLimitOrder("s2", "BTC-USD", models.SideSell, d("50000.00"), d("0.5"))
	r.Submit(sell1)
	r.Submit(sell2)
	buy := models.NewLimitOrder("buyer", "BTC-USD", models.SideBuy, d("50000.00"), d("0.5"))
	res, _ := r.Submit(buy)
	assert.Equal(t, sell1.ID, res.Trades[0].SellOrderID)
}

func TestMatching_MarketOrder(t *testing.T) {
	r := newTestRegistry()
	r.Submit(models.NewLimitOrder("seller", "BTC-USD", models.SideSell, d("50000.00"), d("1.0")))
	buy := models.NewMarketOrder("buyer", "BTC-USD", models.SideBuy, d("1.0"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	require.Len(t, res.Trades, 1)
	assert.True(t, res.Trades[0].Price.Equal(d("50000.00")))
}

func TestMatching_MarketOrder_NoLiquidity(t *testing.T) {
	r := newTestRegistry()
	buy := models.NewMarketOrder("buyer", "BTC-USD", models.SideBuy, d("1.0"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	assert.Empty(t, res.Trades)
	assert.Equal(t, models.OrderStatusCancelled, buy.Status)
}

func TestMatching_SweepMultipleLevels(t *testing.T) {
	r := newTestRegistry()
	r.Submit(models.NewLimitOrder("s1", "BTC-USD", models.SideSell, d("50000.00"), d("0.3")))
	r.Submit(models.NewLimitOrder("s2", "BTC-USD", models.SideSell, d("50100.00"), d("0.3")))
	r.Submit(models.NewLimitOrder("s3", "BTC-USD", models.SideSell, d("50200.00"), d("0.3")))
	buy := models.NewLimitOrder("buyer", "BTC-USD", models.SideBuy, d("50200.00"), d("1.0"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	assert.Len(t, res.Trades, 3)
	assert.True(t, buy.Remaining.Equal(d("0.1")))
}

func TestMatching_SelfMatchPrevented(t *testing.T) {
	r := newTestRegistry()
	r.Submit(models.NewLimitOrder("same-bot", "BTC-USD", models.SideSell, d("50000.00"), d("1.0")))
	buy := models.NewLimitOrder("same-bot", "BTC-USD", models.SideBuy, d("50000.00"), d("1.0"))
	res, err := r.Submit(buy)
	require.NoError(t, err)
	assert.Empty(t, res.Trades)
}

func TestMatching_Cancel(t *testing.T) {
	r := newTestRegistry()
	sell := models.NewLimitOrder("seller", "BTC-USD", models.SideSell, d("50000.00"), d("1.0"))
	r.Submit(sell)
	cancelled, err := r.Cancel("BTC-USD", sell.ID)
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusCancelled, cancelled.Status)
}

func TestMatching_InvalidQuantity(t *testing.T) {
	r := newTestRegistry()
	order := models.NewLimitOrder("bot1", "BTC-USD", models.SideBuy, d("50000.00"), d("0"))
	_, err := r.Submit(order)
	assert.Error(t, err)
}

func TestMatching_UnknownSymbol(t *testing.T) {
	r := newTestRegistry()
	order := models.NewLimitOrder("bot1", "UNKNOWN", models.SideBuy, d("50000.00"), d("1.0"))
	_, err := r.Submit(order)
	assert.Error(t, err)
}
