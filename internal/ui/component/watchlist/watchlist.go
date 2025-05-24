package watchlist

import (
	"fmt"
	"strings"

	"github.com/achannarasappa/ticker/v4/internal/analysis" // Added
	c "github.com/achannarasappa/ticker/v4/internal/common"
	s "github.com/achannarasappa/ticker/v4/internal/sorter"
	row "github.com/achannarasappa/ticker/v4/internal/ui/component/watchlist/row"
	u "github.com/achannarasappa/ticker/v4/internal/ui/util"

	tea "github.com/charmbracelet/bubbletea"
)

// Config represents the configuration for the watchlist component
type Config struct {
	Separate              bool
	ShowHoldings          bool
	ExtraInfoExchange     bool
	ExtraInfoFundamentals bool
	Sort                  string
	Styles                c.Styles
}

// Model for watchlist section
type Model struct {
	width          int
	assets         []*analysis.AnalyzedAsset // Changed
	assetsBySymbol map[string]*analysis.AnalyzedAsset // Changed
	sorter         s.Sorter
	config         Config
	cellWidths     row.CellWidthsContainer
	rows           []*row.Model
	rowsBySymbol   map[string]*row.Model
}

// Messages for replacing assets
type SetAssetsMsg []analysis.AnalyzedAsset // Changed

// Messages for updating assets
// UpdateAssetsMsg []c.Asset // This type is not used in the provided code, SetAssetsMsg is used for full updates.
// If individual updates were needed, this would also change.

// NewModel returns a model with default values
func NewModel(config Config) *Model {
	return &Model{
		width:          80,
		config:         config,
		assets:         make([]*analysis.AnalyzedAsset, 0), // Changed
		assetsBySymbol: make(map[string]*analysis.AnalyzedAsset), // Changed
		sorter:         s.NewSorter(config.Sort),
		rowsBySymbol:   make(map[string]*row.Model),
	}
}

// Init initializes the watchlist
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the watchlist
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SetAssetsMsg:

		var cmd tea.Cmd
		cmds := make([]tea.Cmd, 0)

		// Convert []analysis.AnalyzedAsset to []*analysis.AnalyzedAsset and update assetsBySymbol map
		assets := make([]*analysis.AnalyzedAsset, len(msg)) // Changed type
		assetsBySymbol := make(map[string]*analysis.AnalyzedAsset) // Changed type

		for i := range msg {
			assets[i] = &msg[i]
			assetsBySymbol[msg[i].BaseAsset.Symbol] = assets[i] // Changed to use BaseAsset.Symbol
		}

		assets = m.sorter(assets)

		for i, asset := range assets { // asset is now *analysis.AnalyzedAsset
			if i < len(m.rows) {
				// row.UpdateAssetMsg(asset) now takes *analysis.AnalyzedAsset
				m.rows[i], cmd = m.rows[i].Update(row.UpdateAssetMsg(asset))
				cmds = append(cmds, cmd)
				m.rowsBySymbol[assets[i].BaseAsset.Symbol] = m.rows[i] // Changed to use BaseAsset.Symbol
			} else {
				// row.New now takes *analysis.AnalyzedAsset
				m.rows = append(m.rows, row.New(row.Config{
					Separate:              m.config.Separate,
					ExtraInfoExchange:     m.config.ExtraInfoExchange,
					ExtraInfoFundamentals: m.config.ExtraInfoFundamentals,
					ShowHoldings:          m.config.ShowHoldings,
					Styles:                m.config.Styles,
					Asset:                 asset, // asset is *analysis.AnalyzedAsset
				}))
				m.rowsBySymbol[assets[i].BaseAsset.Symbol] = m.rows[len(m.rows)-1] // Changed to use BaseAsset.Symbol
			}
		}

		if len(assets) < len(m.rows) {
			m.rows = m.rows[:len(assets)]
		}

		m.assets = assets
		m.assetsBySymbol = assetsBySymbol

		// TODO: only set conditionally if all assets have changed
		m.cellWidths = getCellWidths(m.assets)
		for i, r := range m.rows {
			m.rows[i], _ = r.Update(row.SetCellWidthsMsg{
				Width:      m.width,
				CellWidths: m.cellWidths,
			})
		}

		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:

		m.width = msg.Width
		m.cellWidths = getCellWidths(m.assets)
		for i, r := range m.rows {
			m.rows[i], _ = r.Update(row.SetCellWidthsMsg{
				Width:      m.width,
				CellWidths: m.cellWidths,
			})
		}

		return m, nil

	case row.FrameMsg:

		var cmd tea.Cmd
		cmds := make([]tea.Cmd, 0)

		// TODO: send message to a specific row rather than all rows
		for i, r := range m.rows {
			m.rows[i], cmd = r.Update(msg)
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)

	}

	return m, nil
}

// View rendering hook for bubbletea
func (m *Model) View() string {

	if m.width < 80 {
		return fmt.Sprintf("Terminal window too narrow to render content\nResize to fix (%d/80)", m.width)
	}

	rows := make([]string, 0)
	for _, row := range m.rows {
		rows = append(rows, row.View())
	}

	return strings.Join(rows, "\n")

}
func getCellWidths(assets []*analysis.AnalyzedAsset) row.CellWidthsContainer { // Changed parameter type

	cellMaxWidths := row.CellWidthsContainer{}

	for _, asset := range assets { // asset is now *analysis.AnalyzedAsset
		var quoteLength int

		// Access fields via asset.BaseAsset
		volumeMarketCapLength := len(u.ConvertFloatToString(asset.BaseAsset.QuoteExtended.MarketCap, true))

		if asset.BaseAsset.QuoteExtended.FiftyTwoWeekHigh == 0.0 {
			quoteLength = len(u.ConvertFloatToString(asset.BaseAsset.QuotePrice.Price, asset.BaseAsset.Meta.IsVariablePrecision))
		}

		if asset.BaseAsset.QuoteExtended.FiftyTwoWeekHigh != 0.0 {
			quoteLength = len(u.ConvertFloatToString(asset.BaseAsset.QuoteExtended.FiftyTwoWeekHigh, asset.BaseAsset.Meta.IsVariablePrecision))
		}

		if volumeMarketCapLength > cellMaxWidths.WidthVolumeMarketCap {
			cellMaxWidths.WidthVolumeMarketCap = volumeMarketCapLength
		}

		if quoteLength > cellMaxWidths.QuoteLength {
			cellMaxWidths.QuoteLength = quoteLength
			cellMaxWidths.WidthQuote = quoteLength + row.WidthChangeStatic
			cellMaxWidths.WidthQuoteExtended = quoteLength
			cellMaxWidths.WidthQuoteRange = row.WidthRangeStatic + (quoteLength * 2)
		}

		if asset.BaseAsset.Holding != (c.Holding{}) {
			positionLength := len(u.ConvertFloatToString(asset.BaseAsset.Holding.Value, asset.BaseAsset.Meta.IsVariablePrecision))
			positionQuantityLength := len(u.ConvertFloatToString(asset.BaseAsset.Holding.Quantity, asset.BaseAsset.Meta.IsVariablePrecision))

			if positionLength > cellMaxWidths.PositionLength {
				cellMaxWidths.PositionLength = positionLength
				cellMaxWidths.WidthPosition = positionLength + row.WidthChangeStatic + row.WidthPositionGutter
			}

			if positionLength > cellMaxWidths.WidthPositionExtended {
				cellMaxWidths.WidthPositionExtended = positionLength
			}

			if positionQuantityLength > cellMaxWidths.WidthPositionExtended {
				cellMaxWidths.WidthPositionExtended = positionQuantityLength
			}

		}

	}

	return cellMaxWidths

}
