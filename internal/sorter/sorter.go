package sorter

import (
	"sort"

	"github.com/achannarasappa/ticker/v4/internal/analysis" // Import for AnalyzedAsset
	c "github.com/achannarasappa/ticker/v4/internal/common"
)

// Sorter represents a function that sorts quotes
type Sorter func([]*analysis.AnalyzedAsset) []*analysis.AnalyzedAsset // Changed to AnalyzedAsset

// NewSorter creates a sorting function
func NewSorter(sortKey string) Sorter { // Renamed 'sort' to 'sortKey' to avoid conflict
	var sortDict = map[string]Sorter{
		"alpha": sortByAlpha,
		"value": sortByValue,
		"user":  sortByUser,
	}
	if sorter, ok := sortDict[sortKey]; ok { // Use sortKey
		return sorter
	}

	return sortByChange
}

func sortByUser(assets []*analysis.AnalyzedAsset) []*analysis.AnalyzedAsset { // Changed to AnalyzedAsset

	assetCount := len(assets)

	if assetCount <= 0 {
		return assets
	}

	sort.SliceStable(assets, func(i, j int) bool {
		return assets[j].BaseAsset.Meta.OrderIndex > assets[i].BaseAsset.Meta.OrderIndex // Access via BaseAsset
	})

	return assets

}

func sortByAlpha(assetsIn []*analysis.AnalyzedAsset) []*analysis.AnalyzedAsset { // Changed to AnalyzedAsset

	assetCount := len(assetsIn)

	if assetCount <= 0 {
		return assetsIn
	}

	assets := make([]*analysis.AnalyzedAsset, assetCount) // Changed to AnalyzedAsset
	copy(assets, assetsIn)

	sort.SliceStable(assets, func(i, j int) bool {
		return assets[j].BaseAsset.Symbol > assets[i].BaseAsset.Symbol // Access via BaseAsset
	})

	return assets
}

func sortByValue(assetsIn []*analysis.AnalyzedAsset) []*analysis.AnalyzedAsset { // Changed to AnalyzedAsset

	assetCount := len(assetsIn)

	if assetCount <= 0 {
		return assetsIn
	}

	assets := make([]*analysis.AnalyzedAsset, assetCount) // Changed to AnalyzedAsset
	copy(assets, assetsIn)

	activeAssets, inactiveAssets := splitActiveAssets(assets) // splitActiveAssets will also be updated

	sort.SliceStable(inactiveAssets, func(i, j int) bool {
		// Assuming Holding.Value is what's intended for sorting inactive assets.
		// If inactive means no holding, this might need adjustment or clarification.
		return inactiveAssets[j].BaseAsset.Holding.Value < inactiveAssets[i].BaseAsset.Holding.Value // Access via BaseAsset
	})

	sort.SliceStable(activeAssets, func(i, j int) bool {
		return activeAssets[j].BaseAsset.Holding.Value < activeAssets[i].BaseAsset.Holding.Value // Access via BaseAsset
	})

	return append(activeAssets, inactiveAssets...)
}

func sortByChange(assetsIn []*analysis.AnalyzedAsset) []*analysis.AnalyzedAsset { // Changed to AnalyzedAsset

	assetCount := len(assetsIn)

	if assetCount <= 0 {
		return assetsIn
	}

	assets := make([]*analysis.AnalyzedAsset, assetCount) // Changed to AnalyzedAsset
	copy(assets, assetsIn)

	activeAssets, inactiveAssets := splitActiveAssets(assets) // splitActiveAssets will also be updated

	sort.SliceStable(activeAssets, func(i, j int) bool {
		return activeAssets[j].BaseAsset.QuotePrice.ChangePercent < activeAssets[i].BaseAsset.QuotePrice.ChangePercent // Access via BaseAsset
	})

	sort.SliceStable(inactiveAssets, func(i, j int) bool {
		return inactiveAssets[j].BaseAsset.QuotePrice.ChangePercent < inactiveAssets[i].BaseAsset.QuotePrice.ChangePercent // Access via BaseAsset
	})

	return append(activeAssets, inactiveAssets...)
}

// splitActiveAssets separates assets based on whether they have holdings
func splitActiveAssets(assets []*analysis.AnalyzedAsset) ([]*analysis.AnalyzedAsset, []*analysis.AnalyzedAsset) { // Changed to AnalyzedAsset
	activeAssets := make([]*analysis.AnalyzedAsset, 0)   // Changed to AnalyzedAsset
	inactiveAssets := make([]*analysis.AnalyzedAsset, 0) // Changed to AnalyzedAsset
	for _, asset := range assets {
		if asset.BaseAsset.Holding != (c.Holding{}) { // Access via BaseAsset
			activeAssets = append(activeAssets, asset)
		} else {
			inactiveAssets = append(inactiveAssets, asset)
		}
	}
	return activeAssets, inactiveAssets
}
