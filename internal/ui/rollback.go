package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/0mykull/gitty/internal/git"
	"github.com/0mykull/gitty/internal/styles"
)

type rollbackState int

const (
	rollbackStateConfirm rollbackState = iota
	rollbackStateWorking
	rollbackStateDone
	rollbackStateError
)

// RollbackModel handles the rollback confirmation flow
type RollbackModel struct {
	state     rollbackState
	spinner   spinner.Model
	form      *huh.Form
	confirmed bool
	err       error
}

// NewRollbackModel creates a new rollback confirmation model
func NewRollbackModel() *RollbackModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return &RollbackModel{
		state:     rollbackStateConfirm,
		spinner:   s,
		confirmed: false,
	}
}

func (m *RollbackModel) Init() tea.Cmd {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Rollback last commit?").
				Description("This will discard the last commit and all changes (git reset --hard HEAD^)").
				Affirmative("Yes, rollback").
				Negative("Cancel").
				Value(&m.confirmed),
		),
	).WithTheme(huh.ThemeCharm())

	return m.form.Init()
}

func (m *RollbackModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			return m, func() tea.Msg {
				return ReturnToMenuMsg{Message: "", Type: ""}
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case rollbackDoneMsg:
		m.state = rollbackStateDone
		return m, func() tea.Msg {
			return ReturnToMenuMsg{Message: "Rollback successful", Type: "success"}
		}

	case rollbackErrorMsg:
		m.state = rollbackStateError
		m.err = msg.err
		return m, nil
	}

	// Update form
	if m.state == rollbackStateConfirm && m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			if m.confirmed {
				m.state = rollbackStateWorking
				return m, m.doRollback
			}
			return m, func() tea.Msg {
				return ReturnToMenuMsg{Message: "Rollback cancelled", Type: "info"}
			}
		}

		return m, cmd
	}

	return m, nil
}

type rollbackDoneMsg struct{}
type rollbackErrorMsg struct{ err error }

func (m *RollbackModel) doRollback() tea.Msg {
	if err := git.Rollback(); err != nil {
		return rollbackErrorMsg{err}
	}
	return rollbackDoneMsg{}
}

func (m *RollbackModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(styles.TitleStyle.Render(styles.Icons.Reset + " Rollback Commit"))
	b.WriteString("\n\n")

	switch m.state {
	case rollbackStateConfirm:
		if m.form != nil {
			b.WriteString(m.form.View())
		}

	case rollbackStateWorking:
		b.WriteString(m.spinner.View() + " Rolling back...")

	case rollbackStateDone:
		b.WriteString(styles.RenderSuccess("Rollback complete"))

	case rollbackStateError:
		b.WriteString(styles.RenderError(m.err.Error()))
	}

	return b.String()
}
