package analysis

import (
	"github.com/achannarasappa/ticker/v4/internal/indicator" // For HistoricalDataPoint
	// "fmt" // For any debugging, remove if not used
	// "math" // For IsNaN if checking *float64 values, not strictly needed if just comparing pointers to nil
)

// GetConsolidatedDecision generates a trading decision (e.g., "BUY", "SELL", "HOLD")
// based on a combination of technical indicators and analysis results.
//
// This function integrates various signals:
// - Trend status and volume surge information from `TrendVolumeOutput`.
// - Support and resistance levels from `SupportResistanceOutput`.
// - MACD conditions (bullish/bearish cross, momentum).
// - RSI conditions (turning up/down from low/high, oversold/overbought).
// - SMA cross conditions (golden/death cross).
// - Price action relative to support/resistance levels, buffered by an ATR multiple.
//
// The decision logic is based on a predefined set of rules that prioritize certain
// signal combinations (e.g., strong breakout with volume and momentum).
//
// Parameters:
//   - data: A slice of indicator.HistoricalDataPoint. The last two points are used to
//     evaluate conditions like MACD crossovers or RSI changes. Indicators (MACD, RSI, ATR, SMAs)
//     should be populated in these data points.
//   - srOutput: The SupportResistanceOutput struct containing calculated support and resistance levels.
//   - tvOutput: The TrendVolumeOutput struct containing trend and volume analysis.
//   - atrMultiplierBuf: A float64 multiplier for the Average True Range (ATR) to create
//     buffers around support and resistance levels for breakout/breakdown signals.
//
// Returns:
//   - A string representing the trading decision, often including Rich formatting tags
//     for UI display (e.g., "[bold green]STRONG BUY (Breakout)[/bold green]").
//   - Returns "N/A (Insufficient Data)" if `len(data) < 2`.
//   - Default decision is "HOLD" if no specific buy/sell signals are met.
func GetConsolidatedDecision(
	data []indicator.HistoricalDataPoint, // Slice of historical data, expecting indicators to be populated
	srOutput SupportResistanceOutput, // Support and Resistance levels
	tvOutput TrendVolumeOutput, // Trend and Volume analysis results
	atrMultiplierBuf float64,
) string {

	if len(data) < 2 { // Need at least last and previous data points for some conditions (MACD cross, RSI change)
		return "N/A (Insufficient Data)"
	}

	last := data[len(data)-1]
	prev := data[len(data)-2]

	// ATR for dynamic buffers
	// Default to 0.5% of close price if ATR is not available.
	atrVal := 0.005 * last.Close 
	if last.ATR != nil { // Check if ATR is calculated and not nil
		atrVal = *last.ATR
	}
	supBuffer := atrVal * atrMultiplierBuf
	resBuffer := atrVal * atrMultiplierBuf

	// --- MACD Conditions ---
	macdBullishCross := false
	if last.MACDHist != nil && prev.MACDHist != nil && *last.MACDHist > 0 && *prev.MACDHist <= 0 {
		macdBullishCross = true
	}
	// macd_bearish_cross (unused in Python decision tree for now, but for completeness if needed later)
	// if last.MACDHist != nil && prev.MACDHist != nil && *last.MACDHist < 0 && *prev.MACDHist >= 0 {
	// 	 macd_bearish_cross = true
	// }
	macdBullishMomentum := false
	if last.MACDHist != nil && *last.MACDHist > 0 {
		macdBullishMomentum = true
	}
	macdBearishMomentum := false
	if last.MACDHist != nil && *last.MACDHist < 0 {
		macdBearishMomentum = true
	}

	// --- RSI Conditions ---
	rsiTurningUpFromLow := false
	if last.RSI != nil && prev.RSI != nil && *last.RSI < 40 && *last.RSI > *prev.RSI {
		rsiTurningUpFromLow = true
	}
	rsiTurningDownFromHigh := false
	if last.RSI != nil && prev.RSI != nil && *last.RSI > 60 && *last.RSI < *prev.RSI {
		rsiTurningDownFromHigh = true
	}
	rsiOversold := false
	if last.RSI != nil && *last.RSI < 30 {
		rsiOversold = true
	}
	rsiOverbought := false
	if last.RSI != nil && *last.RSI > 70 {
		rsiOverbought = true
	}
    
    // --- SMA Cross conditions ---
    sma50CrossedSma200Bullish := false
    if prev.SMA50 != nil && prev.SMA200 != nil && last.SMA50 != nil && last.SMA200 != nil {
        if *prev.SMA50 <= *prev.SMA200 && *last.SMA50 > *last.SMA200 {
            sma50CrossedSma200Bullish = true
        }
    }
    sma50CrossedSma200Bearish := false
    if prev.SMA50 != nil && prev.SMA200 != nil && last.SMA50 != nil && last.SMA200 != nil {
        if *prev.SMA50 >= *prev.SMA200 && *last.SMA50 < *last.SMA200 {
            sma50CrossedSma200Bearish = true
        }
    }

	// --- BUY SIGNALS ---
	// 1. Strong Breakout Buy
	if srOutput.Resistance != nil && tvOutput.IsStrongUpTrend && last.Close > (*srOutput.Resistance+resBuffer) && tvOutput.IsVolumeSurge && macdBullishMomentum {
		return "[bold green]STRONG BUY (Breakout)[/bold green]"
	}

	// 2. Bounce from Support Buy
	if srOutput.Support != nil && tvOutput.IsStrongUpTrend && last.Low <= (*srOutput.Support+supBuffer) && rsiTurningUpFromLow && macdBullishMomentum {
		return "[green]BUY (Bounce on Support)[/green]"
	}
    
    // 3. Trend Following Buy (Golden Cross + Confirmation)
    if sma50CrossedSma200Bullish && tvOutput.IsStrongUpTrend && macdBullishMomentum && (last.RSI != nil && !rsiOverbought) { // Ensure RSI is not nil
         return "[green]BUY (Golden Cross Confirmation)[/green]"
    }

	// 4. General Trend & Momentum Buy
	if tvOutput.IsStrongUpTrend && macdBullishMomentum && last.RSI != nil && *last.RSI > 40 && *last.RSI < 65 {
		return "[green]BUY (Trend & Momentum)[/green]"
	}
       
	// --- SELL SIGNALS ---
	// 1. Strong Breakdown Sell
	if srOutput.Support != nil && tvOutput.IsStrongDownTrend && last.Close < (*srOutput.Support-supBuffer) && tvOutput.IsVolumeSurge && macdBearishMomentum {
		return "[bold red]STRONG SELL (Breakdown)[/bold red]"
	}

	// 2. Reversal from Resistance Sell
	if srOutput.Resistance != nil && tvOutput.IsStrongDownTrend && last.High >= (*srOutput.Resistance-resBuffer) && rsiTurningDownFromHigh && macdBearishMomentum {
		return "[red]SELL (Reversal at Resistance)[/red]"
	}
    
    // 3. Trend Following Sell (Death Cross + Confirmation)
    if sma50CrossedSma200Bearish && tvOutput.IsStrongDownTrend && macdBearishMomentum && (last.RSI != nil && !rsiOversold) { // Ensure RSI is not nil
        return "[red]SELL (Death Cross Confirmation)[/red]"
    }
       
	// 4. General Trend Weakening / Momentum Sell
    priceBelowSMA50 := false
    if last.SMA50 != nil { // Check SMA50 is not nil before dereferencing
        priceBelowSMA50 = last.Close < *last.SMA50
    }
	if (tvOutput.IsStrongDownTrend || priceBelowSMA50) && macdBearishMomentum && last.RSI != nil && *last.RSI < 60 && *last.RSI > 35 {
		return "[red]SELL (Trend & Momentum)[/red]"
	}

	// --- HOLD / CAUTION SIGNALS ---
	if rsiOverbought && tvOutput.IsStrongUpTrend {
		return "[yellow]HOLD (Overbought - Caution)[/yellow]"
	}
	if rsiOversold && tvOutput.IsStrongDownTrend {
		return "[yellow]HOLD (Oversold - Caution)[/yellow]"
	}
	if rsiOverbought && !tvOutput.IsStrongUpTrend { // Overbought in non-uptrend
		return "[orange]SELL (Overbought Reversal Risk)[/orange]"
	}
	if rsiOversold && !tvOutput.IsStrongDownTrend { // Oversold in non-downtrend
		return "[light_green]BUY (Oversold Bounce Risk)[/light_green]"
	}

	return "HOLD" // Default decision
}
