package ticket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SearchSuite struct {
	suite.Suite
}

func TestSearchSuite(t *testing.T) {
	suite.Run(t, new(SearchSuite))
}

func (s *SearchSuite) TestSearch_TitleMatch() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Title: "Fix authentication bug", Created: time.Now()},
		{ID: "t2", Status: StatusOpen, Title: "Add logging", Created: time.Now()},
	}

	matches := Search(tickets, "authentication", false, "")
	require.Len(s.T(), matches, 1)
	require.Equal(s.T(), "t1", matches[0].Ticket.ID)
	require.Empty(s.T(), matches[0].Context)
}

func (s *SearchSuite) TestSearch_DescriptionMatch() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Title: "Simple title", Description: "Contains the search keyword here", Created: time.Now()},
	}

	matches := Search(tickets, "keyword", false, "")
	require.Len(s.T(), matches, 1)
	require.Equal(s.T(), "t1", matches[0].Ticket.ID)
	require.NotEmpty(s.T(), matches[0].Context)
	require.Contains(s.T(), matches[0].Context, "keyword")
}

func (s *SearchSuite) TestSearch_CaseInsensitive() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Title: "Fix Authentication Bug", Created: time.Now()},
	}

	matches := Search(tickets, "authentication", false, "")
	require.Len(s.T(), matches, 1)
}

func (s *SearchSuite) TestSearch_CaseSensitive() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Title: "Fix Authentication Bug", Created: time.Now()},
	}

	matches := Search(tickets, "authentication", true, "")
	require.Empty(s.T(), matches)

	matches = Search(tickets, "Authentication", true, "")
	require.Len(s.T(), matches, 1)
}

func (s *SearchSuite) TestSearch_StatusFilter() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Title: "auth open", Created: time.Now()},
		{ID: "t2", Status: StatusClosed, Title: "auth closed", Created: time.Now()},
	}

	matches := Search(tickets, "auth", false, "open")
	require.Len(s.T(), matches, 1)
	require.Equal(s.T(), "t1", matches[0].Ticket.ID)
}

func (s *SearchSuite) TestSearch_NoMatch() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Title: "Something", Description: "Else", Created: time.Now()},
	}

	matches := Search(tickets, "nonexistent", false, "")
	require.Empty(s.T(), matches)
}

func (s *SearchSuite) TestSearch_SortedByPriority() {
	tickets := []*Ticket{
		{ID: "t3", Status: StatusOpen, Title: "auth", Priority: 3, Created: time.Now()},
		{ID: "t1", Status: StatusOpen, Title: "auth", Priority: 1, Created: time.Now()},
		{ID: "t2", Status: StatusOpen, Title: "auth", Priority: 2, Created: time.Now()},
	}

	matches := Search(tickets, "auth", false, "")
	require.Len(s.T(), matches, 3)
	require.Equal(s.T(), "t1", matches[0].Ticket.ID)
	require.Equal(s.T(), "t2", matches[1].Ticket.ID)
	require.Equal(s.T(), "t3", matches[2].Ticket.ID)
}

func (s *SearchSuite) TestSearch_TitleMatchSkipsDescription() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Title: "auth fix", Description: "also has auth in description", Created: time.Now()},
	}

	matches := Search(tickets, "auth", false, "")
	require.Len(s.T(), matches, 1)
	require.Empty(s.T(), matches[0].Context, "title match should not extract description context")
}

func (s *SearchSuite) TestExtractContext_Middle() {
	text := "This is a longer description that contains the search term somewhere in the middle"
	ctx := ExtractContext(text, 46, 11, 15)
	require.Contains(s.T(), ctx, "search term")
}

func (s *SearchSuite) TestExtractContext_Start() {
	text := "Search at the very beginning of text"
	ctx := ExtractContext(text, 0, 6, 10)
	require.Contains(s.T(), ctx, "Search")
}

func (s *SearchSuite) TestExtractContext_End() {
	text := "Some text at the very end"
	ctx := ExtractContext(text, 22, 3, 10)
	require.Contains(s.T(), ctx, "end")
}

func (s *SearchSuite) TestExtractContext_NewlineReplaced() {
	text := "Line one\nLine two\nLine three"
	ctx := ExtractContext(text, 5, 3, 20)
	require.NotContains(s.T(), ctx, "\n")
}
