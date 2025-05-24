package indicator

import (
	"math" // For Abs
)

// CalculateRSI computes the Relative Strength Index (RSI) for the 'Close' price
// of a series of historical data points over a specified period.
//
// The RSI is a momentum oscillator that measures the speed and change of price movements.
// It oscillates between zero and 100. Traditionally, RSI is considered overbought
// when above 70 and oversold when below 30.
//
// The calculation involves:
// 1. Calculating price changes (gains and losses).
// 2. Calculating the initial average gain and average loss over the first `period` changes.
//    The first RSI value corresponds to `data[period]`.
// 3. Calculating subsequent RSI values using Wilder's smoothing method for average gain/loss.
//    AvgGain = (Previous AvgGain * (Period - 1) + Current Gain) / Period
//    AvgLoss = (Previous AvgLoss * (Period - 1) + Current Loss) / Period
// 4. RSI = 100 - (100 / (1 + RS)), where RS = Average Gain / Average Loss.
//    If Average Loss is zero, RSI is 100.
//
// Parameters:
//   - data: A slice of HistoricalDataPoint structs. Requires `period + 1` data points
//     to calculate the first RSI value.
//   - period: An integer for the RSI period (e.g., 14). Must be positive.
//
// Returns:
//   - A slice of *float64, representing the RSI values.
//   - Elements are nil where RSI cannot be calculated (i.e., for the first `period` data points).
//   - If period is invalid or data length is insufficient (<= period), a slice of nils of
//     the same length as data is returned.
func CalculateRSI(data []HistoricalDataPoint, period int) []*float64 {
	// Need at least 'period' changes, which means 'period + 1' data points.
	// The first RSI value is calculated at index 'period' of the data slice.
	if period <= 0 || len(data) <= period {
		// Return slice of nils with the same length as data.
		// Default for []*float64 elements is nil.
		return make([]*float64, len(data))
	}

	rsiValues := make([]*float64, len(data)) // Initialized with nils

	// Calculate price changes. changes[0] will be unused (or 0.0).
	// changes[i] corresponds to data[i].Close - data[i-1].Close.
	changes := make([]float64, len(data))
	for i := 1; i < len(data); i++ {
		changes[i] = data[i].Close - data[i-1].Close
	}

	var avgGain, avgLoss float64

	// Calculate initial average gain and loss for the first RSI value.
	// This first RSI value corresponds to data[period].
	// It uses 'period' number of changes: changes[1] through changes[period].
	gainSum := 0.0
	lossSum := 0.0
	for i := 1; i <= period; i++ { // Iterate over the first 'period' changes
		change := changes[i]
		if change > 0 {
			gainSum += change
		} else {
			lossSum += math.Abs(change) // lossSum is sum of positive values
		}
	}
	avgGain = gainSum / float64(period)
	avgLoss = lossSum / float64(period)

	// Calculate first RSI value at index 'period'
	if avgLoss == 0 {
		val := 100.0
		rsiValues[period] = &val
	} else {
		rs := avgGain / avgLoss
		val := 100.0 - (100.0 / (1.0 + rs))
		rsiValues[period] = &val
	}

	// Calculate subsequent RSI values using Wilder's smoothing method
	// Start from index period + 1
	for i := period + 1; i < len(data); i++ {
		change := changes[i]
		currentGain := 0.0
		currentLoss := 0.0

		if change > 0 {
			currentGain = change
		} else {
			currentLoss = math.Abs(change)
		}

		// Wilder's smoothing:
		// AvgGain = (Previous AvgGain * (Period - 1) + Current Gain) / Period
		// AvgLoss = (Previous AvgLoss * (Period - 1) + Current Loss) / Period
		avgGain = (avgGain*float64(period-1) + currentGain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + currentLoss) / float64(period)

		if avgLoss == 0 {
			val := 100.0
			rsiValues[i] = &val
		} else {
			rs := avgGain / avgLoss
			val := 100.0 - (100.0 / (1.0 + rs))
			rsiValues[i] = &val
		}
	}

	return rsiValues
}
