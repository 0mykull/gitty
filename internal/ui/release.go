package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/0mykull/gitty/internal/git"
	"github.com/0mykull/gitty/internal/styles"
)

type releaseState int

const (
	releaseStateForm releaseState = iota
	releaseStateWorking
	releaseStateDone
	releaseStateError
)

// ReleaseModel handles the release creation flow
type ReleaseModel struct {
	state   releaseState
	spinner spinner.Model
	form    *huh.Form
	tagName string
	message string
	confirm bool
	err     error
}

// NewReleaseModel creates a new release model
func NewReleaseModel() *ReleaseModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return &ReleaseModel{
		state:   releaseStateForm,
		spinner: s,
	}
}

func (m *ReleaseModel) Init() tea.Cmd {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Tag Name").
				Description("e.g. v1.0.0").
				Value(&m.tagName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("tag name cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Message (Optional)").
				Description("Release notes or summary").
				Value(&m.message),

			huh.NewConfirm().
				Title("Create and Push Release?").
				Value(&m.confirm),
		),
	).WithTheme(huh.ThemeCharm())

	return tea.Batch(
		m.spinner.Tick,
		m.form.Init(),
	)
}

func (m *ReleaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case releaseDoneMsg:
		m.state = releaseStateDone
		return m, func() tea.Msg {
			return ReturnToMenuMsg{
				Message: fmt.Sprintf("Release %s created and pushed", m.tagName),
				Type:    "success",
			}
		}

	case releaseErrorMsg:
		m.state = releaseStateError
		m.err = msg.err
		return m, nil
	}

	// Update form
	if m.state == releaseStateForm && m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			if m.confirm {
				m.state = releaseStateWorking
				return m, m.doRelease
			}
			return m, func() tea.Msg {
				return ReturnToMenuMsg{Message: "Release cancelled", Type: "info"}
			}
		}

		return m, cmd
	}

	return m, nil
}

type releaseDoneMsg struct{}
type releaseErrorMsg struct{ err error }

func (m *ReleaseModel) doRelease() tea.Msg {
	// Create the tag
	if err := git.TagAnnotated(m.tagName, m.message); err != nil {
		return releaseErrorMsg{fmt.Errorf("failed to create tag: %w", err)}
	}

	// Push tags
	if err := git.PushTags(); err != nil {
		return releaseErrorMsg{fmt.Errorf("failed to push tags: %w", err)}
	}

	return releaseDoneMsg{}
}

func (m *ReleaseModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(styles.TitleStyle.Render(styles.Icons.Star + " Create Release"))
	b.WriteString("\n\n")

	switch m.state {
	case releaseStateForm:
		if m.form != nil {
			b.WriteString(m.form.View())
		}

	case releaseStateWorking:
		b.WriteString(m.spinner.View() + " Creating and pushing release...")

	case releaseStateDone:
		b.WriteString(styles.RenderSuccess("Release created successfully"))

	case releaseStateError:
		b.WriteString(styles.RenderError(m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Press esc to go back"))
	}

	return b.String()
}
