package orderbook

import (
	"container/list"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
)

type PriceLevel struct {
	Price    decimal.Decimal
	orders   *list.List
	index    map[string]*list.Element
	quantity decimal.Decimal
}

func newPriceLevel(price decimal.Decimal) *PriceLevel {
	return &PriceLevel{
		Price:  price,
		orders: list.New(),
		index:  make(map[string]*list.Element),
	}
}

func (pl *PriceLevel) Add(order *models.Order) {
	element := pl.orders.PushBack(order)
	pl.index[order.ID] = element
	pl.quantity = pl.quantity.Add(order.Quantity)
}

func (pl *PriceLevel) Cancel(orderID string) bool {
	element, ok := pl.index[orderID]
	if !ok {
		return false
	}
	order := element.Value.(*models.Order)
	pl.quantity = pl.quantity.Sub(order.Quantity)
	pl.orders.Remove(element)
	delete(pl.index, orderID)
	return true
}

func (pl *PriceLevel) Front() *models.Order {
	element := pl.orders.Front()
	if element == nil {
		return nil
	}
	return element.Value.(*models.Order)
}

func (pl *PriceLevel) PopFront() *models.Order {
	element := pl.orders.Front()
	if element == nil {
		return nil
	}
	order := element.Value.(*models.Order)
	pl.quantity = pl.quantity.Sub(order.Quantity)
	pl.orders.Remove(element)
	delete(pl.index, order.ID)
	return order
}

func (pl *PriceLevel) ReduceQuantity(qty decimal.Decimal) {
	pl.quantity = pl.quantity.Sub(qty)
	if pl.quantity.IsNegative() {
		pl.quantity = decimal.FromInt(0)
	}
}

func (pl *PriceLevel) TotalQuantity() decimal.Decimal { return pl.quantity }
func (pl *PriceLevel) OrderCount() int                { return pl.orders.Len() }
func (pl *PriceLevel) IsEmpty() bool                  { return pl.orders.Len() == 0 }

func (pl *PriceLevel) Snapshot() models.PriceLevelSnapshot {
	return models.PriceLevelSnapshot{
		Price:    pl.Price,
		Quantity: pl.quantity,
		Orders:   pl.orders.Len(),
	}
}
