package indicator

import (
	"math" // For Abs and Max
)

// CalculateATR computes the Average True Range (ATR) for a series of historical data points
// over a specified period. ATR is a measure of market volatility.
//
// The True Range (TR) for a given period is the greatest of the following:
//  1. Current High minus Current Low.
//  2. Absolute value of (Current High minus Previous Close).
//  3. Absolute value of (Current Low minus Previous Close).
//
// The ATR is then calculated using Wilder's smoothing method (an EMA of TR values):
//  1. The first ATR value is an SMA of the first `period` TR values. This value corresponds
//     to `data[period]`.
//  2. Subsequent ATRs = ((Previous ATR * (period - 1)) + Current TR) / period.
//
// Parameters:
//   - data: A slice of HistoricalDataPoint structs. Requires `period + 1` data points
//     to calculate the first ATR value (as TR calculation starts from data[1]).
//   - period: An integer for the ATR period (e.g., 14). Must be positive.
//
// Returns:
//   - A slice of *float64, representing the ATR values.
//   - Elements are nil where ATR cannot be calculated (i.e., for the first `period` data points).
//   - If period is invalid or data length is insufficient (<= period), a slice of nils of
//     the same length as data is returned.
func CalculateATR(data []HistoricalDataPoint, period int) []*float64 {
	// Need at least 'period' data points to calculate TRs and then the first ATR.
	// The first TR is for data[1] (using data[0] for PrevClose).
	// 'period' TRs are from data[1]...data[period].
	// The first ATR (SMA of these TRs) is calculated for data[period].
	// So, we need len(data) to be at least period + 1.
	// However, the loop for TR goes up to len(data)-1.
	// If len(data) == period + 1, then TRs up to trueRanges[period] are calculated.
	// The initial ATR sum uses trueRanges[1] to trueRanges[period].
	// This means data points data[0] through data[period] are involved.
	// So, len(data) must be > period.
	if period <= 0 || len(data) <= period {
		// If len(data) == period, we can't calculate trueRanges[period] if TR starts at index 1.
		// e.g. period=14, len(data)=14. Loop for TR is i=1 to 13. trueRanges[1]..trueRanges[13].
		// Sum for initial ATR needs trueRanges[1]..trueRanges[14]. Fails.
		// So, len(data) must be at least period+1 for the first ATR to be at data[period].
		return make([]*float64, len(data)) // Default nil values
	}

	atrValues := make([]*float64, len(data))  // Initialized with nils
	trueRanges := make([]float64, len(data)) // trueRanges[0] will be unused

	// Calculate True Range (TR)
	// TR for data[0] could be High[0] - Low[0], but we start from data[1] for consistency
	// with formulas needing PreviousClose.
	// trueRanges[i] corresponds to the TR for data[i].
	for i := 1; i < len(data); i++ {
		highMinusLow := data[i].High - data[i].Low
		highMinusPrevClose := math.Abs(data[i].High - data[i-1].Close)
		lowMinusPrevClose := math.Abs(data[i].Low - data[i-1].Close)

		trueRanges[i] = math.Max(highMinusLow, math.Max(highMinusPrevClose, lowMinusPrevClose))
	}

	// Calculate initial ATR (SMA of TRs for the period)
	// The first ATR is calculated at index 'period' of data, using TRs from trueRanges[1]...trueRanges[period].
	// This involves data points data[0] (for prevClose of data[1]) up to data[period].
	trSum := 0.0
	// Ensure trueRanges has enough elements up to 'period'.
	// This is implicitly handled by len(data) > period check,
	// as trueRanges has same length as data, and TRs are filled up to len(data)-1.
	for i := 1; i <= period; i++ {
		// If period is large, e.g., period = len(data) -1, this loop is fine.
		// If period = len(data), this loop will go out of bounds for trueRanges.
		// But len(data) > period check prevents period == len(data).
		// Max value for period is len(data)-1. Loop goes up to len(data)-1.
		trSum += trueRanges[i]
	}

	// This safety check for period == 0 is mostly redundant due to the initial check period <= 0
	// but doesn't harm.
	if period == 0 {
		return atrValues // Should have already returned via period <= 0
	}

	currentAtr := trSum / float64(period)
	atrValues[period] = &currentAtr // Store first ATR at index 'period'

	// Calculate subsequent ATR values using Wilder's smoothing
	// Subsequent ATRs start from index period + 1
	for i := period + 1; i < len(data); i++ {
		// Previous ATR is currentAtr (which is atrValues[i-1])
		// Current TR is trueRanges[i]
		currentAtr = (currentAtr*float64(period-1) + trueRanges[i]) / float64(period)
		atrValues[i] = &currentAtr
	}

	return atrValues
}
