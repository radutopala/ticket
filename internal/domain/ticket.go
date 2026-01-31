// Package domain contains core domain models.
package domain

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Status represents the ticket status.
type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusClosed     Status = "closed"
)

// String returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}

// IsValid checks if the status is valid.
func (s Status) IsValid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusClosed:
		return true
	default:
		return false
	}
}

// ParseStatus parses a string into a Status.
func ParseStatus(s string) (Status, error) {
	switch s {
	case "open":
		return StatusOpen, nil
	case "in_progress":
		return StatusInProgress, nil
	case "closed":
		return StatusClosed, nil
	default:
		return "", fmt.Errorf("invalid status: %s", s)
	}
}

// Type represents the ticket type.
type Type string

const (
	TypeTask    Type = "task"
	TypeBug     Type = "bug"
	TypeFeature Type = "feature"
	TypeEpic    Type = "epic"
	TypeChore   Type = "chore"
)

// String returns the string representation of the type.
func (t Type) String() string {
	return string(t)
}

// IsValid checks if the type is valid.
func (t Type) IsValid() bool {
	switch t {
	case TypeTask, TypeBug, TypeFeature, TypeEpic, TypeChore:
		return true
	default:
		return false
	}
}

// ParseType parses a string into a Type.
func ParseType(s string) (Type, error) {
	switch s {
	case "task":
		return TypeTask, nil
	case "bug":
		return TypeBug, nil
	case "feature":
		return TypeFeature, nil
	case "epic":
		return TypeEpic, nil
	case "chore":
		return TypeChore, nil
	default:
		return "", fmt.Errorf("invalid type: %s", s)
	}
}

// Note represents a timestamped note on a ticket.
type Note struct {
	Timestamp time.Time
	Content   string
}

// Ticket represents a ticket in the system.
type Ticket struct {
	// Frontmatter fields
	ID          string    `yaml:"id"`
	Status      Status    `yaml:"status"`
	Type        Type      `yaml:"type,omitempty"`
	Priority    int       `yaml:"priority,omitempty"`
	Assignee    string    `yaml:"assignee,omitempty"`
	Parent      string    `yaml:"parent,omitempty"`
	ExternalRef string    `yaml:"external-ref,omitempty"`
	Tags        []string  `yaml:"tags,omitempty"`
	Deps        []string  `yaml:"deps,omitempty"`
	Links       []string  `yaml:"links,omitempty"`
	Created     time.Time `yaml:"created"`

	// Body fields (not in frontmatter)
	Title       string `yaml:"-"`
	Description string `yaml:"-"`
	Design      string `yaml:"-"`
	Acceptance  string `yaml:"-"`
	Notes       []Note `yaml:"-"`
}

// ParseFromFile reads and parses a ticket from a file.
func ParseFromFile(path string) (*Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read ticket file: %w", err)
	}

	return Parse(data)
}

// Parse parses a ticket from markdown content.
func Parse(data []byte) (*Ticket, error) {
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	var ticket Ticket
	if err := yaml.Unmarshal(frontmatter, &ticket); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	ticket.ParseMarkdownBody(string(body))

	return &ticket, nil
}

// WriteToFile writes the ticket to a file.
func (t *Ticket) WriteToFile(path string) error {
	data, err := t.Render()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write ticket file: %w", err)
	}

	return nil
}

// Render renders the ticket as markdown content.
func (t *Ticket) Render() ([]byte, error) {
	var buf bytes.Buffer

	// Write frontmatter
	buf.WriteString("---\n")

	frontmatter, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}
	buf.Write(frontmatter)
	buf.WriteString("---\n")

	// Write body
	buf.WriteString(t.RenderMarkdownBody())

	return buf.Bytes(), nil
}

// ParseMarkdownBody parses the markdown body and populates body fields.
func (t *Ticket) ParseMarkdownBody(content string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentSection string
	var sectionContent strings.Builder

	flushSection := func() {
		text := strings.TrimSpace(sectionContent.String())
		switch currentSection {
		case "title":
			t.Title = text
		case "description":
			t.Description = text
		case "design":
			t.Design = text
		case "acceptance":
			t.Acceptance = text
		case "notes":
			t.Notes = parseNotes(text)
		}
		sectionContent.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Check for section headers
		if strings.HasPrefix(line, "# ") {
			flushSection()
			t.Title = strings.TrimPrefix(line, "# ")
			currentSection = "title"
			continue
		}

		if strings.HasPrefix(line, "## ") {
			flushSection()
			header := strings.TrimPrefix(line, "## ")
			switch strings.ToLower(header) {
			case "design":
				currentSection = "design"
			case "acceptance criteria":
				currentSection = "acceptance"
			case "notes":
				currentSection = "notes"
			default:
				currentSection = "description"
				sectionContent.WriteString(line)
				sectionContent.WriteString("\n")
			}
			continue
		}

		// After title, content is description until we hit a known section
		if currentSection == "title" && line != "" {
			currentSection = "description"
		}

		sectionContent.WriteString(line)
		sectionContent.WriteString("\n")
	}

	flushSection()
}

// RenderMarkdownBody renders the body fields as markdown.
func (t *Ticket) RenderMarkdownBody() string {
	var buf strings.Builder

	// Title
	if t.Title != "" {
		buf.WriteString("# ")
		buf.WriteString(t.Title)
		buf.WriteString("\n\n")
	}

	// Description
	if t.Description != "" {
		buf.WriteString(t.Description)
		buf.WriteString("\n\n")
	}

	// Design
	if t.Design != "" {
		buf.WriteString("## Design\n\n")
		buf.WriteString(t.Design)
		buf.WriteString("\n\n")
	}

	// Acceptance Criteria
	if t.Acceptance != "" {
		buf.WriteString("## Acceptance Criteria\n\n")
		buf.WriteString(t.Acceptance)
		buf.WriteString("\n\n")
	}

	// Notes
	if len(t.Notes) > 0 {
		buf.WriteString("## Notes\n\n")
		for _, note := range t.Notes {
			buf.WriteString(fmt.Sprintf("### %s\n\n", note.Timestamp.Format(time.RFC3339)))
			buf.WriteString(note.Content)
			buf.WriteString("\n\n")
		}
	}

	return buf.String()
}

// splitFrontmatter splits the content into frontmatter and body.
func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return nil, nil, fmt.Errorf("missing frontmatter delimiter")
	}

	content = content[4:] // Skip first "---\n"

	idx := strings.Index(content, "\n---\n")
	if idx == -1 {
		return nil, nil, fmt.Errorf("missing closing frontmatter delimiter")
	}

	frontmatter := content[:idx]
	body := content[idx+5:] // Skip "\n---\n"

	return []byte(frontmatter), []byte(body), nil
}

// parseNotes parses notes from the notes section content.
func parseNotes(content string) []Note {
	var notes []Note
	var currentNote *Note
	var noteContent strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "### ") {
			// Flush previous note
			if currentNote != nil {
				currentNote.Content = strings.TrimSpace(noteContent.String())
				notes = append(notes, *currentNote)
				noteContent.Reset()
			}

			timestamp := strings.TrimPrefix(line, "### ")
			t, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				continue
			}
			currentNote = &Note{Timestamp: t}
			continue
		}

		if currentNote != nil {
			noteContent.WriteString(line)
			noteContent.WriteString("\n")
		}
	}

	// Flush last note
	if currentNote != nil {
		currentNote.Content = strings.TrimSpace(noteContent.String())
		notes = append(notes, *currentNote)
	}

	return notes
}
