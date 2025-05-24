package indicator

// CalculateSMA computes the Simple Moving Average (SMA) for the 'Close' price
// of a series of historical data points over a specified period.
//
// The SMA is calculated by taking the arithmetic mean of the 'Close' prices
// over a sliding window of the given period.
//
// Parameters:
//   - data: A slice of HistoricalDataPoint structs, representing the time series data.
//     Each point must have at least a 'Close' price.
//   - period: An integer representing the number of data points to include in the SMA calculation.
//     If period is less than or equal to 0, or if there's insufficient data for the period,
//     the function will return a slice of nils.
//
// Returns:
//   - A slice of *float64, where each element is a pointer to the calculated SMA value for the
//     corresponding data point in the input slice.
//   - Elements will be nil for data points at the beginning of the series where the SMA cannot
//     be calculated due to insufficient preceding data for the given period.
//   - If the input data is empty or the period is invalid (<=0), a slice of nils of the same
//     length as the input data will be returned.
func CalculateSMA(data []HistoricalDataPoint, period int) []*float64 {
	if period <= 0 || len(data) == 0 {
		// Return a slice of nils with the same length as data if period is invalid or data is empty.
		// If len(data) is 0, this makes an empty slice, which is fine.
		// If len(data) > 0 but period is invalid, all points get nil SMA.
		result := make([]*float64, len(data))
		for i := range result {
			result[i] = nil // Explicitly set to nil, though this is default for []*float64
		}
		return result
	}

	smaValues := make([]*float64, len(data))
	sum := 0.0

	for i := 0; i < len(data); i++ {
		sum += data[i].Close // Add current close price to sum

		if i >= period {
			// Window is full, subtract the element that is sliding out
			sum -= data[i-period].Close 
			
			avg := sum / float64(period)
			smaValues[i] = &avg
		} else if i == period-1 { 
			// Window has just become full, this is the first point where SMA can be calculated
			avg := sum / float64(period)
			smaValues[i] = &avg
		} else {
			// Not enough data yet for this point to calculate SMA
			smaValues[i] = nil
		}
	}
	return smaValues
}
