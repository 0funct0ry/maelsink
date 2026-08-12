package tmpl

import (
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// domainDocs documents higher-level dict-shaped fakers composed from the
// primitive and id generators above, plus gofakeit's credit-card generator.
func (e *Engine) domainDocs() []FuncDoc {
	return []FuncDoc{
		{Name: "fCreditCard", Category: CategoryGenerate, Args: "[type]", Returns: "dict",
			Description: "dict{number,type,cvv,exp} — Luhn-valid test card.", Fn: e.fCreditCard},
		{Name: "fTransaction", Category: CategoryGenerate, Returns: "dict",
			Description: "dict{id,amount,currency,status,timestamp,merchant}.", Fn: e.fTransaction},
		{Name: "fProduct", Category: CategoryGenerate, Returns: "dict",
			Description: "dict{sku,name,category,price,currency,qty}.", Fn: e.fProduct},
		{Name: "fOrder", Category: CategoryGenerate, Args: "[items]", Returns: "dict",
			Description: "dict{id,items,total,currency,created} — `items` fProduct line items (default 1-3).", Fn: e.fOrder},
		{Name: "fInvoice", Category: CategoryGenerate, Args: "[items]", Returns: "dict",
			Description: "dict{invoiceNumber,order,subtotal,tax,total,currency,issued,dueDate,billTo}.", Fn: e.fInvoice},
	}
}

// fCreditCard returns a Luhn-valid fake credit card as a dict, optionally
// restricted to the given card type(s) (e.g. "Visa", "Mastercard").
func (e *Engine) fCreditCard(cardType ...string) map[string]any {
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

// fTransaction returns a fake financial transaction as a dict.
func (e *Engine) fTransaction() map[string]any {
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

// fProduct returns a fake catalog product as a dict.
func (e *Engine) fProduct() map[string]any {
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

// fOrder returns a fake order as a dict with the given number of line
// items (default 1-3 random items).
func (e *Engine) fOrder(items ...int) map[string]any {
	n := e.randInt(1, 3)
	if len(items) > 0 && items[0] > 0 {
		n = items[0]
	}
	id, _ := e.uuidv4()
	var lineItems []map[string]any
	var total float64
	for i := 0; i < n; i++ {
		p := e.fProduct()
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

// fInvoice returns a fake invoice as a dict, built on top of fOrder.
func (e *Engine) fInvoice(items ...int) map[string]any {
	order := e.fOrder(items...)
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
