// Package historical provides a service layer for fetching and processing
// historical stock data from the Yahoo Finance API.
package historical

import (
	"fmt"

	"github.com/achannarasappa/ticker/v4/internal/indicator"
	"github.com/achannarasappa/ticker/v4/internal/monitor/yahoo/unary"
)

// FetchAndProcessHistoricalData retrieves historical market data for a given symbol using the
// provided UnaryAPI client, then transforms this raw data into a slice of
// indicator.HistoricalDataPoint structs.
//
// This function acts as a bridge between the raw API data fetching mechanism
// and the structured format required for technical indicator calculations and analysis.
//
// Parameters:
//   - unaryAPI: A pointer to an initialized `unary.UnaryAPI` client, used to make
//     the actual HTTP request to the Yahoo Finance API.
//   - symbol: The stock ticker symbol (e.g., "AAPL") for which to fetch historical data.
//   - dataRange: A string specifying the range of historical data to fetch (e.g., "1y", "6mo").
//     The exact valid values depend on the Yahoo Finance API.
//   - interval: A string specifying the interval between data points (e.g., "1d", "1wk").
//     The exact valid values depend on the Yahoo Finance API.
//
// Returns:
//   - A slice of indicator.HistoricalDataPoint structs, populated with OHLCV data.
//     Technical indicator fields within these structs (SMA, RSI, etc.) will be nil at this stage.
//   - An error if any part of the process fails (e.g., API request error, data parsing error,
//     or if the API client is nil). Returns a more specific error message if the API itself
//     reports an error (e.g., invalid symbol).
func FetchAndProcessHistoricalData(
	unaryAPI *unary.UnaryAPI,
	symbol string,
	dataRange string, // e.g., "1y", "2y"
	interval string, // e.g., "1d", "1wk"
) ([]indicator.HistoricalDataPoint, error) {

	if unaryAPI == nil {
		return nil, fmt.Errorf("UnaryAPI client is nil")
	}

	// Step 1: Fetch raw historical data
	chartResponse, err := unaryAPI.GetHistoricalData(symbol, dataRange, interval)
	if err != nil {
		// Check if the error is already a ChartError from GetHistoricalData (which it could be)
		// If GetHistoricalData already formats this, this specific check might be redundant,
		// but it's good for clarity if GetHistoricalData could return other types of errors too.
		if chartResponse != nil && chartResponse.Chart.Error != nil {
			return nil, fmt.Errorf("yahoo API error for %s (%s): %s", symbol, chartResponse.Chart.Error.Code, chartResponse.Chart.Error.Description)
		}
		return nil, fmt.Errorf("error fetching historical data for %s: %w", symbol, err)
	}

	if chartResponse == nil {
		return nil, fmt.Errorf("received nil chart response for %s (and no error)", symbol) // Clarified error message
	}

	// Step 1.b: Explicitly check for API-level errors if not already handled by `err` above
	// GetHistoricalData already returns an error if chartResponse.Chart.Error is populated,
	// so this specific block might be redundant if GetHistoricalData is robust.
	// However, an explicit check here ensures service layer responsibility.
	if chartResponse.Chart.Error != nil {
		// This check ensures that even if GetHistoricalData didn't return an error but populated Chart.Error, we catch it.
		// (Though GetHistoricalData is designed to return an error in this case)
		if chartResponse.Chart.Error.Code != "" || chartResponse.Chart.Error.Description != "" {
			return nil, fmt.Errorf("yahoo API error for %s (%s): %s", symbol, chartResponse.Chart.Error.Code, chartResponse.Chart.Error.Description)
		}
	}

	if len(chartResponse.Chart.Result) == 0 {
		// This can happen if the API returns a 200 OK but no actual data (e.g., symbol valid but no historical data for range)
		// Or if Chart.Error was nil but Result was also empty.
		return nil, fmt.Errorf("no chart results found in response for %s", symbol)
	}

	// Step 2: Transform the data
	// unary.TransformHistoricalData is already designed to handle chartResponse.Chart.Result[0]
	// and will return an error if chartResponse.Chart.Result is empty or issues within.
	historicalDataPoints, err := unary.TransformHistoricalData(chartResponse, symbol)
	if err != nil {
		return nil, fmt.Errorf("error transforming historical data for %s: %w", symbol, err)
	}

	return historicalDataPoints, nil
}
