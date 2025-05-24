package util

import (
	"math"
	"regexp" // Added for StripRichTags
	"strconv"

	c "github.com/achannarasappa/ticker/v4/internal/common"
)

// StripRichTags removes Rich-like tags (e.g., [bold green], [/bold green]) from a string.
// This regex is simplified and might not cover all complex Rich tag syntaxes,
// but should handle common cases like [bold green] or [red].
// It looks for an opening bracket, optionally a slash, any characters that are not ']', and then a closing bracket.
func StripRichTags(input string) string {
	re := regexp.MustCompile(`\[/?([a-zA-Z0-9_=\s-]+)\]`)
	return re.ReplaceAllString(input, "")
}

func getPrecision(f float64) int {

	v := math.Abs(f)

	if v == 0.0 {
		return 2
	}

	if v >= 1000000 {
		return 0
	}

	if v < 10 {
		return 4
	}

	if v < 100 {
		return 3
	}

	if v >= 1000 && f < 0 {
		return 1
	}

	return 2
}

// ConvertFloatToString formats a float as a string including handling large or small numbers
func ConvertFloatToString(f float64, isVariablePrecision bool) string {

	var unit string

	if !isVariablePrecision {
		return strconv.FormatFloat(f, 'f', 2, 64)
	}

	if f > 1000000000000 {
		f /= 1000000000000
		unit = " T"
	}

	if f > 1000000000 {
		f /= 1000000000
		unit = " B"
	}

	if f > 1000000 {
		f /= 1000000
		unit = " M"
	}

	prec := getPrecision(f)

	return strconv.FormatFloat(f, 'f', prec, 64) + unit
}

// ValueText formats a float as a styled string
func ValueText(value float64, styles c.Styles) string {
	if value <= 0.0 {
		return ""
	}

	return styles.Text(ConvertFloatToString(value, false))
}
