package payment

// SKU 商品定义。
type SKU struct {
	Type    string // "report" | "credits"
	Credits int    // type=report 时为 0
	// Prices 按币种存价格（分），key 为小写币种码。
	Prices map[string]int
}

// Catalog 商品目录，金额后端定义，前端只传 sku。
var Catalog = map[string]SKU{
	"report_single": {Type: "report", Credits: 0, Prices: map[string]int{"usd": 599, "cny": 2900}},
	"pack_50":       {Type: "credits", Credits: 50, Prices: map[string]int{"usd": 999, "cny": 4900}},
	"pack_120":      {Type: "credits", Credits: 120, Prices: map[string]int{"usd": 1999, "cny": 9900}},
}

// CurrencyForProvider 返回某渠道的结算币种。
func CurrencyForProvider(provider string) string {
	if provider == "alipay" {
		return "cny"
	}
	return "usd"
}

// PriceFor 返回某 SKU 在指定币种下的金额（分），不存在返回 0,false。
func PriceFor(sku, currency string) (int, bool) {
	s, ok := Catalog[sku]
	if !ok {
		return 0, false
	}
	amount, ok := s.Prices[currency]
	return amount, ok
}
