package tmpl

import (
	"fmt"
	"text/template"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// domainFuncMap returns higher-level dict-shaped fakers composed from the
// primitive and id generators above, plus gofakeit's credit-card generator.
func (e *Engine) domainFuncMap() template.FuncMap {
	return template.FuncMap{
		"fakeCreditCard":  e.fakeCreditCard,
		"fakeTransaction": e.fakeTransaction,
		"fakeProduct":     e.fakeProduct,
		"fakeOrder":       e.fakeOrder,
		"fakeInvoice":     e.fakeInvoice,
	}
}

// fakeCreditCard returns a Luhn-valid fake credit card as a dict, optionally
// restricted to the given card type(s) (e.g. "Visa", "Mastercard").
func (e *Engine) fakeCreditCard(cardType ...string) map[string]any {
	var cco *gofakeit.CreditCardOptions
	if len(cardType) > 0 && cardType[0] != "" {
		cco = &gofakeit.CreditCardOptions{Types: cardType}
	}
	number := e.faker.CreditCardNumber(cco)
	ct := e.faker.CreditCardType()
	if cco != nil {
		ct = cardType[0]
	}
	return map[string]any{
		"number": number,
		"type":   ct,
		"cvv":    e.faker.CreditCardCvv(),
		"exp":    e.faker.CreditCardExp(),
	}
}

// fakeTransaction returns a fake financial transaction as a dict.
func (e *Engine) fakeTransaction() map[string]any {
	id, _ := e.uuidv4()
	statuses := []string{"pending", "completed", "failed", "refunded"}
	return map[string]any{
		"id":        id,
		"amount":    e.randFloat(1, 5000, 2),
		"currency":  "USD",
		"status":    statuses[e.rnd.Intn(len(statuses))],
		"timestamp": time.Now().Format(time.RFC3339),
		"merchant":  e.faker.Company(),
	}
}

// fakeProduct returns a fake catalog product as a dict.
func (e *Engine) fakeProduct() map[string]any {
	categories := []string{"Electronics", "Home", "Toys", "Grocery", "Apparel", "Books"}
	return map[string]any{
		"sku":      "SKU-" + e.randString(8, base62Alphabet),
		"name":     e.faker.ProductName(),
		"category": categories[e.rnd.Intn(len(categories))],
		"price":    e.randFloat(1, 500, 2),
		"currency": "USD",
		"qty":      e.randInt(0, 500),
	}
}

// fakeOrder returns a fake order as a dict with the given number of line
// items (default 1-3 random items).
func (e *Engine) fakeOrder(items ...int) map[string]any {
	n := e.randInt(1, 3)
	if len(items) > 0 && items[0] > 0 {
		n = items[0]
	}
	id, _ := e.uuidv4()
	var lineItems []map[string]any
	var total float64
	for i := 0; i < n; i++ {
		p := e.fakeProduct()
		qty := e.randInt(1, 5)
		price := p["price"].(float64)
		lineItems = append(lineItems, map[string]any{
			"product": p,
			"qty":     qty,
			"subtotal": func() float64 {
				return roundCents(price * float64(qty))
			}(),
		})
		total += price * float64(qty)
	}
	return map[string]any{
		"id":       id,
		"items":    lineItems,
		"total":    roundCents(total),
		"currency": "USD",
		"created":  time.Now().Format(time.RFC3339),
	}
}

// fakeInvoice returns a fake invoice as a dict, built on top of fakeOrder.
func (e *Engine) fakeInvoice(items ...int) map[string]any {
	order := e.fakeOrder(items...)
	total := order["total"].(float64)
	tax := roundCents(total * 0.08)
	return map[string]any{
		"invoiceNumber": fmt.Sprintf("INV-%06d", e.randInt(1, 999999)),
		"order":         order,
		"subtotal":      total,
		"tax":           tax,
		"total":         roundCents(total + tax),
		"currency":      "USD",
		"issued":        time.Now().Format(time.RFC3339),
		"dueDate":       time.Now().AddDate(0, 0, 30).Format(time.RFC3339),
		"billTo":        e.faker.Name(),
	}
}

func roundCents(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
