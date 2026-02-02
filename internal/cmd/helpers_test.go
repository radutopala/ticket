package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HelpersSuite struct {
	suite.Suite
}

func TestHelpersSuite(t *testing.T) {
	suite.Run(t, new(HelpersSuite))
}

func (s *HelpersSuite) TestRemoveFromSlice() {
	tests := []struct {
		name      string
		slice     []string
		value     string
		wantSlice []string
		wantFound bool
	}{
		{
			name:      "value found at beginning",
			slice:     []string{"a", "b", "c"},
			value:     "a",
			wantSlice: []string{"b", "c"},
			wantFound: true,
		},
		{
			name:      "value found in middle",
			slice:     []string{"a", "b", "c"},
			value:     "b",
			wantSlice: []string{"a", "c"},
			wantFound: true,
		},
		{
			name:      "value found at end",
			slice:     []string{"a", "b", "c"},
			value:     "c",
			wantSlice: []string{"a", "b"},
			wantFound: true,
		},
		{
			name:      "value not found",
			slice:     []string{"a", "b", "c"},
			value:     "d",
			wantSlice: []string{"a", "b", "c"},
			wantFound: false,
		},
		{
			name:      "empty slice",
			slice:     []string{},
			value:     "a",
			wantSlice: []string{},
			wantFound: false,
		},
		{
			name:      "single element found",
			slice:     []string{"a"},
			value:     "a",
			wantSlice: []string{},
			wantFound: true,
		},
		{
			name:      "single element not found",
			slice:     []string{"a"},
			value:     "b",
			wantSlice: []string{"a"},
			wantFound: false,
		},
		{
			name:      "multiple occurrences removes all",
			slice:     []string{"a", "b", "a", "c", "a"},
			value:     "a",
			wantSlice: []string{"b", "c"},
			wantFound: true,
		},
		{
			name:      "nil slice",
			slice:     nil,
			value:     "a",
			wantSlice: []string{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			gotSlice, gotFound := removeFromSlice(tt.slice, tt.value)
			require.Equal(s.T(), tt.wantFound, gotFound)
			require.Equal(s.T(), tt.wantSlice, gotSlice)
		})
	}
}
