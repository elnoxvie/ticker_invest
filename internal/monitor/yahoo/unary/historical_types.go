// Package unary provides the client and types for interacting with the Yahoo Finance Unary API.
// This includes fetching current quotes, historical chart data, and managing session details like cookies and crumbs.
package unary

// ChartResponse is the top-level structure for the Yahoo Finance v8 chart API response.
// It encapsulates the chart data or an error if the request failed at the API level.
type ChartResponse struct {
	Chart Chart `json:"chart"` // Contains the actual chart results or error.
}

// Chart holds either the result of a successful chart data fetch or an error object.
type Chart struct {
	Result []ChartResult `json:"result"` // Slice of chart results; typically one per symbol requested.
	Error  *ChartError   `json:"error"`  // Pointer to a ChartError if the API returned an error for this request.
}

// ChartError represents an error object returned by the Yahoo Finance chart API.
type ChartError struct {
	Code        string `json:"code"`        // Error code (e.g., "Not Found", "Forbidden").
	Description string `json:"description"` // Human-readable description of the error.
}

// ChartResult contains the detailed market data for a symbol, including metadata,
// timestamps, and the core OHLCV (Open, High, Low, Close, Volume) indicators.
type ChartResult struct {
	Meta       ChartMeta       `json:"meta"`       // Metadata about the symbol and the data returned.
	Timestamp  []int64         `json:"timestamp"`  // Slice of Unix timestamps for each data point.
	Indicators ChartIndicators `json:"indicators"` // Contains the OHLCV data.
}

// ChartMeta provides metadata related to the financial instrument and the data query.
// This includes currency, symbol, exchange information, trading periods, and data granularity.
type ChartMeta struct {
	Currency             string   `json:"currency"`             // The currency the instrument is traded in (e.g., "USD").
	Symbol               string   `json:"symbol"`               // The ticker symbol (e.g., "AAPL").
	ExchangeName         string   `json:"exchangeName"`         // Name of the exchange (e.g., "NMS", "NYQ").
	InstrumentType       string   `json:"instrumentType"`       // Type of instrument (e.g., "EQUITY", "CURRENCY").
	FirstTradeDate       int64    `json:"firstTradeDate"`       // Unix timestamp of the first trade date.
	RegularMarketTime    int64    `json:"regularMarketTime"`    // Unix timestamp of the regular market time.
	GMTOffset            int64    `json:"gmtoffset"`            // GMT offset in seconds.
	Timezone             string   `json:"timezone"`             // Timezone (e.g., "EST", "EDT").
	ExchangeTimezoneName string   `json:"exchangeTimezoneName"` // Full timezone name of the exchange (e.g., "America/New_York").
	RegularMarketPrice   float64  `json:"regularMarketPrice"`   // Last regular market price.
	ChartPreviousClose   float64  `json:"chartPreviousClose"`   // Previous closing price for the chart.
	PriceHint            int      `json:"priceHint"`            // Number of decimal places for price values.
	CurrentTradingPeriod struct {
		Pre     TradingPeriod `json:"pre"`     // Pre-market trading period details.
		Regular TradingPeriod `json:"regular"` // Regular market trading period details.
		Post    TradingPeriod `json:"post"`    // Post-market trading period details.
	} `json:"currentTradingPeriod"` // Details about current trading periods (pre, regular, post).
	DataGranularity string   `json:"dataGranularity"` // Interval of the data (e.g., "1d", "1wk").
	Range           string   `json:"range"`           // Requested range for the data (e.g., "1y", "max").
	ValidRanges     []string `json:"validRanges"`     // List of valid ranges for this symbol/instrument.
}

// TradingPeriod defines the start time, end time, and GMT offset for a specific trading session (pre, regular, or post).
type TradingPeriod struct {
	Timezone  string `json:"timezone"`  // Timezone for this period.
	Start     int64  `json:"start"`     // Unix timestamp for the start of the period.
	End       int64  `json:"end"`       // Unix timestamp for the end of the period.
	GMTOffset int64  `json:"gmtoffset"` // GMT offset for this period.
}

// ChartIndicators wraps the quote indicators, primarily the OHLCV data.
// Yahoo's API structure includes a "quote" array which holds the OHLCV time series.
type ChartIndicators struct {
	Quote []QuoteIndicator `json:"quote"` // Array of quote indicators, typically containing one element with OHLCV data.
	// AdjClose []AdjCloseIndicator `json:"adjclose"` // Adjusted close prices, if requested (not used in this implementation).
}

// QuoteIndicator holds the time series data for Open, High, Low, Close prices, and Volume.
// Each field is a slice corresponding to the timestamps in ChartResult.Timestamp.
// Note: Yahoo Finance API might return nulls for some data points, especially for very old data
// or specific intervals. This implementation uses plain float64/int64 slices, assuming
// that missing values are either absent from the slice at that index (if lengths differ, handled by transformation)
// or represented as zero/appropriate default. For robust null handling, slices of pointers (e.g., []*float64)
// would be necessary, but this adds complexity to data processing.
type QuoteIndicator struct {
	Open   []float64 `json:"open"`   // Slice of opening prices for each timestamp.
	High   []float64 `json:"high"`   // Slice of high prices for each timestamp.
	Low    []float64 `json:"low"`    // Slice of low prices for each timestamp.
	Close  []float64 `json:"close"`  // Slice of closing prices for each timestamp.
	Volume []int64   `json:"volume"` // Slice of trading volumes for each timestamp.
}
