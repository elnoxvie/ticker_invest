package indicator

// CalculateEMA computes the Exponential Moving Average (EMA) for the 'Close' price
// of a series of historical data points over a specified period.
//
// The EMA is a type of moving average that places a greater weight and significance
// on the most recent data points. The calculation starts by seeding the first EMA
// value with a Simple Moving Average (SMA) of the same period. Subsequent EMA values
// are calculated using the formula:
// EMA = (Current Price - Previous EMA) * Multiplier + Previous EMA
// where Multiplier = 2 / (Period + 1).
//
// Parameters:
//   - data: A slice of HistoricalDataPoint structs.
//   - period: An integer for the EMA period. Must be positive.
//
// Returns:
//   - A slice of *float64, representing the EMA values.
//   - Elements are nil where EMA cannot be calculated (e.g., insufficient data for initial SMA).
//   - If period is invalid or data is empty, a slice of nils of the same length as data is returned.
func CalculateEMA(data []HistoricalDataPoint, period int) []*float64 {
	if period <= 0 || len(data) == 0 {
		// Return slice of nils with the same length as data.
		// If len(data) is 0, this makes an empty slice.
		// If len(data) > 0 but period is invalid, all points get nil EMA.
		result := make([]*float64, len(data))
		for i := range result {
			result[i] = nil // Explicitly nil, though it's the default for []*float64
		}
		return result
	}

	emaValues := make([]*float64, len(data))
	// Initialize all to nil first, as EMAs are only calculated from period-1 onwards.
	for i := range emaValues {
		emaValues[i] = nil
	}

	if len(data) < period {
		// Not enough data to calculate any EMA (even the first SMA seed).
		// All values in emaValues are already nil, so return as is.
		return emaValues
	}

	// Calculate the initial SMA for the first EMA value
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i].Close
	}
	firstEma := sum / float64(period)
	emaValues[period-1] = &firstEma

	// Calculate multiplier
	// Multiplier (Smoothing Factor) = 2 / (Period + 1)
	multiplier := 2.0 / (float64(period) + 1.0)

	// Calculate subsequent EMAs
	for i := period; i < len(data); i++ {
		// EMA = (Current Price - Previous EMA) * Multiplier + Previous EMA
		// Previous EMA is emaValues[i-1].
		// It's guaranteed to be non-nil here because we start calculating from i = period,
		// and emaValues[period-1] was seeded with the SMA.
		// If data[i-1] was used instead of emaValues[i-1] for previous EMA, it would be incorrect.
		
		// Ensure emaValues[i-1] is not nil before dereferencing.
		// This should always be true due to the seeding at period-1 and sequential calculation.
		// However, defensive check can be useful.
		if emaValues[i-1] == nil {
			// This state implies an issue if period > 0 and len(data) >= period,
			// as the chain of calculation would be broken.
			// Propagate nil to indicate the EMA could not be computed for this point.
			emaValues[i] = nil 
			continue // Or handle as an error depending on strictness
		}
		
		ema := (data[i].Close - *emaValues[i-1]) * multiplier + *emaValues[i-1]
		emaValues[i] = &ema
	}

	return emaValues
}

// CalculateEMAOnValues computes the Exponential Moving Average (EMA) for a given slice of *float64 values.
// This function is useful when calculating an EMA on a series that might already contain nil values,
// such as the MACD line (which is itself derived from EMAs and can have leading nils).
//
// The calculation methodology is similar to CalculateEMA:
// 1. It finds the first sequence of `period` non-nil values to calculate an initial SMA.
//    If such a sequence cannot be found (e.g., too many nils, or not enough data),
//    nils are returned for those points where the EMA cannot be seeded.
// 2. The first EMA value is seeded with this SMA.
// 3. Subsequent EMAs are calculated using the standard formula, propagating nils if the current
//    input value is nil or if the previous EMA was nil.
//
// Parameters:
//   - values: A slice of *float64, representing the values to calculate EMA on.
//     May contain nil entries.
//   - period: An integer for the EMA period. Must be positive.
//
// Returns:
//   - A slice of *float64, representing the EMA values.
//   - Elements are nil where EMA cannot be calculated.
//   - If period is invalid or values slice is empty, a slice of nils of the same length as values is returned.
func CalculateEMAOnValues(values []*float64, period int) []*float64 {
	if period <= 0 || len(values) == 0 {
		// Initialize with nils, length of values.
		// If len(values) is 0, this is an empty slice.
		// If period is invalid, all results are nil.
		emaResults := make([]*float64, len(values))
		for i := range emaResults {
			emaResults[i] = nil
		}
		return emaResults
	}

	emaResults := make([]*float64, len(values))
	for i := range emaResults { // Initialize all to nil
		emaResults[i] = nil
	}
	
	// Find the first valid index (non-nil value) to start potential SMA calculation
	firstValidIndex := -1
	for i, val := range values {
		if val != nil {
			firstValidIndex = i
			break
		}
	}

	// If no non-nil values found, or not enough data points from firstValidIndex for a full period,
	// return the slice of nils.
	if firstValidIndex == -1 || (len(values)-firstValidIndex) < period {
		return emaResults 
	}

	// Attempt to calculate the initial SMA for the first EMA value
	sum := 0.0
	count := 0
	// Iterate for 'period' number of elements starting from firstValidIndex
	// to gather data for the first SMA.
	for i := firstValidIndex; i < firstValidIndex+period; i++ {
		// Check if index is within bounds (it should be due to earlier check, but good for safety)
		if i >= len(values) { 
			return emaResults // Should not happen if (len(values)-firstValidIndex) < period check is correct
		}
		if values[i] != nil {
			sum += *values[i]
			count++
		} else {
			// A nil was encountered within the first 'period' items needed for SMA.
			// Cannot reliably seed SMA, so return nils.
			// This handles interspersed nils in the initial window.
			return emaResults
		}
	}
    
	// If not enough non-nil values were found in the window (e.g. if period was 5, but only 4 non-nils found)
	if count < period {
		return emaResults
	}

	// Seed the first EMA at the end of the first full period of non-nil data
	firstEmaVal := sum / float64(period)
	emaResults[firstValidIndex+period-1] = &firstEmaVal

	// Calculate multiplier
	multiplier := 2.0 / (float64(period) + 1.0)

	// Calculate subsequent EMAs
	// Start from the element immediately after the one where the first EMA was placed.
	for i := firstValidIndex + period; i < len(values); i++ {
		currentValuePtr := values[i]
		prevEmaPtr := emaResults[i-1] // Previous EMA calculated

		if currentValuePtr == nil || prevEmaPtr == nil {
			// If current input value is nil, or previous EMA is nil,
			// the EMA calculation cannot continue for this point. Result remains nil.
			emaResults[i] = nil 
			continue
		}
		
		// EMA = (Current Value - Previous EMA) * Multiplier + Previous EMA
		ema := (*currentValuePtr - *prevEmaPtr) * multiplier + *prevEmaPtr
		emaResults[i] = &ema
	}

	return emaResults
}
