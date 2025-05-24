// Package indicator provides functions for calculating common financial technical indicators.
// These indicators are typically applied to time series data of market prices (OHLCV).
package indicator

import "time"

// HistoricalDataPoint represents a single data point in a time series of market data,
// including Open, High, Low, Close (OHLC) prices, Volume, and calculated technical indicators.
// Pointers are used for indicator fields to distinguish between a calculated value of 0.0
// and a value that has not been computed or is not applicable for a given period.
type HistoricalDataPoint struct {
	Date   time.Time // Date of the data point.
	Open   float64   // Opening price.
	High   float64   // Highest price during the period.
	Low    float64   // Lowest price during the period.
	Close  float64   // Closing price.
	Volume int64     // Trading volume.

	// Technical Indicators
	SMA50      *float64 // Simple Moving Average (e.g., 50 periods).
	SMA200     *float64 // Simple Moving Average (e.g., 200 periods).
	RSI        *float64 // Relative Strength Index.
	MACD       *float64 // Moving Average Convergence Divergence (MACD Line).
	MACDSignal *float64 // MACD Signal Line (EMA of MACD Line).
	MACDHist   *float64 // MACD Histogram (MACD Line - Signal Line).
	ATR        *float64 // Average True Range.
}
