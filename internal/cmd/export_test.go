package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExportSuite struct {
	suite.Suite
}

func TestExportSuite(t *testing.T) {
	suite.Run(t, new(ExportSuite))
}

func (s *ExportSuite) TestGetString_ValidValue() {
	m := map[string]any{
		"key": "value",
	}
	result := getString(m, "key")
	require.Equal(s.T(), "value", result)
}

func (s *ExportSuite) TestGetString_NilValue() {
	m := map[string]any{
		"key": nil,
	}
	result := getString(m, "key")
	require.Equal(s.T(), "", result)
}

func (s *ExportSuite) TestGetString_MissingKey() {
	m := map[string]any{}
	result := getString(m, "nonexistent")
	require.Equal(s.T(), "", result)
}

func (s *ExportSuite) TestGetString_NumericValue() {
	m := map[string]any{
		"count": 42,
	}
	result := getString(m, "count")
	require.Equal(s.T(), "42", result)
}

func (s *ExportSuite) TestGetString_BoolValue() {
	m := map[string]any{
		"active": true,
	}
	result := getString(m, "active")
	require.Equal(s.T(), "true", result)
}

func (s *ExportSuite) TestJoinStrings_ValidSlice() {
	m := map[string]any{
		"tags": []any{"bug", "urgent", "backend"},
	}
	result := joinStrings(m, "tags")
	require.Equal(s.T(), "bug;urgent;backend", result)
}

func (s *ExportSuite) TestJoinStrings_NilValue() {
	m := map[string]any{
		"tags": nil,
	}
	result := joinStrings(m, "tags")
	require.Equal(s.T(), "", result)
}

func (s *ExportSuite) TestJoinStrings_MissingKey() {
	m := map[string]any{}
	result := joinStrings(m, "nonexistent")
	require.Equal(s.T(), "", result)
}

func (s *ExportSuite) TestJoinStrings_EmptySlice() {
	m := map[string]any{
		"tags": []any{},
	}
	result := joinStrings(m, "tags")
	require.Equal(s.T(), "", result)
}

func (s *ExportSuite) TestJoinStrings_SingleItem() {
	m := map[string]any{
		"tags": []any{"single"},
	}
	result := joinStrings(m, "tags")
	require.Equal(s.T(), "single", result)
}

func (s *ExportSuite) TestJoinStrings_MixedTypes() {
	m := map[string]any{
		"items": []any{"text", 123, true},
	}
	result := joinStrings(m, "items")
	require.Equal(s.T(), "text;123;true", result)
}

func (s *ExportSuite) TestJoinStrings_NotASlice() {
	m := map[string]any{
		"notslice": "just a string",
	}
	result := joinStrings(m, "notslice")
	require.Equal(s.T(), "", result)
}

func (s *ExportSuite) TestExportJSON_EmptySlice() {
	var buf bytes.Buffer
	err := exportJSON(&buf, []any{})
	require.NoError(s.T(), err)
	require.Equal(s.T(), "[]\n", buf.String())
}

func (s *ExportSuite) TestExportJSON_SimpleData() {
	var buf bytes.Buffer
	data := map[string]string{"key": "value"}
	err := exportJSON(&buf, data)
	require.NoError(s.T(), err)
	require.Contains(s.T(), buf.String(), `"key": "value"`)
}

func (s *ExportSuite) TestExportCSV_EmptySlice() {
	var buf bytes.Buffer
	err := exportCSV(&buf, []any{})
	require.NoError(s.T(), err)
	// Should contain just the header
	require.Contains(s.T(), buf.String(), "ID,Status,Type")
}
