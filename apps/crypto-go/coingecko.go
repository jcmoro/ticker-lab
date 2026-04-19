package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type CoinGeckoClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewCoinGeckoClient() *CoinGeckoClient {
	return &CoinGeckoClient{
		baseURL: "https://api.coingecko.com/api/v3",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type simplePriceResponse map[string]struct {
	EUR         float64 `json:"eur"`
	USD         float64 `json:"usd"`
	EURMarketCap float64 `json:"eur_market_cap"`
	EUR24hChange float64 `json:"eur_24h_change"`
}

func (c *CoinGeckoClient) FetchPrices(coins []struct{ ID, Symbol, Name string }) ([]CryptoPrice, error) {
	ids := make([]string, len(coins))
	for i, coin := range coins {
		ids[i] = coin.ID
	}

	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=eur,usd&include_market_cap=true&include_24hr_change=true",
		c.baseURL, strings.Join(ids, ","))

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("coingecko request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko returned %d", resp.StatusCode)
	}

	var data simplePriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("coingecko decode failed: %w", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	coinMap := make(map[string]struct{ Symbol, Name string })
	for _, coin := range coins {
		coinMap[coin.ID] = struct{ Symbol, Name string }{coin.Symbol, coin.Name}
	}

	var prices []CryptoPrice
	for id, p := range data {
		meta, ok := coinMap[id]
		if !ok {
			continue
		}
		prices = append(prices, CryptoPrice{
			CoinID:    id,
			Symbol:    meta.Symbol,
			Name:      meta.Name,
			PriceEUR:  p.EUR,
			PriceUSD:  p.USD,
			MarketCap: p.EURMarketCap,
			Change24h: p.EUR24hChange,
			Date:      today,
		})
	}

	return prices, nil
}

type marketChartResponse struct {
	Prices [][]float64 `json:"prices"` // [[timestamp_ms, price], ...]
}

// FetchHistory fetches daily price history for a single coin.
// CoinGecko rate limit: ~10 req/min on free tier — caller should throttle.
func (c *CoinGeckoClient) FetchHistory(coinID string, days int) ([]CryptoPrice, error) {
	url := fmt.Sprintf("%s/coins/%s/market_chart?vs_currency=eur&days=%d&interval=daily",
		c.baseURL, coinID, days)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("coingecko history request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko history returned %d", resp.StatusCode)
	}

	var data marketChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("coingecko history decode failed: %w", err)
	}

	// Find coin metadata
	var symbol, name string
	for _, coin := range topCoins {
		if coin.ID == coinID {
			symbol = coin.Symbol
			name = coin.Name
			break
		}
	}

	var prices []CryptoPrice
	for _, point := range data.Prices {
		if len(point) < 2 {
			continue
		}
		ts := time.UnixMilli(int64(point[0])).UTC()
		date := ts.Format("2006-01-02")
		prices = append(prices, CryptoPrice{
			CoinID:   coinID,
			Symbol:   symbol,
			Name:     name,
			PriceEUR: point[1],
			PriceUSD: 0, // market_chart only returns one currency
			Date:     date,
		})
	}

	return prices, nil
}
