// package orderbook
package orderbook_test

import (
	"testing"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/orderbook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func d(s string) decimal.Decimal { return decimal.MustFromString(s) }

func TestOrderBook_BestBid(t *testing.T) {
	ob := orderbook.NewOrderBook("BTC-USD")
	ob.AddOrder(models.NewLimitOrder("b1", "BTC-USD", models.SideBuy, d("50000.00"), d("1.0")))
	ob.AddOrder(models.NewLimitOrder("b2", "BTC-USD", models.SideBuy, d("51000.00"), d("0.5")))
	ob.AddOrder(models.NewLimitOrder("b3", "BTC-USD", models.SideBuy, d("49000.00"), d("2.0")))
	assert.True(t, ob.BestBidPrice().Equal(d("51000.00")))
}

func TestOrderBook_BestAsk(t *testing.T) {
	ob := orderbook.NewOrderBook("BTC-USD")
	ob.AddOrder(models.NewLimitOrder("s1", "BTC-USD", models.SideSell, d("52000.00"), d("1.0")))
	ob.AddOrder(models.NewLimitOrder("s2", "BTC-USD", models.SideSell, d("51500.00"), d("0.5")))
	ob.AddOrder(models.NewLimitOrder("s3", "BTC-USD", models.SideSell, d("53000.00"), d("2.0")))
	assert.True(t, ob.BestAskPrice().Equal(d("51500.00")))
}

func TestOrderBook_Spread(t *testing.T) {
	ob := orderbook.NewOrderBook("BTC-USD")
	ob.AddOrder(models.NewLimitOrder("b1", "BTC-USD", models.SideBuy, d("50000.00"), d("1.0")))
	ob.AddOrder(models.NewLimitOrder("s1", "BTC-USD", models.SideSell, d("50100.00"), d("1.0")))
	assert.True(t, ob.Spread().Equal(d("100.00")))
}

func TestOrderBook_Cancel(t *testing.T) {
	ob := orderbook.NewOrderBook("BTC-USD")
	o1 := models.NewLimitOrder("b1", "BTC-USD", models.SideBuy, d("50000.00"), d("1.0"))
	o2 := models.NewLimitOrder("b2", "BTC-USD", models.SideBuy, d("51000.00"), d("0.5"))
	ob.AddOrder(o1)
	ob.AddOrder(o2)
	cancelled, err := ob.CancelOrder(o2.ID)
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusCancelled, cancelled.Status)
	assert.True(t, ob.BestBidPrice().Equal(d("50000.00")))
}

func TestOrderBook_Snapshot_SortOrder(t *testing.T) {
	ob := orderbook.NewOrderBook("BTC-USD")
	for i := 0; i < 5; i++ {
		price := d("50000.00").Sub(decimal.FromInt(int64(i * 100)))
		ob.AddOrder(models.NewLimitOrder("b", "BTC-USD", models.SideBuy, price, d("1.0")))
	}
	for i := 0; i < 5; i++ {
		price := d("51000.00").Add(decimal.FromInt(int64(i * 100)))
		ob.AddOrder(models.NewLimitOrder("s", "BTC-USD", models.SideSell, price, d("1.0")))
	}
	snap := ob.Snapshot(3)
	assert.Len(t, snap.Bids, 3)
	assert.Len(t, snap.Asks, 3)
	assert.True(t, snap.Bids[0].Price.Greater(snap.Bids[1].Price), "bids must be descending")
	assert.True(t, snap.Asks[0].Price.Less(snap.Asks[1].Price), "asks must be ascending")
}
