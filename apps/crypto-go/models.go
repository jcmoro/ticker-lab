package main

type CryptoPrice struct {
	CoinID     string  `json:"coin_id"`
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	PriceEUR   float64 `json:"price_eur"`
	PriceUSD   float64 `json:"price_usd"`
	MarketCap  float64 `json:"market_cap_eur,omitempty"`
	Change24h  float64 `json:"change_24h,omitempty"`
	Date       string  `json:"date"`
}

type HistoryPoint struct {
	Date  string  `json:"date"`
	Price float64 `json:"price"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Engine    string `json:"engine"`
	Timestamp string `json:"timestamp"`
}

type ProblemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

var topCoins = []struct {
	ID     string
	Symbol string
	Name   string
}{
	{"bitcoin", "BTC", "Bitcoin"},
	{"ethereum", "ETH", "Ethereum"},
	{"solana", "SOL", "Solana"},
	{"binancecoin", "BNB", "BNB"},
	{"ripple", "XRP", "XRP"},
	{"cardano", "ADA", "Cardano"},
	{"dogecoin", "DOGE", "Dogecoin"},
	{"avalanche-2", "AVAX", "Avalanche"},
	{"polkadot", "DOT", "Polkadot"},
	{"polygon-ecosystem-token", "POL", "Polygon"},
	{"chainlink", "LINK", "Chainlink"},
	{"uniswap", "UNI", "Uniswap"},
	{"cosmos", "ATOM", "Cosmos"},
	{"litecoin", "LTC", "Litecoin"},
	{"filecoin", "FIL", "Filecoin"},
	{"aptos", "APT", "Aptos"},
	{"arbitrum", "ARB", "Arbitrum"},
	{"optimism", "OP", "Optimism"},
	{"near", "NEAR", "NEAR Protocol"},
	{"internet-computer", "ICP", "Internet Computer"},
}
