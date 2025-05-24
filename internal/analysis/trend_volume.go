package analysis

import (
	"fmt" // For string formatting if needed for status messages
	"math" // For IsNaN if checking *float64 values - not used here, but good practice if needed for float pointers

	"github.com/achannarasappa/ticker/v4/internal/indicator" // Assuming HistoricalDataPoint is here
)

// TrendVolumeOutput holds the results of trend and volume analysis.
// It includes textual descriptions of the trend and volume status,
// boolean flags for specific conditions (e.g., strong uptrend, volume surge),
// and flags indicating the price's position relative to key SMAs.
type TrendVolumeOutput struct {
	TrendStatus         string // Describes the overall market trend (e.g., "Strong Uptrend", "Mixed").
	VolumeStatus        string // Describes the current volume relative to its average (e.g., "SURGE", "Below Avg").
	IsStrongUpTrend     bool   // True if the asset is in a strong uptrend (Price > SMA50, SMA50 > SMA200).
	IsStrongDownTrend   bool   // True if the asset is in a strong downtrend (Price < SMA50, SMA50 < SMA200).
	IsVolumeSurge       bool   // True if the current volume is significantly higher than its recent average.
	PriceAboveSMA50     bool   // True if the last closing price is above the 50-period SMA.
	PriceAboveSMA200    bool   // True if the last closing price is above the 200-period SMA.
}

// GetTrendAndVolumeStatus determines the current trend and volume status for a given
// series of historical data points.
//
// Trend is determined by comparing the last closing price to its 50-period (SMA50)
// and 200-period (SMA200) Simple Moving Averages. It identifies strong uptrends
// (Price > SMA50 and SMA50 > SMA200) and strong downtrends (Price < SMA50 and SMA50 < SMA200),
// as well as other general uptrend/downtrend or mixed conditions.
//
// Volume status is determined by comparing the last day's volume to the average volume
// over a specified lookback period. A volume surge is flagged if the current volume
// exceeds the average volume by a given multiplier.
//
// Parameters:
//   - data: A slice of indicator.HistoricalDataPoint, which should have SMA50 and SMA200
//     populated for the last data point for full trend analysis. Volume data is also required.
//   - lookbackSRDays: The number of recent days to consider for calculating the average volume.
//     Defaults to 20 if an invalid (<=0) value is provided.
//   - volumeSurgeMultiplier: The multiplier for the average volume to detect a volume surge.
//     Defaults to 1.5 if an invalid (<=0) value is provided.
//
// Returns:
//   - A TrendVolumeOutput struct containing the determined trend and volume statuses and flags.
//     Status fields default to "N/A" if calculations cannot be performed (e.g., insufficient data).
func GetTrendAndVolumeStatus(
	data []indicator.HistoricalDataPoint,
	lookbackSRDays int,
	volumeSurgeMultiplier float64,
) TrendVolumeOutput {

	output := TrendVolumeOutput{
		TrendStatus:  "N/A",
		VolumeStatus: "N/A",
	}

	// Initial checks for data sufficiency
	if len(data) < 1 { // Need at least one data point for the 'last' element
		return output
	}
	// If len(data) is 1, SMA checks will proceed, but volume check might say "Short History"
	// Python original had < 2, but some basic checks can be done with 1 point if SMAs are pre-calculated.
	// The volume lookback check is more specific later.

	if lookbackSRDays <= 0 { // Basic validation for lookback period
		lookbackSRDays = 20 // Default if invalid
	}
	if volumeSurgeMultiplier <= 0 { // Basic validation for multiplier
		volumeSurgeMultiplier = 1.5 // Default
	}

	last := data[len(data)-1]

	// --- Trend Determination ---
	sma50Valid := last.SMA50 != nil && !math.IsNaN(*last.SMA50)
	sma200Valid := last.SMA200 != nil && !math.IsNaN(*last.SMA200)

	if sma50Valid {
		output.PriceAboveSMA50 = last.Close > *last.SMA50
	}
	if sma200Valid {
		output.PriceAboveSMA200 = last.Close > *last.SMA200
	}

	// Stronger trend definitions
	if sma50Valid && sma200Valid {
		// Strong Uptrend: Price > SMA50 AND SMA50 > SMA200
		if output.PriceAboveSMA50 && (*last.SMA50 > *last.SMA200) {
			output.IsStrongUpTrend = true
		}
		// Strong Downtrend: Price < SMA50 AND SMA50 < SMA200
		// (Note: Python script has `last.Close < last.SMA50` redundant if `!output.PriceAboveSMA50` implies it)
		if !output.PriceAboveSMA50 && (*last.SMA50 < *last.SMA200) {
			output.IsStrongDownTrend = true
		}
	}
	// If only one SMA is valid, strong trend cannot be determined in the same way.
	// IsStrongUpTrend/IsStrongDownTrend will remain false.

	// Determine TrendStatus string based on available data
	if output.IsStrongUpTrend {
		output.TrendStatus = "[bold green]Strong Uptrend[/bold green]"
	} else if output.IsStrongDownTrend {
		output.TrendStatus = "[bold red]Strong Downtrend[/bold red]"
	} else if sma50Valid && sma200Valid { // Both SMAs are valid, but not a "strong" trend by above definition
		if output.PriceAboveSMA50 && output.PriceAboveSMA200 { // Price > SMA50 and Price > SMA200
			output.TrendStatus = "[green]Uptrend (Price > SMAs)[/green]"
		} else if !output.PriceAboveSMA50 && !output.PriceAboveSMA200 { // Price < SMA50 and Price < SMA200
			output.TrendStatus = "[red]Downtrend (Price < SMAs)[/red]"
		} else if output.PriceAboveSMA50 && !output.PriceAboveSMA200 { // Price > SMA50 but Price < SMA200
			output.TrendStatus = "[yellow]Mixed (Price between SMAs, SMA50 > SMA200 ?)[/yellow]" // Or could be more specific
		} else if !output.PriceAboveSMA50 && output.PriceAboveSMA200 { // Price < SMA50 but Price > SMA200 (SMA50 must be < SMA200)
			output.TrendStatus = "[yellow]Mixed (Price between SMAs, SMA50 < SMA200 ?)[/yellow]"
		} else {
			output.TrendStatus = "Sideways/Mixed (SMAs Valid)"
		}
	} else if sma50Valid { // Only SMA50 is valid
		if output.PriceAboveSMA50 {
			output.TrendStatus = "[light_green]Uptrend (Price > SMA50)[/light_green]"
		} else { // Price < SMA50
			output.TrendStatus = "[light_red]Downtrend (Price < SMA50)[/light_red]"
		}
	} else if sma200Valid { // Only SMA200 is valid (and SMA50 is not)
		if output.PriceAboveSMA200 {
			output.TrendStatus = "[light_green]Uptrend (Price > SMA200)[/light_green]"
		} else { // Price < SMA200
			output.TrendStatus = "[light_red]Downtrend (Price < SMA200)[/light_red]"
		}
	} else { // No SMAs are valid
		output.TrendStatus = "N/A (No SMA Data)"
	}


	// --- Volume Check ---
	// Ensure there are enough data points for the lookback period for volume average.
	// The Python script's avg_vol_5 = df['Volume'].tail(LOOKBACK_SR).mean() includes current day in average.
	// So, we need at least `lookbackSRDays` data points in `data`.
	// If len(data) < lookbackSRDays, average calculation might be on less than `lookbackSRDays` points.
	
	if len(data) == 0 { // Should have been caught by earlier len(data) < 1, but double check for volume logic.
		output.VolumeStatus = "N/A (No Data)"
		return output // Return early as 'last' would not be valid.
	}

	volSum := int64(0)
	actualDaysForAvg := 0
	
	// Determine start index for volume sum to average last `lookbackSRDays` elements including current.
	volStartIndex := len(data) - lookbackSRDays
	if volStartIndex < 0 {
		volStartIndex = 0 // Average all available data if less than lookbackSRDays
	}

	for k := volStartIndex; k < len(data); k++ {
		volSum += data[k].Volume
		actualDaysForAvg++
	}

	if actualDaysForAvg > 0 {
		avgVolume := float64(volSum) / float64(actualDaysForAvg)
		if math.IsNaN(avgVolume) || avgVolume == 0 { // Handle avgVolume being zero or NaN
			output.VolumeStatus = "N/A (Avg Vol Invalid)"
		} else if last.Volume > int64(avgVolume*volumeSurgeMultiplier) {
			output.IsVolumeSurge = true
			// Ensure volumeSurgeMultiplier is part of the string if it's dynamic
			output.VolumeStatus = fmt.Sprintf("[bold yellow]SURGE (Vol > %.1fx Avg)[/bold yellow]", volumeSurgeMultiplier)
		} else if last.Volume > int64(avgVolume) {
			output.VolumeStatus = "[green]Above Avg[/green]"
		} else if last.Volume < int64(avgVolume) {
			output.VolumeStatus = "[red]Below Avg[/red]"
		} else { // last.Volume == int64(avgVolume)
			output.VolumeStatus = "Avg"
		}
	} else { // This case should ideally not be hit if len(data) >= 1
		output.VolumeStatus = "N/A (No Vol Data for Avg)"
	}
	
	// Refine VolumeStatus if not enough data for a *full* lookback, but some data existed
	if actualDaysForAvg < lookbackSRDays && actualDaysForAvg > 0 && output.VolumeStatus != "N/A (Avg Vol Invalid)" {
		if output.IsVolumeSurge {
			output.VolumeStatus = fmt.Sprintf("[bold yellow]SURGE (Vol > %.1fx Avg, %d-day)[/bold yellow]", volumeSurgeMultiplier, actualDaysForAvg)
		} else {
			// Append to existing status
			output.VolumeStatus += fmt.Sprintf(" (%d-day avg)", actualDaysForAvg)
		}
	} else if actualDaysForAvg == 0 { // Should be caught by len(data) < 1 or lead to (No Vol Data for Avg)
        output.VolumeStatus = "N/A (Insufficient Vol Data)";
    }


	return output
}
