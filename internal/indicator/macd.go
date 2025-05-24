package indicator

// CalculateMACD computes the Moving Average Convergence Divergence (MACD) indicator,
// which includes the MACD line, Signal line, and MACD Histogram.
//
// The MACD line is the difference between a fast EMA and a slow EMA of the 'Close' prices.
// The Signal line is an EMA of the MACD line.
// The MACD Histogram is the MACD line minus the Signal line.
//
// Parameters:
//   - data: A slice of HistoricalDataPoint structs.
//   - fastPeriod: The period for the fast Exponential Moving Average (EMA). Typically 12.
//   - slowPeriod: The period for the slow Exponential Moving Average (EMA). Typically 26.
//     Must be greater than fastPeriod.
//   - signalPeriod: The period for the EMA of the MACD line (the Signal line). Typically 9.
//
// Returns:
//   - macdLine: A slice of *float64 representing the MACD line.
//   - signalLine: A slice of *float64 representing the Signal line.
//   - macdHistogram: A slice of *float64 representing the MACD Histogram.
//   - All returned slices will have the same length as the input data. Elements will be nil
//     where the respective value cannot be calculated (e.g., due to insufficient data or
//     if underlying EMAs are nil).
//   - If input parameters are invalid (e.g., periods <= 0, fastPeriod >= slowPeriod,
//     or data is empty), all three returned slices will be slices of nils with the
//     same length as the input data.
func CalculateMACD(
	data []HistoricalDataPoint,
	fastPeriod int,
	slowPeriod int,
	signalPeriod int,
) (macdLineReturn []*float64, signalLineReturn []*float64, macdHistogramReturn []*float64) {

	numDataPoints := len(data)
	// Initialize slices of nils if input is invalid or insufficient
	// Also covers fastPeriod >= slowPeriod as an invalid configuration for standard MACD.
	if numDataPoints == 0 || fastPeriod <= 0 || slowPeriod <= 0 || signalPeriod <= 0 || fastPeriod >= slowPeriod {
		macdLineReturn = make([]*float64, numDataPoints)
		signalLineReturn = make([]*float64, numDataPoints)
		macdHistogramReturn = make([]*float64, numDataPoints)
		// Explicitly fill with nils, though this is the default for []*float64
		for i := 0; i < numDataPoints; i++ {
			macdLineReturn[i] = nil
			signalLineReturn[i] = nil
			macdHistogramReturn[i] = nil
		}
		return
	}

	// Calculate Fast and Slow EMAs on the 'Close' price
	// CalculateEMA is in the same package (ema.go)
	fastEMA := CalculateEMA(data, fastPeriod)
	slowEMA := CalculateEMA(data, slowPeriod)

	// Calculate MACD Line
	// MACD Line = Fast EMA - Slow EMA
	macdLine := make([]*float64, numDataPoints)
	for i := 0; i < numDataPoints; i++ {
		if fastEMA[i] != nil && slowEMA[i] != nil {
			// Ensure both EMAs are available to calculate MACD
			val := *fastEMA[i] - *slowEMA[i]
			macdLine[i] = &val
		} else {
			macdLine[i] = nil
		}
	}

	// Calculate Signal Line (EMA of MACD Line)
	// CalculateEMAOnValues is in the same package (ema.go)
	signalLine := CalculateEMAOnValues(macdLine, signalPeriod)

	// Calculate MACD Histogram
	// MACD Histogram = MACD Line - Signal Line
	macdHistogram := make([]*float64, numDataPoints)
	for i := 0; i < numDataPoints; i++ {
		// Ensure both MACD line and Signal line are available
		if macdLine[i] != nil && signalLine[i] != nil {
			val := *macdLine[i] - *signalLine[i]
			macdHistogram[i] = &val
		} else {
			macdHistogram[i] = nil
		}
	}

	return macdLine, signalLine, macdHistogram
}
