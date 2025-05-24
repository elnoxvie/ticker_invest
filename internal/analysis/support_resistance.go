package analysis

import (
	"math"
	"github.com/achannarasappa/ticker/v4/internal/indicator" // Assuming HistoricalDataPoint is here
)

// SupportResistanceOutput holds the calculated short-term support and resistance levels.
// Support is the lowest 'Low' price in the lookback period.
// Resistance is the highest 'High' price in the lookback period.
// Both fields are pointers to float64, allowing them to be nil if levels cannot be determined.
type SupportResistanceOutput struct {
	Support    *float64 // Pointer to the calculated support level.
	Resistance *float64 // Pointer to the calculated resistance level.
}

// GetSupportResistance calculates short-term support and resistance levels based on
// the minimum 'Low' and maximum 'High' prices over a specified lookback period.
//
// Support is identified as the lowest 'Low' price encountered within the lookback window.
// Resistance is identified as the highest 'High' price encountered within the lookback window.
//
// Parameters:
//   - data: A slice of indicator.HistoricalDataPoint structs. Each point must have 'Low' and 'High' prices.
//   - lookbackSRDays: The number of recent data points (days) to consider for finding the
//     minimum 'Low' (support) and maximum 'High' (resistance). Must be positive.
//
// Returns:
//   - A SupportResistanceOutput struct.
//   - If `lookbackSRDays` is invalid (<=0) or if `len(data)` is less than `lookbackSRDays`,
//     the `Support` and `Resistance` fields in the returned struct will be nil, indicating
//     that the levels could not be determined.
//   - Otherwise, `Support` will point to the minimum 'Low' and `Resistance` will point to
//     the maximum 'High' in the lookback window.
func GetSupportResistance(
	data []indicator.HistoricalDataPoint,
	lookbackSRDays int,
) SupportResistanceOutput {

	output := SupportResistanceOutput{Support: nil, Resistance: nil}

	if lookbackSRDays <= 0 {
		// Invalid lookback period, cannot determine S/R.
		return output
	}
	
	if len(data) < lookbackSRDays {
		// Not enough data for the lookback period.
		// This also covers len(data) == 0.
		return output
	}

	// Consider the most recent 'lookbackSRDays' data points.
	// startIndex will be len(data) - lookbackSRDays.
	startIndex := len(data) - lookbackSRDays
	window := data[startIndex:] // This creates a slice of the last 'lookbackSRDays' elements.

	// This check is technically redundant if len(data) >= lookbackSRDays and lookbackSRDays > 0,
	// as window would always have at least one element if lookbackSRDays is 1.
	// However, it's a harmless safeguard.
	if len(window) == 0 { 
		return output
	}

	minLow := math.MaxFloat64
	maxHigh := -math.MaxFloat64 // Smallest possible float value using -math.MaxFloat64

	// The problem description implies HistoricalDataPoint.Low and .High are float64,
	// so they are always present and are not pointers. No nil checks needed for point.Low/point.High themselves.
	// The hasLow/hasHigh flags are good for robustness in case a window could be empty
	// or if future versions of HistoricalDataPoint might have nullable Low/High.
	// Given current constraints, they might seem redundant if len(window) > 0 is guaranteed.
	// However, keeping them for clarity or future-proofing is fine.
	
	// Since we've established len(window) >= lookbackSRDays and lookbackSRDays > 0,
	// the loop will run at least once. So, hasLow and hasHigh will become true.
	// The initial values of minLow/maxHigh are appropriate for finding min/max.

	for _, point := range window {
		if point.Low < minLow {
			minLow = point.Low
		}
		if point.High > maxHigh {
			maxHigh = point.High
		}
	}

	// If the loop ran (which it will if len(window) > 0), minLow and maxHigh will hold valid values.
	// We don't need hasLow/hasHigh if len(window) is guaranteed > 0.
	// The initial check for `len(data) < lookbackSRDays` ensures `window` is not empty.
	output.Support = &minLow
	output.Resistance = &maxHigh
	
	return output
}
