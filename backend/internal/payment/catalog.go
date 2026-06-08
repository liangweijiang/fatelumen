package payment

// SKU 商品定义。
type SKU struct {
	Type        string // "report" | "credits"
	AmountCents int
	Currency    string
	Credits     int // type=report 时 credits=0
}

// Catalog 商品目录，金额后端定义，前端只传 sku。
// DECISION: MVP 仅 USD，v2 可扩展为 map[sku]map[currency]SKU。
var Catalog = map[string]SKU{
	"report_single": {Type: "report", AmountCents: 599, Currency: "usd", Credits: 0},
	"pack_50":       {Type: "credits", AmountCents: 999, Currency: "usd", Credits: 50},
	"pack_120":      {Type: "credits", AmountCents: 1999, Currency: "usd", Credits: 120},
}
