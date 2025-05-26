// Package asset provides functionalities for managing and processing financial assets,
// including aggregating lot data, calculating holding summaries, and orchestrating
// the retrieval of quote data, historical data, and technical analysis for assets.
package asset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time" // Added for http.Client timeout

	"github.com/achannarasappa/ticker/v4/internal/analysis"
	c "github.com/achannarasappa/ticker/v4/internal/common"
	"github.com/achannarasappa/ticker/v4/internal/indicator" // For indicator.HistoricalPoint
	"github.com/achannarasappa/ticker/v4/internal/monitor/yahoo/historical"
	"github.com/achannarasappa/ticker/v4/internal/monitor/yahoo/unary"
	u "github.com/achannarasappa/ticker/v4/internal/ui/util" // For StripRichTags
)

// AggregatedLot represents a cost basis lot of an asset grouped by symbol.
// It sums up the cost and quantity from multiple individual lots of the same asset.
type AggregatedLot struct {
	Symbol     string  // The ticker symbol of the asset.
	Cost       float64 // Total cost of all aggregated lots.
	Quantity   float64 // Total quantity of the asset.
	OrderIndex int     // Original order index, primarily for maintaining user-defined order.
}

// HoldingSummary represents a consolidated summary of all asset holdings.
// It includes the total market value, total cost basis, and overall change (total and day).
type HoldingSummary struct {
	Value       float64         // Total current market value of all holdings.
	Cost        float64         // Total cost basis of all holdings.
	TotalChange c.HoldingChange // Overall change (value - cost) in amount and percentage.
	DayChange   c.HoldingChange // Total change for the current trading day in amount and percentage.
}

// currencyRateByUse holds different currency conversion rates applicable for various contexts
// such as current quote price, historical position cost, and summary calculations.
// This struct helps in applying the correct conversion rate when dealing with assets
// in different currencies, ensuring that all financial values can be consistently
// displayed or aggregated in a target currency.
type currencyRateByUse struct {
	QuotePrice     float64 // Rate for converting current prices (e.g., market price, day's high/low).
	PositionCost   float64 // Rate for converting historical costs of positions.
	SummaryValue   float64 // Rate for converting current values for summary aggregation.
	SummaryCost    float64 // Rate for converting cost basis for summary aggregation.
	ToCurrencyCode string  // The target currency code after conversion.
}

// getCurrencyRateByUse determines the appropriate currency conversion rates based on the asset's original
// currency, the application's target display currency, and any available direct conversion rates.
// It checks if a direct conversion to the target display currency is needed and possible.
// If not, it attempts to find a rate via the context's pre-loaded currency rates.
// If no suitable conversion is found, it defaults to a rate of 1.0 (no conversion).
//
// Parameters:
//   - ctx: The application context, containing configuration (like DisplayCurrency) and pre-fetched CurrencyRates.
//   - fromCurrency: The original currency code of the asset/value.
//   - toCurrencyActual: The currency code to which `rate` applies, if `rate` is provided (e.g., from an API response).
//   - rate: A direct conversion rate (fromCurrency -> toCurrencyActual) that might be available.
//
// Returns:
//   - A currencyRateByUse struct populated with the determined rates for different use cases
//     and the effective target currency code.
func getCurrencyRateByUse(ctx c.Context, fromCurrency, toCurrencyActual string, rate float64) currencyRateByUse {
	targetDisplayCurrency := strings.ToUpper(ctx.Config.Currency)
	fromCurrency = strings.ToUpper(fromCurrency)

	finalRate := 1.0
	effectiveToCurrency := fromCurrency

	if fromCurrency != targetDisplayCurrency {
		if strings.ToUpper(toCurrencyActual) == targetDisplayCurrency {
			finalRate = rate
			effectiveToCurrency = targetDisplayCurrency
		} else {
			// TODO: Resolve how CurrencyRates should be accessed or passed to this function.
			// if cr, ok := ctx.CurrencyRates[fromCurrency]; ok && cr.ToCurrency == targetDisplayCurrency {
			// 	finalRate = cr.Rate
			// 	effectiveToCurrency = targetDisplayCurrency
			// } else {
			// 	finalRate = 1.0
			// 	effectiveToCurrency = fromCurrency
			// }
		}
	} else {
		effectiveToCurrency = targetDisplayCurrency
	}

	return currencyRateByUse{
		QuotePrice:     finalRate,
		PositionCost:   finalRate,
		SummaryValue:   finalRate,
		SummaryCost:    finalRate,
		ToCurrencyCode: effectiveToCurrency,
	}
}

// convertAssetQuotePriceCurrency applies the determined conversion rate to the financial fields
// within an c.QuotePrice struct. This ensures that price-related values are displayed
// in the target currency.
//
// Parameters:
//   - rates: The currencyRateByUse struct containing the appropriate conversion rate for quote prices.
//   - qp: The c.QuotePrice struct whose fields are to be converted.
//
// Returns:
//   - An c.QuotePrice struct with its financial fields adjusted to the target currency.
//     The ChangePercent field remains unchanged as it's a relative value.
func convertAssetQuotePriceCurrency(rates currencyRateByUse, qp c.QuotePrice) c.QuotePrice {
	qp.Price *= rates.QuotePrice
	qp.Change *= rates.QuotePrice
	qp.PriceDayHigh *= rates.QuotePrice
	qp.PriceDayLow *= rates.QuotePrice
	qp.PriceOpen *= rates.QuotePrice
	qp.PricePrevClose *= rates.QuotePrice
	return qp
}

// convertAssetQuoteExtendedCurrency applies the determined conversion rate to the financial fields
// within an c.QuoteExtended struct, such as market capitalization and 52-week high/low.
//
// Parameters:
//   - rates: The currencyRateByUse struct containing the appropriate conversion rate.
//   - qe: The c.QuoteExtended struct whose fields are to be converted.
//
// Returns:
//   - An c.QuoteExtended struct with its financial fields adjusted to the target currency.
func convertAssetQuoteExtendedCurrency(rates currencyRateByUse, qe c.QuoteExtended) c.QuoteExtended {
	qe.MarketCap *= rates.QuotePrice
	qe.FiftyTwoWeekHigh *= rates.QuotePrice
	qe.FiftyTwoWeekLow *= rates.QuotePrice
	return qe
}

// GetAnalyzedAssets is a core orchestration function that takes a group of asset quotes,
// fetches necessary additional data (like historical prices), performs technical analysis,
// and returns a slice of fully analyzed assets along with a summary of holdings.
//
// The process for each asset involves:
//  1. Initializing a base asset structure from the provided quote.
//  2. Applying currency conversions to financial data based on the application's display currency.
//  3. Fetching historical market data for the asset using the provided `unaryAPI` client.
//     Historical data range and interval are determined by application configuration with defaults.
//  4. If historical data is successfully retrieved:
//     a. Calculating various technical indicators (SMA, EMA, RSI, MACD, ATR) based on configured periods.
//     b. Populating these indicators back into the historical data points.
//     c. Performing higher-level analysis:
//     i. Determining trend and volume status.
//     ii. Calculating support and resistance levels.
//     iii. Generating a consolidated trading decision based on all gathered data and analysis.
//  5. Consolidating all information into an `analysis.AnalyzedAsset` struct.
//  6. Aggregating holding summaries (total value, cost, changes).
//  7. Calculating holding weights for each asset relative to the total portfolio value.
//  8. Fetching AI-generated summaries for the decision and trend.
//
// Parameters:
//   - ctx: The application context, containing configuration (e.g., display currency, analysis parameters)
//     and other shared resources.
//   - assetGroupQuote: Contains the group of assets to be processed, along with their current quotes.
//   - unaryAPI: A pointer to an initialized `unary.UnaryAPI` client, used for fetching historical
//     data from the Yahoo Finance API. This client handles API session management (cookies, crumb).
//
// Returns:
//   - A slice of `analysis.AnalyzedAsset` structs, where each struct contains the original asset data
//     enriched with calculated technical indicators, analysis results (trend, S/R, volume), and a
//     trading decision.
//   - A `HoldingSummary` struct, providing an aggregated view of all holdings processed, including
//     total market value, cost basis, and overall day/total changes.

// AIAPIRequest defines the structure for the request to the AI completions endpoint.
type AIAPIRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// AIAPIResponseChoice defines a single choice in the AI API response.
type AIAPIResponseChoice struct {
	Text string `json:"text"`
}

// AIAPIResponse defines the structure for the response from the AI completions endpoint.
type AIAPIResponse struct {
	Choices []AIAPIResponseChoice `json:"choices"`
}

// getAISummary calls the local AI API to get a summary for a given prompt.
func getAISummary(ctx c.Context, httpClient *http.Client, prompt string) string { // Removed logger, will use ctx.Logger; Added ctx

	// Check if AI is enabled in the config
	if ctx.Config.AIAPI.Enabled != nil && !*ctx.Config.AIAPI.Enabled {
		if ctx.Logger != nil {
			ctx.Logger.Printf("AI queries are disabled in the configuration.")
		}
		return "" // Return empty if AI is explicitly disabled
	}
	// If Enabled is nil (not set in YAML), also treat as disabled by default.
	if ctx.Config.AIAPI.Enabled == nil {
		if ctx.Logger != nil {
			ctx.Logger.Printf("AI queries are disabled by default (ai_api.enabled not set in config).")
		}
		return "" // Return empty if AI is not explicitly enabled
	}

	// Use configured AI API settings, with defaults
	endpoint := ctx.Config.AIAPI.Endpoint
	if endpoint == "" {
		endpoint = "http://10.5.0.2:1234/v1/completions" // Default endpoint
	}
	modelName := ctx.Config.AIAPI.ModelName
	if modelName == "" {
		modelName = "local-model" // Default model name
	}
	maxTokens := ctx.Config.AIAPI.MaxTokens
	if maxTokens == 0 {
		maxTokens = 150 // Default max tokens
	}
	temperature := ctx.Config.AIAPI.Temperature
	// Default temperature is 0.7, allow 0 as a valid value if explicitly set.
	// So, we only apply default if it's not present in a way that implies zero was intended.
	// However, typical YAML unmarshalling would set it to 0 if omitted and it's a float64.
	// For simplicity, if it's 0.0 and not explicitly set, we might assume default.
	// A better way would be to use pointers in AIConfig for optional values or check if the key exists.
	// For now, we'll assume if it's 0, use default, unless we enhance AIConfig to differentiate.
	if temperature == 0.0 { // Assuming 0.0 means use default, adjust if 0 is a valid non-default T.
		temperature = 0.7 // Default temperature
	}

	requestPayload := AIAPIRequest{
		Model:       modelName,
		Prompt:      prompt,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Printf("Error marshalling AI API request: %v", err)
		}
		return ""
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Printf("Error creating AI API request: %v", err)
		}
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Printf("Error sending AI API request: %v", err)
		}
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if ctx.Logger != nil {
			ctx.Logger.Printf("AI API request failed with status %s", resp.Status)
		}
		return ""
	}

	var apiResponse AIAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Printf("Error decoding AI API response: %v", err)
		}
		return ""
	}

	// print the AI API response for debugging
	if ctx.Logger != nil {
		ctx.Logger.Printf("AI API response: %+v", apiResponse)
	}

	if len(apiResponse.Choices) > 0 && apiResponse.Choices[0].Text != "" {
		responseText := apiResponse.Choices[0].Text
		// Attempt to remove the original prompt if the response starts with it.
		// Also trim any leading/trailing whitespace from the result.
		cleanedText := strings.TrimSpace(strings.TrimPrefix(responseText, prompt))
		return cleanedText
	}

	if ctx.Logger != nil {
		ctx.Logger.Printf("No summary found in AI API response or choices array empty")
	}
	return ""
}

// buildAIPrompt constructs a detailed prompt for the AI API.
func buildAIPrompt(symbol, analysisType, decisionOrTrend string, price float64, indicators *indicator.HistoricalDataPoint, support, resistance *float64, volume float64) string { // Changed indicator.HistoricalPoint to indicator.HistoricalDataPoint
	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf("Explain why the %s for stock symbol %s is %s. ", analysisType, symbol, decisionOrTrend))
	promptBuilder.WriteString("Consider the following technical indicators: ")
	promptBuilder.WriteString(fmt.Sprintf("Current Price: %.2f. ", price))

	if indicators != nil {
		if indicators.SMA50 != nil {
			promptBuilder.WriteString(fmt.Sprintf("SMA50: %.2f. ", *indicators.SMA50))
		}
		if indicators.SMA200 != nil {
			promptBuilder.WriteString(fmt.Sprintf("SMA200: %.2f. ", *indicators.SMA200))
		}
		if indicators.RSI != nil {
			promptBuilder.WriteString(fmt.Sprintf("RSI: %.2f. ", *indicators.RSI))
		}
		if indicators.MACD != nil {
			promptBuilder.WriteString(fmt.Sprintf("MACD: %.2f. ", *indicators.MACD))
		}
		// Add other indicators as needed, e.g., ATR, MACDSignal, MACDHist
	}
	if support != nil {
		promptBuilder.WriteString(fmt.Sprintf("Support: %.2f. ", *support))
	}
	if resistance != nil {
		promptBuilder.WriteString(fmt.Sprintf("Resistance: %.2f. ", *resistance))
	}
	if volume != 0 { // Assuming QuoteExtended.Volume is the relevant volume
		promptBuilder.WriteString(fmt.Sprintf("Volume: %.0f. ", volume))
	}
	promptBuilder.WriteString("Provide a concise summary in less than 5 lines.")
	return promptBuilder.String()
}

func GetAnalyzedAssets(
	ctx c.Context,
	assetGroupQuote c.AssetGroupQuote,
	unaryAPI *unary.UnaryAPI, // Client to fetch historical data
) ([]analysis.AnalyzedAsset, HoldingSummary) {

	// Retrieve analysis and data parameters from config, with defaults
	confLookbackSR := ctx.Config.AnalysisLookbackSRDays
	if confLookbackSR == 0 {
		confLookbackSR = 20
	}
	confAtrPeriod := ctx.Config.AnalysisAtrPeriod
	if confAtrPeriod == 0 {
		confAtrPeriod = 14
	}
	confAtrMultiplier := ctx.Config.AnalysisAtrMultiplier
	if confAtrMultiplier == 0.0 {
		confAtrMultiplier = 0.25
	}
	confVolumeSurge := ctx.Config.AnalysisVolumeSurge
	if confVolumeSurge == 0.0 {
		confVolumeSurge = 1.5
	}
	confSma50Period := ctx.Config.AnalysisSma50Period
	if confSma50Period == 0 {
		confSma50Period = 50
	}
	confSma200Period := ctx.Config.AnalysisSma200Period
	if confSma200Period == 0 {
		confSma200Period = 200
	}
	confRsiPeriod := ctx.Config.AnalysisRsiPeriod
	if confRsiPeriod == 0 {
		confRsiPeriod = 14
	}
	confMacdFastPeriod := ctx.Config.AnalysisMacdFastPeriod
	if confMacdFastPeriod == 0 {
		confMacdFastPeriod = 12
	}
	confMacdSlowPeriod := ctx.Config.AnalysisMacdSlowPeriod
	if confMacdSlowPeriod == 0 {
		confMacdSlowPeriod = 26
	}
	confMacdSignalPeriod := ctx.Config.AnalysisMacdSignalPeriod
	if confMacdSignalPeriod == 0 {
		confMacdSignalPeriod = 9
	}
	confHistDataRange := ctx.Config.HistoricalDataRange
	if confHistDataRange == "" {
		confHistDataRange = "1y"
	}
	confHistDataInterval := ctx.Config.HistoricalDataInterval
	if confHistDataInterval == "" {
		confHistDataInterval = "1d"
	}

	var holdingSummary HoldingSummary
	analyzedAssets := make([]analysis.AnalyzedAsset, 0)
	holdingsBySymbol := getLots(assetGroupQuote.AssetGroup.ConfigAssetGroup.Holdings)
	orderIndex := make(map[string]int)

	for i, symbol := range assetGroupQuote.AssetGroup.ConfigAssetGroup.Watchlist {
		if _, exists := orderIndex[symbol]; !exists {
			orderIndex[strings.ToLower(symbol)] = i
		}
	}
	for i, symbol := range assetGroupQuote.AssetGroup.ConfigAssetGroup.Holdings {
		if _, exists := orderIndex[symbol.Symbol]; !exists {
			orderIndex[strings.ToLower(symbol.Symbol)] = i + len(assetGroupQuote.AssetGroup.ConfigAssetGroup.Watchlist)
		}
	}

	for _, assetQuote := range assetGroupQuote.AssetQuotes {
		baseAsset := c.Asset{
			Name:          assetQuote.Name,
			Symbol:        assetQuote.Symbol,
			Class:         assetQuote.Class,
			Currency:      assetQuote.Currency,
			QuotePrice:    assetQuote.QuotePrice,
			QuoteExtended: assetQuote.QuoteExtended,
			QuoteFutures:  assetQuote.QuoteFutures,
			QuoteSource:   assetQuote.QuoteSource,
			Exchange:      assetQuote.Exchange,
			Meta: c.Meta{
				IsVariablePrecision: assetQuote.Meta.IsVariablePrecision,
				SymbolInSourceAPI:   assetQuote.Meta.SymbolInSourceAPI,
				OrderIndex:          orderIndex[strings.ToLower(assetQuote.Symbol)],
			},
		}

		currencyRateByUseVals := getCurrencyRateByUse(ctx, assetQuote.Currency.FromCurrencyCode, assetQuote.Currency.ToCurrencyCode, assetQuote.Currency.Rate)
		baseAsset.Currency.ToCurrencyCode = currencyRateByUseVals.ToCurrencyCode
		baseAsset.QuotePrice = convertAssetQuotePriceCurrency(currencyRateByUseVals, baseAsset.QuotePrice)
		baseAsset.QuoteExtended = convertAssetQuoteExtendedCurrency(currencyRateByUseVals, baseAsset.QuoteExtended)
		baseAsset.Holding = getHoldingFromAssetQuote(assetQuote, holdingsBySymbol, currencyRateByUseVals)

		analysisRes := analysis.AnalysisResults{Decision: "N/A (Analysis Pending)"}
		histData, err := historical.FetchAndProcessHistoricalData(unaryAPI, assetQuote.Symbol, confHistDataRange, confHistDataInterval)

		// Initialize HTTP client for AI API calls
		// It's generally better to initialize this once and reuse, but for simplicity here,
		// we create it per asset or it could be passed into GetAnalyzedAssets.
		// For production, consider a shared client.
		aiClient := &http.Client{}
		if ctx.Config.AIAPI.RequestTimeoutSeconds > 0 {
			aiClient.Timeout = time.Duration(ctx.Config.AIAPI.RequestTimeoutSeconds) * time.Second
		} else {
			aiClient.Timeout = 10 * time.Second // Default timeout
		}

		if err == nil && len(histData) > 0 {
			sma50Values := indicator.CalculateSMA(histData, confSma50Period)
			sma200Values := indicator.CalculateSMA(histData, confSma200Period)
			rsiValues := indicator.CalculateRSI(histData, confRsiPeriod)
			macdLine, macdSignal, macdHist := indicator.CalculateMACD(histData, confMacdFastPeriod, confMacdSlowPeriod, confMacdSignalPeriod)
			atrValues := indicator.CalculateATR(histData, confAtrPeriod)

			for i := range histData {
				if i < len(sma50Values) {
					histData[i].SMA50 = sma50Values[i]
				}
				if i < len(sma200Values) {
					histData[i].SMA200 = sma200Values[i]
				}
				if i < len(rsiValues) {
					histData[i].RSI = rsiValues[i]
				}
				if i < len(macdLine) {
					histData[i].MACD = macdLine[i]
				}
				if i < len(macdSignal) {
					histData[i].MACDSignal = macdSignal[i]
				}
				if i < len(macdHist) {
					histData[i].MACDHist = macdHist[i]
				}
				if i < len(atrValues) {
					histData[i].ATR = atrValues[i]
				}
			}

			// This is the main success path: err == nil && len(histData) > 0
			// Populate indicators into histData (already done in the loop above this search block)

			// Safely set LastHistoricalPointWithIndicators
			// This 'if' is technically redundant due to the outer 'if len(histData) > 0',
			// but it's a good safeguard if histData could be modified elsewhere (it's not here).
			if len(histData) > 0 {
				analysisRes.LastHistoricalPointWithIndicators = &histData[len(histData)-1]
			} else {
				// This case should ideally not be hit due to the outer check.
				if ctx.Logger != nil {
					ctx.Logger.Printf("Historical data became empty unexpectedly before analysis for %s", assetQuote.Symbol)
				}
				analysisRes.Decision = "[grey]N/A (Processing Issue)[/grey]"
				analysisRes.TrendStatus = "N/A"
				analysisRes.VolumeStatus = "N/A"
				// Skip further analysis for this asset
				analyzedAssets = append(analyzedAssets, analysis.AnalyzedAsset{BaseAsset: baseAsset, Analysis: analysisRes})
				holdingSummary = addHoldingToHoldingSummary(holdingSummary, baseAsset.Holding, currencyRateByUseVals)
				continue // Continue to the next asset
			}

			minDataForSomeAnalysis := confRsiPeriod + 1
			if len(histData) >= minDataForSomeAnalysis {
				tvOutput := analysis.GetTrendAndVolumeStatus(histData, confLookbackSR, confVolumeSurge)
				srOutput := analysis.GetSupportResistance(histData, confLookbackSR)

				analysisRes.TrendStatus = tvOutput.TrendStatus
				analysisRes.VolumeStatus = tvOutput.VolumeStatus
				analysisRes.IsStrongUpTrend = tvOutput.IsStrongUpTrend
				analysisRes.IsStrongDownTrend = tvOutput.IsStrongDownTrend
				analysisRes.IsVolumeSurge = tvOutput.IsVolumeSurge
				analysisRes.PriceAboveSMA50 = tvOutput.PriceAboveSMA50
				analysisRes.PriceAboveSMA200 = tvOutput.PriceAboveSMA200
				analysisRes.Support = srOutput.Support
				analysisRes.Resistance = srOutput.Resistance
				analysisRes.Decision = analysis.GetConsolidatedDecision(histData, srOutput, tvOutput, confAtrMultiplier)

				// Fetch AI Summaries after core analysis is complete
				// Only fetch if AI is enabled
				if ctx.Config.AIAPI.Enabled != nil && *ctx.Config.AIAPI.Enabled {
					if analysisRes.Decision != "" && !strings.Contains(analysisRes.Decision, "N/A") {
						decisionPrompt := buildAIPrompt(
							baseAsset.Symbol,
							"investment decision",
							u.StripRichTags(analysisRes.Decision),
							baseAsset.QuotePrice.Price,
							analysisRes.LastHistoricalPointWithIndicators,
							analysisRes.Support,
							analysisRes.Resistance,
							baseAsset.QuoteExtended.Volume,
						)
						analysisRes.DecisionSummary = getAISummary(ctx, aiClient, decisionPrompt) // Pass ctx
					}

					if analysisRes.TrendStatus != "" && !strings.Contains(analysisRes.TrendStatus, "N/A") {
						trendPrompt := buildAIPrompt(
							baseAsset.Symbol,
							"current trend",
							u.StripRichTags(analysisRes.TrendStatus),
							baseAsset.QuotePrice.Price,
							analysisRes.LastHistoricalPointWithIndicators,
							analysisRes.Support,
							analysisRes.Resistance,
							baseAsset.QuoteExtended.Volume,
						)
						analysisRes.TrendSummary = getAISummary(ctx, aiClient, trendPrompt) // Pass ctx
					}
				} else {
					if ctx.Logger != nil {
						ctx.Logger.Printf("Skipping AI summary generation as AI is disabled.")
					}
				}

			} else {
				analysisRes.Decision = "N/A (Insufficient Hist. Data for Analysis)"
			}
		} else {
			if err != nil {
				analysisRes.Decision = "[grey]N/A (Hist. Data Error)[/grey]"
			} else {
				analysisRes.Decision = "[grey]N/A (No Hist. Data)[/grey]"
			}
		}

		analyzedAssets = append(analyzedAssets, analysis.AnalyzedAsset{
			BaseAsset: baseAsset,
			Analysis:  analysisRes,
		})

		holdingSummary = addHoldingToHoldingSummary(holdingSummary, baseAsset.Holding, currencyRateByUseVals)
	}

	if holdingSummary.Value != 0 {
		for i := range analyzedAssets {
			if analyzedAssets[i].BaseAsset.Holding.Value != 0 {
				analyzedAssets[i].BaseAsset.Holding.Weight = (analyzedAssets[i].BaseAsset.Holding.Value / holdingSummary.Value) * 100
			} else {
				analyzedAssets[i].BaseAsset.Holding.Weight = 0
			}
		}
	}

	return analyzedAssets, holdingSummary
}

func addHoldingToHoldingSummary(holdingSummary HoldingSummary, holding c.Holding, currencyRateByUseVals currencyRateByUse) HoldingSummary {
	if holding.Cost == 0 && holding.Value == 0 {
		return holdingSummary
	}

	value := holdingSummary.Value + (holding.Value * currencyRateByUseVals.SummaryValue)
	cost := holdingSummary.Cost + (holding.Cost * currencyRateByUseVals.SummaryCost)
	dayChangeAmount := holdingSummary.DayChange.Amount + (holding.DayChange.Amount * currencyRateByUseVals.SummaryValue)

	var totalChangePercent, dayChangePercent float64
	if cost != 0 {
		totalChangePercent = ((value - cost) / cost) * 100
	}
	if value != 0 {
		dayChangePercent = (dayChangeAmount / value) * 100
	}

	return HoldingSummary{
		Value: value,
		Cost:  cost,
		TotalChange: c.HoldingChange{
			Amount:  value - cost,
			Percent: totalChangePercent,
		},
		DayChange: c.HoldingChange{
			Amount:  dayChangeAmount,
			Percent: dayChangePercent,
		},
	}
}

func getHoldingFromAssetQuote(assetQuote c.AssetQuote, lotsBySymbol map[string]AggregatedLot, currencyRateByUseVals currencyRateByUse) c.Holding {
	if aggregatedLot, ok := lotsBySymbol[assetQuote.Symbol]; ok {
		valueDisplay := aggregatedLot.Quantity * assetQuote.QuotePrice.Price
		costDisplay := aggregatedLot.Cost * currencyRateByUseVals.PositionCost

		totalChangeAmountDisplay := valueDisplay - costDisplay
		var totalChangePercent float64
		if costDisplay != 0 {
			totalChangePercent = (totalChangeAmountDisplay / costDisplay) * 100
		}

		dayChangeAmountDisplay := assetQuote.QuotePrice.Change * aggregatedLot.Quantity

		unitValueDisplay := 0.0
		if aggregatedLot.Quantity != 0 {
			unitValueDisplay = valueDisplay / aggregatedLot.Quantity
		}
		unitCostDisplay := 0.0
		if aggregatedLot.Quantity != 0 {
			unitCostDisplay = costDisplay / aggregatedLot.Quantity
		}

		return c.Holding{
			Value:     valueDisplay,
			Cost:      costDisplay,
			Quantity:  aggregatedLot.Quantity,
			UnitValue: unitValueDisplay,
			UnitCost:  unitCostDisplay,
			DayChange: c.HoldingChange{
				Amount:  dayChangeAmountDisplay,
				Percent: assetQuote.QuotePrice.ChangePercent,
			},
			TotalChange: c.HoldingChange{
				Amount:  totalChangeAmountDisplay,
				Percent: totalChangePercent,
			},
			Weight: 0,
		}
	}
	return c.Holding{}
}

func getLots(lots []c.Lot) map[string]AggregatedLot {
	if lots == nil {
		return map[string]AggregatedLot{}
	}
	aggregatedLots := map[string]AggregatedLot{}
	for i, lot := range lots {
		aggregatedLot, ok := aggregatedLots[lot.Symbol]
		if !ok {
			aggregatedLots[lot.Symbol] = AggregatedLot{
				Symbol:     lot.Symbol,
				Cost:       (lot.UnitCost * lot.Quantity) + lot.FixedCost,
				Quantity:   lot.Quantity,
				OrderIndex: i,
			}
		} else {
			aggregatedLot.Quantity += lot.Quantity
			aggregatedLot.Cost += (lot.UnitCost * lot.Quantity) + lot.FixedCost
			aggregatedLots[lot.Symbol] = aggregatedLot
		}
	}
	return aggregatedLots
}

func updateHoldingWeights(assets []c.Asset, holdingSummary HoldingSummary) []c.Asset {
	if holdingSummary.Value == 0 {
		return assets
	}
	for i, asset := range assets {
		assets[i].Holding.Weight = (asset.Holding.Value / holdingSummary.Value) * 100
	}
	return assets
}
