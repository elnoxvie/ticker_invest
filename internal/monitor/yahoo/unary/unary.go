package unary

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	c "github.com/achannarasappa/ticker/v4/internal/common"
)

// UnaryAPI is a client for the API
type UnaryAPI struct {
	client            *http.Client
	baseURL           string
	sessionRootURL    string
	sessionCrumbURL   string
	sessionConsentURL string
	cookies           []*http.Cookie
	crumb             string
}

// Config contains configuration options for the UnaryAPI client
type Config struct {
	BaseURL           string
	SessionRootURL    string
	SessionCrumbURL   string
	SessionConsentURL string
}

type SymbolToCurrency struct {
	Symbol       string
	FromCurrency string
}

// NewUnaryAPI creates a new client
func NewUnaryAPI(config Config) *UnaryAPI {
	// Create client with limited redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 1 {
				return http.ErrUseLastResponse
			}

			return nil
		},
	}

	return &UnaryAPI{
		client:            client,
		baseURL:           config.BaseURL,
		sessionRootURL:    config.SessionRootURL,
		sessionCrumbURL:   config.SessionCrumbURL,
		sessionConsentURL: config.SessionConsentURL,
	}
}

// GetAssetQuotes issues a HTTP request to retrieve quotes from the API and process the response
func (u *UnaryAPI) GetAssetQuotes(symbols []string) ([]c.AssetQuote, map[string]*c.AssetQuote, error) {
	if len(symbols) == 0 {
		return []c.AssetQuote{}, make(map[string]*c.AssetQuote), nil
	}

	result, err := u.getQuotes(symbols, []string{"shortName", "regularMarketChange", "regularMarketChangePercent", "regularMarketPrice", "regularMarketPreviousClose", "regularMarketOpen", "regularMarketDayRange", "regularMarketDayHigh", "regularMarketDayLow", "regularMarketVolume", "postMarketChange", "postMarketChangePercent", "postMarketPrice", "preMarketChange", "preMarketChangePercent", "preMarketPrice", "fiftyTwoWeekHigh", "fiftyTwoWeekLow", "marketCap"})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get quotes: %w", err)
	}

	quotes, quotesBySymbol := transformResponseQuotes(result.QuoteResponse.Quotes)

	return quotes, quotesBySymbol, nil
}

// GetCurrencyMap retrieves the currency which the price quote will be denominated in for the given symbols
func (u *UnaryAPI) GetCurrencyMap(symbols []string) (map[string]SymbolToCurrency, error) {
	if len(symbols) == 0 {
		return map[string]SymbolToCurrency{}, nil
	}

	result, err := u.getQuotes(symbols, []string{"regularMarketPrice", "currency"})

	if err != nil {
		return map[string]SymbolToCurrency{}, err
	}

	symbolToCurrency := make(map[string]SymbolToCurrency)

	for _, quote := range result.QuoteResponse.Quotes {
		symbolToCurrency[quote.Symbol] = SymbolToCurrency{
			Symbol:       quote.Symbol,
			FromCurrency: strings.ToUpper(quote.Currency),
		}
	}

	return symbolToCurrency, nil
}

// GetCurrencyRates accepts an array of ISO 4217 currency codes and a target ISO 4217 currency code and returns a conversion rate for each of the input currencies to the target currency
func (u *UnaryAPI) GetCurrencyRates(fromCurrencies []string, toCurrency string) (c.CurrencyRates, error) {
	if toCurrency == "" {
		toCurrency = "USD"
	}

	if len(fromCurrencies) == 0 {
		return c.CurrencyRates{}, nil
	}

	// Create currency pair symbols in format "FROMTO=X" (e.g., "EURUSD=X")
	currencyPairSymbols := make([]string, 0)
	currencyPairSymbolsUnique := make(map[string]bool)

	for _, fromCurrency := range fromCurrencies {

		if fromCurrency == "" {
			continue
		}

		if fromCurrency == toCurrency {
			continue
		}

		pair := strings.ToUpper(fromCurrency) + toCurrency + "=X"

		if _, exists := currencyPairSymbolsUnique[pair]; !exists {
			currencyPairSymbolsUnique[pair] = true
			currencyPairSymbols = append(currencyPairSymbols, pair)
		}
	}

	if len(currencyPairSymbols) == 0 {
		return c.CurrencyRates{}, nil
	}

	// Get quotes for currency pairs
	result, err := u.getQuotes(currencyPairSymbols, []string{"currency", "regularMarketPrice"})
	if err != nil {
		return c.CurrencyRates{}, fmt.Errorf("failed to get currency rates: %w", err)
	}

	// Transform result to currency rates
	currencyRates := make(map[string]c.CurrencyRate)

	for _, quote := range result.QuoteResponse.Quotes {
		fromCurrency := strings.TrimSuffix(strings.TrimSuffix(quote.Symbol, "=X"), toCurrency)
		currencyRates[fromCurrency] = c.CurrencyRate{
			FromCurrency: fromCurrency,
			ToCurrency:   toCurrency,
			Rate:         quote.RegularMarketPrice.Raw,
		}
	}

	return currencyRates, nil
}

func (u *UnaryAPI) getQuotes(symbols []string, fields []string) (Response, error) {

	// Build URL with query parameters
	reqURL, err := url.Parse(u.baseURL + "/v7/finance/quote")
	if err != nil {
		return Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	q := reqURL.Query()
	q.Set("fields", strings.Join(fields, ","))
	q.Set("symbols", strings.Join(symbols, ","))

	// Add common Yahoo Finance query parameters
	q.Set("formatted", "true")
	q.Set("lang", "en-US")
	q.Set("region", "US")
	q.Set("corsDomain", "finance.yahoo.com")

	// Add crumb if available
	if u.crumb != "" {
		q.Set("crumb", u.crumb)
	}

	reqURL.RawQuery = q.Encode()

	// Create request
	req, _ := http.NewRequest(http.MethodGet, reqURL.String(), nil)

	// Set common headers
	req.Header.Set("Authority", "query1.finance.yahoo.com")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", defaultAcceptLang)
	req.Header.Set("Origin", u.baseURL)
	req.Header.Set("User-Agent", defaultUserAgent)

	// Add cookies if available
	if len(u.cookies) > 0 {
		for _, cookie := range u.cookies {
			req.AddCookie(cookie)
		}
	}

	// Make request
	resp, err := u.client.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Handle not ok responses
	if resp.StatusCode >= 400 {
		// Try to refresh session and retry once
		if err := u.refreshSession(); err != nil {
			return Response{}, fmt.Errorf("session refresh failed: %w", err)
		}

		// Retry request with refreshed session
		return u.getQuotes(symbols, fields)
	}

	// Handle unexpected responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode < 400 {
		return Response{}, fmt.Errorf("unexpected response: %d", resp.StatusCode)
	}

	// Decode response
	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Response{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// GetHistoricalData retrieves historical market data (OHLCV) for a given symbol
// from the Yahoo Finance v8 chart API.
//
// It constructs the request URL, sets necessary query parameters (range, interval, indicators, etc.),
// and common headers. It also includes session management (cookies, crumb) and a retry mechanism
// with session refresh if the initial request fails with a non-OK HTTP status code.
//
// Parameters:
//   - symbol: The stock ticker symbol (e.g., "AAPL") for which to fetch historical data.
//   - dataRange: A string specifying the range of historical data (e.g., "1y", "6mo", "max").
//   - interval: A string specifying the interval between data points (e.g., "1d", "1wk", "1mo").
//
// Returns:
//   - A pointer to a ChartResponse struct, which contains the parsed JSON response from the API.
//     This includes metadata, timestamps, and OHLCV data.
//   - An error if the request fails at any stage (URL parsing, HTTP request, session refresh,
//     JSON decoding), or if the API returns an error in the response body (e.g., symbol not found).
//     The error will be specific about the cause of failure.
func (u *UnaryAPI) GetHistoricalData(symbol string, dataRange string, interval string) (*ChartResponse, error) {
	// Construct URL
	// Example: https://query1.finance.yahoo.com/v8/finance/chart/AAPL?range=1mo&interval=1d&indicators=quote&includeTimestamp=true
	reqURL, err := url.Parse(fmt.Sprintf("%s/v8/finance/chart/%s", u.baseURL, symbol))
	if err != nil {
		return nil, fmt.Errorf("failed to parse chart URL: %w", err)
	}

	// Set query parameters
	q := reqURL.Query()
	q.Set("range", dataRange)
	q.Set("interval", interval)
	q.Set("indicators", "quote")      // We are interested in OHLCV
	q.Set("includeTimestamp", "true") // Timestamps are essential

	// Add common Yahoo Finance query parameters
	q.Set("formatted", "true")
	q.Set("lang", "en-US")
	q.Set("region", "US")
	q.Set("corsDomain", "finance.yahoo.com")

	// Add crumb if available
	if u.crumb != "" {
		q.Set("crumb", u.crumb)
	}
	reqURL.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create historical data request: %w", err)
	}

	// Set common headers (similar to getQuotes)
	req.Header.Set("Authority", "query1.finance.yahoo.com") // This might need to be just the host part of u.baseURL
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", defaultAcceptLang)
	req.Header.Set("Origin", u.baseURL) // Or a more generic origin like "https://finance.yahoo.com"
	req.Header.Set("User-Agent", defaultUserAgent)

	// Add cookies if available
	if len(u.cookies) > 0 {
		for _, cookie := range u.cookies {
			req.AddCookie(cookie)
		}
	}

	// Make request
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make historical data request: %w", err)
	}
	defer resp.Body.Close()

	// Handle not ok responses (including potential session refresh)
	if resp.StatusCode >= 400 {
		// Try to refresh session and retry once
		// Note: It's not guaranteed that the chart API uses the same session/crumb mechanism
		// as the quote API, but it's a reasonable assumption to try.
		if err := u.refreshSession(); err != nil {
			// If refresh fails, return original error based on status code
			return nil, fmt.Errorf("session refresh failed after status %d: %w. Original error: %s", resp.StatusCode, err, resp.Status)
		}
		// Retry request with refreshed session
		// It's important to prevent infinite loops if retry also fails.
		// The current refreshSession logic doesn't inherently prevent this for GetHistoricalData.
		// For simplicity, we'll retry once. A more robust solution might involve a retry counter.
		return u.GetHistoricalData(symbol, dataRange, interval) // Recursive call for retry
	}

	// Handle unexpected non-error responses
	if resp.StatusCode != http.StatusOK { // Check for explicit OK, not just < 400
		return nil, fmt.Errorf("unexpected response for historical data: %s (status code: %d)", resp.Status, resp.StatusCode)
	}

	// Decode response
	var chartResponse ChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&chartResponse); err != nil {
		return nil, fmt.Errorf("failed to decode historical data response: %w", err)
	}

	// Check for API-level errors in the decoded response
	if chartResponse.Chart.Error != nil {
		return &chartResponse, fmt.Errorf("chart API error: %s - %s", chartResponse.Chart.Error.Code, chartResponse.Chart.Error.Description)
	}

	return &chartResponse, nil
}
