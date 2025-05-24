package unary

import (
	"fmt"
	"time"

	"github.com/achannarasappa/ticker/v4/internal/indicator"
)

// TransformHistoricalData transforms the API response into a slice of HistoricalDataPoint.
func TransformHistoricalData(response *ChartResponse, symbol string) ([]indicator.HistoricalDataPoint, error) {
	if response == nil || response.Chart.Result == nil || len(response.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data found in response for symbol %s: Chart or Result is nil or empty", symbol)
	}

	chartResult := response.Chart.Result[0] // Assuming the first result is the relevant one

	if len(chartResult.Timestamp) == 0 {
		return nil, fmt.Errorf("no timestamps found for symbol %s", symbol)
	}

	if chartResult.Indicators.Quote == nil || len(chartResult.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote indicators found for symbol %s: Quote is nil or empty", symbol)
	}

	quotes := chartResult.Indicators.Quote[0] // Assuming first set of quotes
	numPoints := len(chartResult.Timestamp)

	// Validate that all data slices have the same length as timestamps
	// Also check for nil slices, although Yahoo's API usually returns empty slices instead of nil for these fields.
	if quotes.Open == nil || len(quotes.Open) != numPoints {
		return nil, fmt.Errorf("open data length mismatch or nil for symbol %s. Expected %d, got %d", symbol, numPoints, len(quotes.Open))
	}
	if quotes.High == nil || len(quotes.High) != numPoints {
		return nil, fmt.Errorf("high data length mismatch or nil for symbol %s. Expected %d, got %d", symbol, numPoints, len(quotes.High))
	}
	if quotes.Low == nil || len(quotes.Low) != numPoints {
		return nil, fmt.Errorf("low data length mismatch or nil for symbol %s. Expected %d, got %d", symbol, numPoints, len(quotes.Low))
	}
	if quotes.Close == nil || len(quotes.Close) != numPoints {
		return nil, fmt.Errorf("close data length mismatch or nil for symbol %s. Expected %d, got %d", symbol, numPoints, len(quotes.Close))
	}
	if quotes.Volume == nil || len(quotes.Volume) != numPoints {
		return nil, fmt.Errorf("volume data length mismatch or nil for symbol %s. Expected %d, got %d", symbol, numPoints, len(quotes.Volume))
	}

	historicalData := make([]indicator.HistoricalDataPoint, numPoints)
	for i := 0; i < numPoints; i++ {
		// Yahoo sometimes returns 0 for volume for some intervals/assets where volume is not applicable (e.g. indices, currencies for certain periods)
		// or when a trading day had no volume. This is valid data.
		// Nulls within OHLC data are less common from Yahoo for standard equities but could happen.
		// The current QuoteIndicator uses []float64, not []*float64, so we can't distinguish 0 from actual null.
		// This is usually fine for OHLC as 0 is a very unlikely price, but if this needs to be handled,
		// historical_types.go would need to change. For now, direct assignment is used.

		historicalData[i] = indicator.HistoricalDataPoint{
			Date:   time.Unix(chartResult.Timestamp[i], 0).In(time.UTC), // Store in UTC
			Open:   quotes.Open[i],
			High:   quotes.High[i],
			Low:    quotes.Low[i],
			Close:  quotes.Close[i],
			Volume: quotes.Volume[i],
			// Technical indicator fields (SMA50, SMA200, RSI, etc.) are initially nil
			// because they are pointers in HistoricalDataPoint. They will be calculated later.
		}
	}

	return historicalData, nil
}
