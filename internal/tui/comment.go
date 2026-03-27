package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
)

// CommentResult is returned by RunCommentFlow on success.
type CommentResult struct {
	Posted    github.PostedComment
	AnchorTag string // empty if no anchor was tagged
	Path      string
	Line      int
	Side      string
	Body      string
}

type commentStep int

const (
	stepFile commentStep = iota
	stepLine
	stepBody
	stepAnchor
	stepConfirm
)

type fileItem struct{ path string }

func (f fileItem) Title() string       { return f.path }
func (f fileItem) Description() string { return "" }
func (f fileItem) FilterValue() string { return f.path }

type postDoneMsg struct {
	result github.PostedComment
	err    error
}

// CommentModel is the bubbletea model for the bv comment wizard.
type CommentModel struct {
	pr           model.PR
	ref          model.PRRef
	step         commentStep
	fileList     list.Model
	lineInput    textinput.Model
	bodyInput    textarea.Model
	anchorInput  textinput.Model
	spinner      spinner.Model
	posting      bool
	result       *CommentResult
	err          error
	lineErr      string
	quitting     bool

	selectedFile string
	selectedLine int
	selectedSide string // always "RIGHT" for now
}

func NewCommentModel(pr model.PR, ref model.PRRef, files []string) CommentModel {
	// File list
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = fileItem{path: f}
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#22c55e")).
		BorderLeftForeground(lipgloss.Color("#22c55e"))
	l := list.New(items, delegate, 80, 18)
	l.Title = "Select a file to comment on"
	l.Styles.Title = lipgloss.NewStyle().Bold(true)
	l.SetFilteringEnabled(true)

	// Line number input
	lineInput := textinput.New()
	lineInput.Placeholder = "line number"
	lineInput.CharLimit = 6
	lineInput.Width = 20

	// Body textarea
	bodyInput := textarea.New()
	bodyInput.Placeholder = "Write your comment here..."
	bodyInput.SetWidth(70)
	bodyInput.SetHeight(8)
	bodyInput.ShowLineNumbers = false

	// Anchor tag input
	anchorInput := textinput.New()
	anchorInput.Placeholder = "perf  (or leave blank)"
	anchorInput.CharLimit = 30
	anchorInput.Width = 30

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))

	return CommentModel{
		pr:           pr,
		ref:          ref,
		step:         stepFile,
		fileList:     l,
		lineInput:    lineInput,
		bodyInput:    bodyInput,
		anchorInput:  anchorInput,
		spinner:      sp,
		selectedSide: "RIGHT",
	}
}

func (m CommentModel) Init() tea.Cmd {
	return nil
}

func (m CommentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.posting {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.step == stepFile {
				m.quitting = true
				return m, tea.Quit
			}
			m.step--
			return m, nil
		}

		switch m.step {
		case stepFile:
			if msg.String() == "enter" {
				selected, ok := m.fileList.SelectedItem().(fileItem)
				if ok {
					m.selectedFile = selected.path
					m.step = stepLine
					m.lineInput.Focus()
					return m, textinput.Blink
				}
			}
		case stepLine:
			if msg.String() == "enter" {
				line, err := parseLineNumber(m.lineInput.Value())
				if err != nil || line < 1 {
					m.lineErr = "invalid line number"
					return m, nil
				}
				m.lineErr = ""
				m.selectedLine = line
				m.step = stepBody
				m.bodyInput.Focus()
				return m, textarea.Blink
			}
		case stepBody:
			if msg.String() == "ctrl+d" {
				body := strings.TrimSpace(m.bodyInput.Value())
				if body == "" {
					return m, nil
				}
				m.step = stepAnchor
				m.anchorInput.Focus()
				return m, textinput.Blink
			}
		case stepAnchor:
			if msg.String() == "enter" || msg.String() == "tab" {
				m.step = stepConfirm
				return m, nil
			}
		case stepConfirm:
			if msg.String() == "enter" {
				m.posting = true
				pr := m.pr
				path := m.selectedFile
				line := m.selectedLine
				side := m.selectedSide
				body := strings.TrimSpace(m.bodyInput.Value())
				ref := m.ref
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						posted, err := github.PostReviewComment(
							github.GetClient(), context.Background(), ref, pr.HeadSHA, path, body, side, line,
						)
						return postDoneMsg{result: posted, err: err}
					},
				)
			}
		}

	case tea.WindowSizeMsg:
		m.fileList.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case postDoneMsg:
		m.posting = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		anchorTag := strings.TrimPrefix(strings.TrimSpace(m.anchorInput.Value()), "#")
		m.result = &CommentResult{
			Posted:    msg.result,
			AnchorTag: anchorTag,
			Path:      m.selectedFile,
			Line:      m.selectedLine,
			Side:      m.selectedSide,
			Body:      strings.TrimSpace(m.bodyInput.Value()),
		}
		return m, tea.Quit

	case spinner.TickMsg:
		if m.posting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Delegate to active component
	var cmd tea.Cmd
	switch m.step {
	case stepFile:
		m.fileList, cmd = m.fileList.Update(msg)
	case stepLine:
		m.lineInput, cmd = m.lineInput.Update(msg)
	case stepBody:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	case stepAnchor:
		m.anchorInput, cmd = m.anchorInput.Update(msg)
	}
	return m, cmd
}

func (m CommentModel) View() string {
	if m.posting {
		return fmt.Sprintf("\n  %s  Posting comment...\n", m.spinner.View())
	}

	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Faint(true)
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#38bdf8"))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))

	var sb strings.Builder

	// Breadcrumb
	sb.WriteString("\n")
	if m.selectedFile != "" {
		sb.WriteString("  " + dim.Render("file:") + " " + bold.Render(m.selectedFile) + "\n")
	}
	if m.selectedLine > 0 {
		sb.WriteString("  " + dim.Render("line:") + " " + bold.Render(fmt.Sprintf("%d", m.selectedLine)) + "\n")
	}
	sb.WriteString("\n")

	switch m.step {
	case stepFile:
		sb.WriteString(m.fileList.View())

	case stepLine:
		sb.WriteString("  " + prompt.Render("Line number:") + "  " + m.lineInput.View() + "\n")
		if m.lineErr != "" {
			sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(m.lineErr))
		}
		sb.WriteString("\n  " + dim.Render("press enter to continue, esc to go back"))

	case stepBody:
		sb.WriteString("  " + prompt.Render("Comment:") + "\n\n")
		sb.WriteString(m.bodyInput.View())
		sb.WriteString("\n\n  " + dim.Render("press ctrl+d to continue, esc to go back"))

	case stepAnchor:
		sb.WriteString("  " + prompt.Render("Anchor tag") + " " + dim.Render("(optional, e.g. perf):") + "  " + m.anchorInput.View() + "\n")
		sb.WriteString("\n  " + dim.Render("press enter to continue, esc to go back"))

	case stepConfirm:
		sb.WriteString("  " + bold.Render("Ready to post:") + "\n\n")
		sb.WriteString("  " + dim.Render("file:") + "  " + m.selectedFile + "\n")
		sb.WriteString("  " + dim.Render("line:") + "  " + fmt.Sprintf("%d", m.selectedLine) + "\n")
		body := strings.TrimSpace(m.bodyInput.Value())
		sb.WriteString("  " + dim.Render("body:") + "\n")
		for _, line := range strings.Split(body, "\n") {
			sb.WriteString("    " + line + "\n")
		}
		if tag := strings.TrimPrefix(strings.TrimSpace(m.anchorInput.Value()), "#"); tag != "" {
			sb.WriteString("  " + dim.Render("anchor:") + "  " + green.Render("#"+tag) + "\n")
		} else {
			sb.WriteString("  " + dim.Render("anchor:") + "  " + dim.Render("(none)") + "\n")
		}
		sb.WriteString("\n  " + prompt.Render("press enter to post, esc to go back"))

	}

	if m.err != nil {
		sb.WriteString("\n\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render("Error: "+m.err.Error()))
	}

	return sb.String()
}

// Result returns the final CommentResult, or nil if cancelled/errored.
func (m CommentModel) Result() *CommentResult { return m.result }

// RunCommentFlow launches the interactive comment wizard.
func RunCommentFlow(pr model.PR, ref model.PRRef, files []string) (*CommentResult, error) {
	m := NewCommentModel(pr, ref, files)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	return final.(CommentModel).Result(), nil
}

// helpers

func parseLineNumber(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
