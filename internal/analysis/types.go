// Package analysis provides functions for higher-level stock analysis,
// including trend detection, support/resistance calculation, and generating consolidated trading decisions.
// It combines raw data and technical indicators to produce actionable insights.
package analysis

import (
	"github.com/achannarasappa/ticker/v4/internal/common"
	"github.com/achannarasappa/ticker/v4/internal/indicator"
)

// AnalysisResults holds all the higher-level analysis data derived for an asset.
// This includes trend status, volume analysis, support/resistance levels,
// the final trading decision, and the most recent historical data point with all indicators.
type AnalysisResults struct {
	TrendStatus       string   // Textual description of the current trend (e.g., "Strong Uptrend").
	VolumeStatus      string   // Textual description of the current volume status (e.g., "SURGE").
	IsStrongUpTrend   bool     // True if a strong uptrend is detected.
	IsStrongDownTrend bool     // True if a strong downtrend is detected.
	IsVolumeSurge     bool     // True if a significant volume surge is detected.
	PriceAboveSMA50   bool     // True if the current price is above the 50-period SMA.
	PriceAboveSMA200  bool     // True if the current price is above the 200-period SMA.
	Support           *float64 // Calculated short-term support level. Nil if not determinable.
	Resistance        *float64 // Calculated short-term resistance level. Nil if not determinable.
	Decision          string   // The final consolidated trading decision (e.g., "BUY", "SELL", "HOLD").
	DecisionSummary   string   // AI-generated summary for the decision.
	TrendSummary      string   // AI-generated summary for the trend.
	// LastHistoricalPointWithIndicators stores the latest historical data point,
	// which includes all calculated technical indicators. This is useful for displaying
	// current indicator values in the UI.
	LastHistoricalPointWithIndicators *indicator.HistoricalDataPoint
	// FullHistoricalDataWithIndicators could optionally store the full series if needed for detailed UI views or further processing.
	// FullHistoricalDataWithIndicators []indicator.HistoricalDataPoint
}

// AnalyzedAsset combines the base asset data (e.g., symbol, name, current quote)
// with its comprehensive technical analysis results.
type AnalyzedAsset struct {
	BaseAsset common.Asset    // Embeds common.Asset to include its fields like Symbol, Name, QuotePrice, etc.
	Analysis  AnalysisResults // Contains all the derived analysis data and the final decision.
}
