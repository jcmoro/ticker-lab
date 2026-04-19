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
