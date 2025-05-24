package row

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/achannarasappa/ticker/v4/internal/analysis" // Added
	c "github.com/achannarasappa/ticker/v4/internal/common"
	u "github.com/achannarasappa/ticker/v4/internal/ui/util"

	grid "github.com/achannarasappa/term-grid"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	WidthMarketState    = 5
	WidthGutter         = 1
	WidthLabel          = 15
	WidthName           = 20
	WidthPositionGutter = 2
	WidthChangeStatic   = 12 // "↓ " + " (100.00%)" = 12 length
	WidthRangeStatic    = 3  // " - " = 3 length
)

var lastID int64 //nolint:gochecknoglobals

type SetCellWidthsMsg struct {
	Width      int
	CellWidths CellWidthsContainer
}

type CellWidthsContainer struct {
	PositionLength        int
	QuoteLength           int
	WidthQuote            int
	WidthQuoteExtended    int
	WidthQuoteRange       int
	WidthPosition         int
	WidthPositionExtended int
	WidthVolumeMarketCap  int
}

type Config struct {
	ID                    int
	Separate              bool
	ShowHoldings          bool
	ExtraInfoExchange     bool
	ExtraInfoFundamentals bool
	Styles                c.Styles
	Asset                 *analysis.AnalyzedAsset // Changed
}

type UpdateAssetMsg *analysis.AnalyzedAsset // Changed

type FrameMsg int

// Model for watchlist row
type Model struct {
	id                   int
	width                int
	config               Config // Config.Asset is now *analysis.AnalyzedAsset
	cellWidths           CellWidthsContainer
	frame                int
	priceStyle           lipgloss.Style
	priceChangeSegment   string
	priceNoChangeSegment string
	priceChangeDirection int
}

// New returns a model with default values
func New(config Config) *Model { // config.Asset is *analysis.AnalyzedAsset

	var id int

	if config.ID != 0 {
		id = config.ID
	} else {
		id = nextID()
	}

	return &Model{
		id:                   id,
		width:                80,
		config:               config, // Store the config which has the AnalyzedAsset
		priceNoChangeSegment: u.ConvertFloatToString(config.Asset.BaseAsset.QuotePrice.Price, config.Asset.BaseAsset.Meta.IsVariablePrecision), // Access via BaseAsset
		priceChangeSegment:   "",
	}
}

// Init initializes the watchlist row
func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {

	switch msg := msg.(type) {
	case SetCellWidthsMsg:
		m.cellWidths = msg.CellWidths
		m.width = msg.Width

		return m, nil

	case UpdateAssetMsg: // msg is now *analysis.AnalyzedAsset

		// If symbol has not changed and price has changed then start the price animation
		// Access fields via BaseAsset
		if m.config.Asset.BaseAsset.Symbol == msg.BaseAsset.Symbol && m.config.Asset.BaseAsset.QuotePrice.Price != msg.BaseAsset.QuotePrice.Price {
			// Reset color and frame on number change
			m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color(""))
			m.frame = 0

			oldPrice := u.ConvertFloatToString(m.config.Asset.BaseAsset.QuotePrice.Price, m.config.Asset.BaseAsset.Meta.IsVariablePrecision)
			newPrice := u.ConvertFloatToString(msg.BaseAsset.QuotePrice.Price, msg.BaseAsset.Meta.IsVariablePrecision)

			if msg.BaseAsset.QuotePrice.Price > m.config.Asset.BaseAsset.QuotePrice.Price {
				m.priceChangeDirection = 1
			} else if msg.BaseAsset.QuotePrice.Price < m.config.Asset.BaseAsset.QuotePrice.Price {
				m.priceChangeDirection = -1
			}

			// Find the last position where prices differ by iterating from right to left
			if len(oldPrice) == len(newPrice) {
				i := len(newPrice) - 1
				highestIndex := i
				for i >= 0 {
					if newPrice[i] != oldPrice[i] {
						highestIndex = i
					}
					i--
				}

				// Split the price into unchanged and changed segments
				m.priceNoChangeSegment = newPrice[:highestIndex]
				m.priceChangeSegment = newPrice[highestIndex:]
			} else {
				m.priceNoChangeSegment = ""
				m.priceChangeSegment = newPrice
			}

			m.config.Asset = msg // msg is *analysis.AnalyzedAsset

			return m, frameCmd(m.id)

		}

		// If symbol has changed or price has not changed then just update the asset
		m.config.Asset = msg // msg is *analysis.AnalyzedAsset
		m.priceNoChangeSegment = u.ConvertFloatToString(msg.BaseAsset.QuotePrice.Price, msg.BaseAsset.Meta.IsVariablePrecision) // Access via BaseAsset
		m.priceChangeSegment = ""

		return m, nil

	case FrameMsg:

		if m.id != int(msg) {
			return m, nil
		}

		if m.frame < 4 && m.priceChangeDirection > 0 {
			switch m.frame {
			case 0:
				m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("154")).Background(lipgloss.Color("22"))
			case 1:
				m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("154")).Background(lipgloss.Color("22"))
			case 2:
				m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("157")).Background(lipgloss.Color("232"))
			case 3:
				m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color(""))
			}

			m.frame++

			return m, frameCmd(m.id)
		}

		if m.frame < 4 && m.priceChangeDirection < 0 {
			switch m.frame {
			case 0:
				m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Background(lipgloss.Color("52"))
			case 1:
				m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Background(lipgloss.Color("52"))
			case 2:
				m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("210")).Background(lipgloss.Color("232"))
			case 3:
				m.priceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color(""))
			}

			m.frame++

			return m, frameCmd(m.id)
		}

		return m, nil
	}

	return m, nil
}

// View rendering hook
func (m *Model) View() string {

	rows := []grid.Row{}

	rows = append(
		rows,
		grid.Row{
			Width: m.width,
			Cells: m.buildCells(),
		})

	if m.config.ExtraInfoExchange {
		rows = append(
			rows,
			grid.Row{
				Width: m.width,
				Cells: []grid.Cell{
					// textTags expects *c.Asset, so we pass m.config.Asset.BaseAsset
					{Text: textTags(&m.config.Asset.BaseAsset, m.config.Styles)},
				},
			})
	}

	if m.config.Separate {
		rows = append(
			rows,
			grid.Row{
				Width: m.width,
				Cells: []grid.Cell{
					{Text: textSeparator(m.width, m.config.Styles)},
				},
			})
	}

	return grid.Render(grid.Grid{Rows: rows, GutterHorizontal: WidthGutter})
}

func (m *Model) buildCells() []grid.Cell { // m.config.Asset is *analysis.AnalyzedAsset

	// Access base asset via m.config.Asset.BaseAsset
	baseAsset := &m.config.Asset.BaseAsset
	analysisData := m.config.Asset.Analysis // Store analysis data

	// This slice will hold all cells that come *after* Name and MarketState
	var cells []grid.Cell

	// If not showing fundamentals AND not showing holdings, this is the most basic view
	if !m.config.ExtraInfoFundamentals && !m.config.ShowHoldings {
		cells = []grid.Cell{
			{Text: textQuote(baseAsset, m.config.Styles, m.priceStyle, m.priceNoChangeSegment, m.priceChangeSegment), Width: m.cellWidths.WidthQuote, Align: grid.Right},
		}
	} else {
		// Build up cells for fundamentals and/or holdings.
		// The original logic prepends, so we'll follow that pattern carefully.
		// Start with the quote cell, as it's the rightmost of the financial data.
		currentFinancialCells := []grid.Cell{
			{Text: textQuote(baseAsset, m.config.Styles, m.priceStyle, m.priceNoChangeSegment, m.priceChangeSegment), Width: m.cellWidths.WidthQuote, Align: grid.Right},
		}

		if m.config.ShowHoldings {
			holdingCells := []grid.Cell{
				{
					Text:  textPositionExtendedLabels(baseAsset, m.config.Styles),
					Width: WidthLabel,
					Align: grid.Right,
				},
				{
					Text:  textPositionExtended(baseAsset, m.config.Styles),
					Width: m.cellWidths.WidthPositionExtended,
					Align: grid.Right,
				},
				{
					Text:  textPosition(baseAsset, m.config.Styles),
					Width: m.cellWidths.WidthPosition,
					Align: grid.Right,
				},
			}
			currentFinancialCells = append(holdingCells, currentFinancialCells...)
		}

		if m.config.ExtraInfoFundamentals {
			fundamentalCells := []grid.Cell{
				{
					Text:  textVolumeMarketCapLabels(baseAsset, m.config.Styles),
					Width: WidthLabel,
					Align: grid.Right,
				},
				{
					Text:  textVolumeMarketCap(baseAsset),
					Width: m.cellWidths.WidthVolumeMarketCap,
					Align: grid.Right,
				},
				{
					Text:  textQuoteRangeLabels(baseAsset, m.config.Styles),
					Width: WidthLabel,
					Align: grid.Right,
				},
				{
					Text:  textQuoteRange(baseAsset, m.config.Styles),
					Width: m.cellWidths.WidthQuoteRange,
					Align: grid.Right,
				},
				{
					Text:  textQuoteExtendedLabels(baseAsset, m.config.Styles),
					Width: WidthLabel,
					Align: grid.Right,
				},
				{
					Text:  textQuoteExtended(baseAsset, m.config.Styles),
					Width: m.cellWidths.WidthQuoteExtended,
					Align: grid.Right,
				},
			}
			currentFinancialCells = append(fundamentalCells, currentFinancialCells...)
		}
		cells = currentFinancialCells
	}


	// Append Analysis Decision and Trend Status cells to the financial data
	analysisDecisionText := textAnalysisDecision(analysisData, m.config.Styles)
	if analysisDecisionText != "" {
		cells = append(cells, grid.Cell{Text: analysisDecisionText, Align: grid.Left})
	}

	analysisTrendStatusText := textAnalysisTrendStatus(analysisData, m.config.Styles)
	if analysisTrendStatusText != "" {
		cells = append(cells, grid.Cell{Text: analysisTrendStatusText, Align: grid.Left})
	}
	
	// Initial Name and Market State cells
	nameAndMarketStateCells := []grid.Cell{
		{Text: textName(baseAsset, m.config.Styles), Width: WidthName},
		{Text: ""}, // Gutter
		{Text: textMarketState(baseAsset, m.config.Styles), Width: WidthMarketState, Align: grid.Right},
	}

	// Prepend Name and Market State cells to the beginning of all other cells
	finalCells := append(nameAndMarketStateCells, cells...)

	return finalCells
}

// textName, textQuote, textPosition, etc. now receive *c.Asset
// These helper functions remain unchanged in their signature as they operate on *c.Asset.
// The caller (buildCells) is responsible for passing the correct part of AnalyzedAsset.

// Helper function for Analysis Decision
func textAnalysisDecision(analysisResult analysis.AnalysisResults, styles c.Styles) string {
	if analysisResult.Decision == "" {
		return "" // Or "N/A" if preferred: return styles.TextLabel("N/A")
	}
	strippedDecision := u.StripRichTags(analysisResult.Decision)
	return styles.Text(strippedDecision) // styles.Text should handle plain text fine
}

// Helper function for Analysis Trend Status
func textAnalysisTrendStatus(analysisResult analysis.AnalysisResults, styles c.Styles) string {
	if analysisResult.TrendStatus == "" {
		return "" // Or "N/A": return styles.TextLabel("N/A")
	}
	strippedTrendStatus := u.StripRichTags(analysisResult.TrendStatus)
	return styles.Text(strippedTrendStatus) // styles.Text should handle plain text fine
}

func textName(asset *c.Asset, styles c.Styles) string {
// The caller (buildCells) is responsible for passing the correct part of AnalyzedAsset.

func textName(asset *c.Asset, styles c.Styles) string {

	if len(asset.Name) > 20 {
		asset.Name = asset.Name[:20]
	}

	return styles.TextBold(asset.Symbol) +
		"\n" +
		styles.TextLabel(asset.Name)
}

func textQuote(asset *c.Asset, styles c.Styles, priceStyle lipgloss.Style, priceNoChangeSegment string, priceChangeSegment string) string {
	return priceNoChangeSegment + priceStyle.Render(priceChangeSegment) +
		"\n" +
		quoteChangeText(asset.QuotePrice.Change, asset.QuotePrice.ChangePercent, asset.Meta.IsVariablePrecision, styles)
}

func textPosition(asset *c.Asset, styles c.Styles) string {

	positionValue := ""
	positionChange := ""

	if asset.Holding.Value != 0.0 {
		positionValue = u.ValueText(asset.Holding.Value, styles) +
			styles.TextLight(
				" ("+
					u.ConvertFloatToString(asset.Holding.Weight, asset.Meta.IsVariablePrecision)+"%"+
					")")
	}
	if asset.Holding.TotalChange.Amount != 0.0 {
		positionChange = quoteChangeText(asset.Holding.TotalChange.Amount, asset.Holding.TotalChange.Percent, asset.Meta.IsVariablePrecision, styles)
	}

	return positionValue +
		"\n" +
		positionChange
}

func textQuoteExtended(asset *c.Asset, styles c.Styles) string {

	if asset.Class == c.AssetClassFuturesContract && asset.QuoteFutures.IndexPrice == 0.0 {
		return ""
	}

	if asset.Class == c.AssetClassFuturesContract {
		return styles.Text(u.ConvertFloatToString(asset.QuoteFutures.IndexPrice, asset.Meta.IsVariablePrecision)) +
			"\n" +
			styles.Text(u.ConvertFloatToString(asset.QuoteFutures.Basis, false)) + "%"
	}

	if asset.QuotePrice.PriceOpen == 0.0 {
		return styles.Text(u.ConvertFloatToString(asset.QuotePrice.PricePrevClose, asset.Meta.IsVariablePrecision)) +
			"\n"
	}

	return styles.Text(u.ConvertFloatToString(asset.QuotePrice.PricePrevClose, asset.Meta.IsVariablePrecision)) +
		"\n" +
		styles.Text(u.ConvertFloatToString(asset.QuotePrice.PriceOpen, asset.Meta.IsVariablePrecision))

}

func textQuoteExtendedLabels(asset *c.Asset, styles c.Styles) string {

	if asset.Class == c.AssetClassFuturesContract && asset.QuoteFutures.IndexPrice == 0.0 {
		return ""
	}

	if asset.Class == c.AssetClassFuturesContract {
		return styles.TextLabel("Index Price:") +
			"\n" +
			styles.TextLabel("Basis:")
	}

	if asset.QuotePrice.PriceOpen == 0.0 {
		return styles.TextLabel("Prev. Close:") +
			"\n"
	}

	return styles.TextLabel("Prev. Close:") +
		"\n" +
		styles.TextLabel("Open:")
}

func textPositionExtended(asset *c.Asset, styles c.Styles) string {

	if asset.Holding.Quantity == 0.0 {
		return ""
	}

	return styles.Text(u.ConvertFloatToString(asset.Holding.UnitCost, asset.Meta.IsVariablePrecision)) +
		"\n" +
		styles.Text(u.ConvertFloatToString(asset.Holding.Quantity, asset.Meta.IsVariablePrecision))

}

func textPositionExtendedLabels(asset *c.Asset, styles c.Styles) string {

	if asset.Holding.Quantity == 0.0 {
		return ""
	}

	return styles.TextLabel("Avg. Cost:") +
		"\n" +
		styles.TextLabel("Quantity:")
}

func textQuoteRange(asset *c.Asset, styles c.Styles) string {

	if asset.Class == c.AssetClassFuturesContract {

		if asset.QuotePrice.PriceDayHigh != 0.0 && asset.QuotePrice.PriceDayLow != 0.0 {
			return u.ConvertFloatToString(asset.QuotePrice.PriceDayLow, asset.Meta.IsVariablePrecision) +
				styles.Text(" - ") +
				u.ConvertFloatToString(asset.QuotePrice.PriceDayHigh, asset.Meta.IsVariablePrecision) +
				"\n" +
				asset.QuoteFutures.Expiry
		}

		return asset.QuoteFutures.Expiry

	}

	if asset.QuotePrice.PriceDayHigh != 0.0 && asset.QuotePrice.PriceDayLow != 0.0 {
		return u.ConvertFloatToString(asset.QuotePrice.PriceDayLow, asset.Meta.IsVariablePrecision) +
			styles.Text(" - ") +
			u.ConvertFloatToString(asset.QuotePrice.PriceDayHigh, asset.Meta.IsVariablePrecision) +
			"\n" +
			u.ConvertFloatToString(asset.QuoteExtended.FiftyTwoWeekLow, asset.Meta.IsVariablePrecision) +
			styles.Text(" - ") +
			u.ConvertFloatToString(asset.QuoteExtended.FiftyTwoWeekHigh, asset.Meta.IsVariablePrecision)
	}

	return ""

}

func textQuoteRangeLabels(asset *c.Asset, styles c.Styles) string {

	if asset.Class == c.AssetClassFuturesContract {

		if asset.QuotePrice.PriceDayHigh != 0.0 && asset.QuotePrice.PriceDayLow != 0.0 {
			return styles.TextLabel("Day Range:") +
				"\n" +
				styles.TextLabel("Expiry:")
		}

		return styles.TextLabel("Expiry:")
	}

	if asset.QuotePrice.PriceDayHigh != 0.0 && asset.QuotePrice.PriceDayLow != 0.0 {
		return styles.TextLabel("Day Range:") +
			"\n" +
			styles.TextLabel("52wk Range:")
	}

	return ""
}

func textVolumeMarketCap(asset *c.Asset) string { // No styles needed here as per original

	if asset.Class == c.AssetClassFuturesContract {
		return u.ConvertFloatToString(asset.QuoteFutures.OpenInterest, true) +
			"\n" +
			u.ConvertFloatToString(asset.QuoteExtended.Volume, true)
	}

	return u.ConvertFloatToString(asset.QuoteExtended.MarketCap, true) +
		"\n" +
		u.ConvertFloatToString(asset.QuoteExtended.Volume, true)
}
func textVolumeMarketCapLabels(asset *c.Asset, styles c.Styles) string {

	if asset.Class == c.AssetClassFuturesContract {
		return styles.TextLabel("Open Interest:") +
			"\n" +
			styles.TextLabel("Volume:")
	}

	return styles.TextLabel("Market Cap:") +
		"\n" +
		styles.TextLabel("Volume:")
}

func textMarketState(asset *c.Asset, styles c.Styles) string {
	if asset.Exchange.IsRegularTradingSession {
		return styles.TextLabel(" ●  ")
	}

	if !asset.Exchange.IsRegularTradingSession && asset.Exchange.IsActive {
		return styles.TextLabel(" ○  ")
	}

	return ""
}

func quoteChangeText(change float64, changePercent float64, isVariablePrecision bool, styles c.Styles) string {
	if change == 0.0 {
		return styles.TextPrice(changePercent, "  "+u.ConvertFloatToString(change, isVariablePrecision)+" ("+u.ConvertFloatToString(changePercent, false)+"%)")
	}

	if change > 0.0 {
		return styles.TextPrice(changePercent, "↑ "+u.ConvertFloatToString(change, isVariablePrecision)+" ("+u.ConvertFloatToString(changePercent, false)+"%)")
	}

	return styles.TextPrice(changePercent, "↓ "+u.ConvertFloatToString(change, isVariablePrecision)+" ("+u.ConvertFloatToString(changePercent, false)+"%)")
}

func textSeparator(width int, styles c.Styles) string {
	return styles.TextLine(strings.Repeat("─", width))
}

func textTags(asset *c.Asset, styles c.Styles) string {

	currencyText := asset.Currency.FromCurrencyCode

	if asset.Currency.ToCurrencyCode != "" && asset.Currency.ToCurrencyCode != asset.Currency.FromCurrencyCode {
		currencyText = asset.Currency.FromCurrencyCode + " → " + asset.Currency.ToCurrencyCode
	}

	return formatTag(currencyText, styles) + " " + formatTag(exchangeDelayText(asset.Exchange.Delay, asset.Exchange.DelayText), styles) + " " + formatTag(asset.Exchange.Name, styles)
}

func exchangeDelayText(delay float64, delayText string) string {

	if delayText != "" {
		return delayText
	}

	if delay <= 0 {
		return "Live"
	}

	return "Delayed " + strconv.FormatFloat(delay, 'f', 0, 64) + "min"
}

func formatTag(text string, style c.Styles) string {
	return style.Tag(" " + text + " ")
}

func frameCmd(id int) tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return FrameMsg(id)
	})
}

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}
