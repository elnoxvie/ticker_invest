package print //nolint:predeclared

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"

	"github.com/achannarasappa/ticker/v4/internal/analysis" // Added for AnalyzedAsset
	"github.com/achannarasappa/ticker/v4/internal/asset"
	c "github.com/achannarasappa/ticker/v4/internal/common"
	mon "github.com/achannarasappa/ticker/v4/internal/monitor"
	"github.com/achannarasappa/ticker/v4/internal/ui/util"

	"github.com/spf13/cobra"
)

// Options to configure print behavior
type Options struct {
	Format string
}

type jsonRow struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Price    string `json:"price"`
	Value    string `json:"value"`
	Cost     string `json:"cost"`
	Quantity string `json:"quantity"`
	Weight   string `json:"weight"`
	// Added fields for analysis results
	Decision     string `json:"decision,omitempty"`
	TrendStatus  string `json:"trend_status,omitempty"`
	VolumeStatus string `json:"volume_status,omitempty"`
	Support      string `json:"support,omitempty"`
	Resistance   string `json:"resistance,omitempty"`
}

type jsonSummary struct {
	TotalValue         string `json:"total_value"`
	TotalCost          string `json:"total_cost"`
	DayChangeAmount    string `json:"day_change_amount"`
	DayChangePercent   string `json:"day_change_percent"`
	TotalChangeAmount  string `json:"total_change_amount"`
	TotalChangePercent string `json:"total_change_percent"` // Added string type
}

func convertAnalyzedAssetsToCSV(analyzedAssets []analysis.AnalyzedAsset) string { // Changed parameter type
	rows := [][]string{
		{"name", "symbol", "price", "value", "cost", "quantity", "weight", "decision", "trend_status", "volume_status", "support", "resistance"}, // Added analysis headers
	}

	for _, analyzedAsset := range analyzedAssets { // Iterate over analyzedAssets
		asset := analyzedAsset.BaseAsset // Extract BaseAsset
		if asset.Holding.Quantity > 0 {
			rows = append(rows, []string{
				asset.Name,
				asset.Symbol,
				util.ConvertFloatToString(asset.QuotePrice.Price, true),
				util.ConvertFloatToString(asset.Holding.Value, true),
				util.ConvertFloatToString(asset.Holding.Cost, true),
				util.ConvertFloatToString(asset.Holding.Quantity, true),
				util.ConvertFloatToString(asset.Holding.Weight, true),
				// Add analysis data
				analyzedAsset.Analysis.Decision,
				analyzedAsset.Analysis.TrendStatus,
				analyzedAsset.Analysis.VolumeStatus,
				fmt.Sprintf("%.2f", analyzedAsset.Analysis.Support),    // Format float
				fmt.Sprintf("%.2f", analyzedAsset.Analysis.Resistance), // Format float
			})
		}
	}

	b := new(bytes.Buffer)
	w := csv.NewWriter(b)
	//nolint:errcheck
	w.WriteAll(rows)

	return b.String()

}

func convertAnalyzedAssetsToJSON(analyzedAssets []analysis.AnalyzedAsset) string { // Changed parameter type
	var rows []jsonRow

	for _, analyzedAsset := range analyzedAssets { // Iterate over analyzedAssets
		asset := analyzedAsset.BaseAsset // Extract BaseAsset
		if asset.Holding.Quantity > 0 {
			rows = append(rows, jsonRow{
				Name:     asset.Name,
				Symbol:   asset.Symbol,
				Price:    fmt.Sprintf("%f", asset.QuotePrice.Price),
				Value:    fmt.Sprintf("%f", asset.Holding.Value),
				Cost:     fmt.Sprintf("%f", asset.Holding.Cost),
				Quantity: fmt.Sprintf("%f", asset.Holding.Quantity),
				Weight:   fmt.Sprintf("%f", asset.Holding.Weight),
				// Add analysis data
				Decision:     analyzedAsset.Analysis.Decision,
				TrendStatus:  analyzedAsset.Analysis.TrendStatus,
				VolumeStatus: analyzedAsset.Analysis.VolumeStatus,
				Support:      fmt.Sprintf("%.2f", analyzedAsset.Analysis.Support),
				Resistance:   fmt.Sprintf("%.2f", analyzedAsset.Analysis.Resistance),
			})
		}
	}

	if len(rows) == 0 {
		return "[]"
	}

	out, err := json.Marshal(rows)

	if err != nil {
		return err.Error()
	}

	return string(out)

}

func convertSummaryToJSON(summary asset.HoldingSummary) string {
	row := jsonSummary{
		TotalValue:         fmt.Sprintf("%f", summary.Value),
		TotalCost:          fmt.Sprintf("%f", summary.Cost),
		DayChangeAmount:    fmt.Sprintf("%f", summary.DayChange.Amount),
		DayChangePercent:   fmt.Sprintf("%f", summary.DayChange.Percent),
		TotalChangeAmount:  fmt.Sprintf("%f", summary.TotalChange.Amount),
		TotalChangePercent: fmt.Sprintf("%f", summary.TotalChange.Percent),
	}

	out, err := json.Marshal(row)

	if err != nil {
		return err.Error()
	}

	return string(out)
}

func convertSummaryToCSV(summary asset.HoldingSummary) string {
	rows := [][]string{
		{"total_value", "total_cost", "day_change_amount", "day_change_percent", "total_change_amount", "total_change_percent"},
		{
			fmt.Sprintf("%f", summary.Value),
			fmt.Sprintf("%f", summary.Cost),
			fmt.Sprintf("%f", summary.DayChange.Amount),
			fmt.Sprintf("%f", summary.DayChange.Percent),
			fmt.Sprintf("%f", summary.TotalChange.Amount),
			fmt.Sprintf("%f", summary.TotalChange.Percent),
		},
	}

	b := new(bytes.Buffer)
	w := csv.NewWriter(b)
	//nolint:errcheck
	w.WriteAll(rows)

	return b.String()
}

// Run prints holdings to the terminal
func Run(dep *c.Dependencies, ctx *c.Context, options *Options) func(*cobra.Command, []string) {
	return func(_ *cobra.Command, _ []string) {

		monitors, _ := mon.NewMonitor(mon.ConfigMonitor{
			RefreshInterval: ctx.Config.RefreshInterval,
			ConfigMonitorsYahoo: mon.ConfigMonitorsYahoo{
				BaseURL:           dep.MonitorYahooBaseURL,
				SessionRootURL:    dep.MonitorYahooSessionRootURL,
				SessionCrumbURL:   dep.MonitorYahooSessionCrumbURL,
				SessionConsentURL: dep.MonitorYahooSessionConsentURL,
			},
			ConfigMonitorPriceCoinbase: mon.ConfigMonitorPriceCoinbase{
				BaseURL:      dep.MonitorPriceCoinbaseBaseURL,
				StreamingURL: dep.MonitorPriceCoinbaseStreamingURL,
			},
		})
		monitors.SetAssetGroup(ctx.Groups[0], 0) //nolint:errcheck
		assetGroupQuote := monitors.GetAssetGroupQuote()
		// Use GetAnalyzedAssets and handle the new return types
		analyzedAssets, _ := asset.GetAnalyzedAssets(*ctx, assetGroupQuote, monitors.UnaryAPIYahoo) // Changed to access the field directly

		if options.Format == "csv" {
			fmt.Println(convertAnalyzedAssetsToCSV(analyzedAssets)) // Use new CSV conversion function

			return
		}

		fmt.Println(convertAnalyzedAssetsToJSON(analyzedAssets)) // Use new JSON conversion function
	}
}

// RunSummary handles the print summary command
func RunSummary(dep *c.Dependencies, ctx *c.Context, options *Options) func(cmd *cobra.Command, args []string) {
	return func(_ *cobra.Command, _ []string) {

		monitors, _ := mon.NewMonitor(mon.ConfigMonitor{
			RefreshInterval: ctx.Config.RefreshInterval,
			ConfigMonitorsYahoo: mon.ConfigMonitorsYahoo{
				BaseURL:           dep.MonitorYahooBaseURL,
				SessionRootURL:    dep.MonitorYahooSessionRootURL,
				SessionCrumbURL:   dep.MonitorYahooSessionCrumbURL,
				SessionConsentURL: dep.MonitorYahooSessionConsentURL,
			},
		})
		monitors.SetAssetGroup(ctx.Groups[0], 0) //nolint:errcheck
		assetGroupQuote := monitors.GetAssetGroupQuote()
		// Use GetAnalyzedAssets and handle the new return types
		_, holdingSummary := asset.GetAnalyzedAssets(*ctx, assetGroupQuote, monitors.UnaryAPIYahoo) // Changed to access the field directly

		if options.Format == "csv" {
			fmt.Println(convertSummaryToCSV(holdingSummary))

			return
		}

		fmt.Println(convertSummaryToJSON(holdingSummary))
	}
}
