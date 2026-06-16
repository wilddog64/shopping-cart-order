package order

import "github.com/shopspring/decimal"

func init() {
	decimal.MarshalJSONWithoutQuotes = true
}
