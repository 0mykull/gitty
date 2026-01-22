package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/1mykull/gitty/internal/git"
	"github.com/1mykull/gitty/internal/styles"
)

type resetState int

const (
	resetStateConfirm resetState = iota
	resetStateWorking
	resetStateDone
	resetStateError
)

// ResetModel handles the reset confirmation flow
type ResetModel struct {
	state     resetState
	spinner   spinner.Model
	form      *huh.Form
	confirmed bool
	err       error
}

// NewResetModel creates a new reset confirmation model
func NewResetModel() *ResetModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return &ResetModel{
		state:     resetStateConfirm,
		spinner:   s,
		confirmed: false,
	}
}

func (m *ResetModel) Init() tea.Cmd {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Reset all changes?").
				Description("This will discard all uncommitted changes (git reset --hard)").
				Affirmative("Yes, reset").
				Negative("Cancel").
				Value(&m.confirmed),
		),
	).WithTheme(huh.ThemeCharm())

	return m.form.Init()
}

func (m *ResetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	}

	// Update form
	if m.state == resetStateConfirm && m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			if m.confirmed {
				m.state = resetStateWorking
				return m, m.doReset
			}
			return m, func() tea.Msg {
				return ReturnToMenuMsg{Message: "Reset cancelled", Type: "info"}
			}
		}

		return m, cmd
	}

	return m, nil
}

type resetDoneMsg struct{}
type resetErrorMsg struct{ err error }

func (m *ResetModel) doReset() tea.Msg {
	if err := git.Reset(); err != nil {
		return resetErrorMsg{err}
	}
	return resetDoneMsg{}
}

func (m *ResetModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(styles.TitleStyle.Render(styles.Icons.Reset + " Reset Changes"))
	b.WriteString("\n\n")

	switch m.state {
	case resetStateConfirm:
		if m.form != nil {
			b.WriteString(m.form.View())
		}

	case resetStateWorking:
		b.WriteString(m.spinner.View() + " Resetting...")

	case resetStateDone:
		b.WriteString(styles.RenderSuccess("Reset complete"))

	case resetStateError:
		b.WriteString(styles.RenderError(m.err.Error()))
	}

	return b.String()
}
