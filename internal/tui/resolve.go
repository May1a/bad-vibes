package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/may/bad-vibes/internal/github"
	"github.com/may/bad-vibes/internal/model"
)

// --- list item ---

type threadItem struct {
	thread model.ReviewThread
}

func (i threadItem) Title() string {
	if i.thread.Path != "" && i.thread.Line > 0 {
		return fmt.Sprintf("%s:%d", i.thread.Path, i.thread.Line)
	}
	if i.thread.Path != "" {
		return i.thread.Path
	}
	return "PR-level comment"
}

func (i threadItem) Description() string {
	if len(i.thread.Comments) == 0 {
		return ""
	}
	c := i.thread.Comments[0]
	body := strings.ReplaceAll(c.Body, "\n", " ")
	if len(body) > 80 {
		body = body[:77] + "..."
	}
	return fmt.Sprintf("@%s — %s", c.Author, body)
}

func (i threadItem) FilterValue() string {
	return i.Title() + " " + i.Description()
}

// ResolvedThread holds the ID and display title of a resolved thread.
type ResolvedThread struct {
	ID    string
	Title string
}

// --- messages ---

type resolvedMsg struct {
	id    string
	title string
}
type resolveErrMsg struct{ err error }

// --- model ---

type ResolveModel struct {
	list      list.Model
	spinner   spinner.Model
	resolving bool
	resolved  []ResolvedThread
	errMsg    string
	quitting  bool
}

func NewResolveModel(threads []model.ReviewThread) ResolveModel {
	items := make([]list.Item, len(threads))
	for i, t := range threads {
		items[i] = threadItem{thread: t}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#facc15")).
		BorderLeftForeground(lipgloss.Color("#facc15"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#a16207")).
		BorderLeftForeground(lipgloss.Color("#facc15"))

	l := list.New(items, delegate, 80, 20)
	l.Title = "Unresolved threads — press enter to resolve, q to quit"
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ef4444"))
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))

	return ResolveModel{list: l, spinner: sp}
}

func (m ResolveModel) Init() tea.Cmd {
	return nil
}

func (m ResolveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.resolving {
			return m, nil // block input while resolving
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			selected, ok := m.list.SelectedItem().(threadItem)
			if !ok {
				return m, nil
			}
			m.resolving = true
			threadID := selected.thread.ID
			title := selected.Title()
			return m, tea.Batch(
				m.spinner.Tick,
				func() tea.Msg {
					if err := github.ResolveThread(github.GetClient(), context.Background(), threadID); err != nil {
						return resolveErrMsg{err}
					}
					return resolvedMsg{id: threadID, title: title}
				},
			)
		}

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case resolvedMsg:
		m.resolving = false
		m.errMsg = ""
		m.resolved = append(m.resolved, ResolvedThread{ID: msg.id, Title: msg.title})
		// Remove the resolved thread from the list
		items := m.list.Items()
		newItems := make([]list.Item, 0, len(items)-1)
		for _, item := range items {
			if ti, ok := item.(threadItem); ok && ti.thread.ID != msg.id {
				newItems = append(newItems, item)
			}
		}
		cmd := m.list.SetItems(newItems)
		if len(newItems) == 0 {
			m.quitting = true
			return m, tea.Quit
		}
		return m, cmd

	case resolveErrMsg:
		m.resolving = false
		m.errMsg = msg.err.Error()
		return m, nil

	case spinner.TickMsg:
		if m.resolving {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if !m.resolving {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m ResolveModel) View() string {
	if m.resolving {
		return fmt.Sprintf("\n  %s  Resolving thread...\n", m.spinner.View())
	}
	view := m.list.View()
	if m.errMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
		view += "\n  " + errStyle.Render("Error: "+m.errMsg)
	}
	return view
}

// Resolved returns the threads resolved during the session.
func (m ResolveModel) Resolved() []ResolvedThread { return m.resolved }

// RunResolveFlow launches the interactive resolve TUI and returns the resolved threads.
func RunResolveFlow(threads []model.ReviewThread) ([]ResolvedThread, error) {
	m := NewResolveModel(threads)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	return final.(ResolveModel).Resolved(), nil
}
