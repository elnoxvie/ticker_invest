package indicator

import (
	"reflect"
	"testing"
	// "time" // Not needed as Date doesn't affect SMA
)

// floatPtr helper to create a *float64 for expected values
func floatPtr(v float64) *float64 {
	return &v
}

func TestCalculateSMA(t *testing.T) {
	tests := []struct {
		name     string
		data     []HistoricalDataPoint
		period   int
		expected []*float64
	}{
		{
			name: "Simple SMA 3 period",
			data: []HistoricalDataPoint{
				{Close: 1.0}, {Close: 2.0}, {Close: 3.0}, {Close: 4.0}, {Close: 5.0},
			},
			period: 3,
			expected: []*float64{
				nil, nil, floatPtr(2.0), floatPtr(3.0), floatPtr(4.0),
			},
		},
		{
			name: "Period equals data length",
			data: []HistoricalDataPoint{
				{Close: 1.0}, {Close: 2.0}, {Close: 3.0},
			},
			period: 3,
			expected: []*float64{
				nil, nil, floatPtr(2.0),
			},
		},
		{
			name: "Period longer than data length",
			data: []HistoricalDataPoint{
				{Close: 1.0}, {Close: 2.0},
			},
			period:   3,
			expected: []*float64{nil, nil},
		},
		{
			name:     "Empty data",
			data:     []HistoricalDataPoint{},
			period:   3,
			expected: []*float64{},
		},
		{
			name: "Zero period",
			data: []HistoricalDataPoint{
				{Close: 1.0}, {Close: 2.0}, {Close: 3.0},
			},
			period:   0,
			expected: []*float64{nil, nil, nil},
		},
		{
			name: "Negative period",
			data: []HistoricalDataPoint{
				{Close: 1.0}, {Close: 2.0}, {Close: 3.0},
			},
			period:   -1,
			expected: []*float64{nil, nil, nil},
		},
		{
			name: "SMA 1 period",
			data: []HistoricalDataPoint{
				{Close: 1.0}, {Close: 2.0}, {Close: 3.0}, {Close: 10.5},
			},
			period: 1,
			expected: []*float64{
				floatPtr(1.0), floatPtr(2.0), floatPtr(3.0), floatPtr(10.5),
			},
		},
		{
			name: "All same values",
			data: []HistoricalDataPoint{
				{Close: 5.0}, {Close: 5.0}, {Close: 5.0}, {Close: 5.0},
			},
			period: 2,
			expected: []*float64{
				nil, floatPtr(5.0), floatPtr(5.0), floatPtr(5.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSMA(tt.data, tt.period)
			if !reflect.DeepEqual(result, tt.expected) {
				// For better error messages, especially with many nil pointers,
				// we might want a custom comparison or formatting function.
				// But for now, default output should be okay.
				t.Errorf("CalculateSMA() got = %v, want %v", result, tt.expected)
			}
		})
	}
}
