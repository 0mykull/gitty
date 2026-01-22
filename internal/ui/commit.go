package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/0mykull/gitty/internal/ai"
	"github.com/0mykull/gitty/internal/config"
	"github.com/0mykull/gitty/internal/git"
	"github.com/0mykull/gitty/internal/styles"
)

type commitState int

const (
	commitStateInput commitState = iota
	commitStateGenerating
	commitStateConfirm
	commitStateCommitting
	commitStateDone
	commitStateNoChanges
	commitStateError
)

// CommitModel handles the commit flow
type CommitModel struct {
	cfg         *config.Config
	useAI       bool
	state       commitState
	spinner     spinner.Model
	textInput   textinput.Model
	textArea    textarea.Model
	commitMsg   string
	renderedMsg string
	renderer    *glamour.TermRenderer
	err         error
	diff        string
	ready       bool
}

// NewCommitModel creates a new commit model
func NewCommitModel(cfg *config.Config, useAI bool) *CommitModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.CharLimit = 200
	ti.Width = 60
	ti.Focus()

	ta := textarea.New()
	ta.Placeholder = "Enter detailed commit message (optional)..."
	ta.SetWidth(60)
	ta.SetHeight(5)

	return &CommitModel{
		cfg:       cfg,
		useAI:     useAI,
		spinner:   s,
		textInput: ti,
		textArea:  ta,
		renderer:  nil, // Will be initialized async
		ready:     false,
	}
}

func (m *CommitModel) Init() tea.Cmd {
	// Start checking status and init renderer in parallel
	return tea.Batch(
		m.spinner.Tick,
		textinput.Blink,
		m.checkStatusAsync,
		m.initRendererCmd,
	)
}

func (m *CommitModel) initRendererCmd() tea.Msg {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return rendererMsg{nil}
	}
	return rendererMsg{r}
}

type rendererMsg struct {
	renderer *glamour.TermRenderer
}

// checkStatusAsync checks git status without blocking
func (m *CommitModel) checkStatusAsync() tea.Msg {
	// Fast check for staged changes using exit code
	if !git.HasStagedChanges() {
		return commitNoChangesMsg{}
	}

	// For manual commit, we don't need the diff immediately
	if !m.useAI {
		return commitReadyMsg{diff: ""}
	}

	// For AI commit, we need the diff
	diff, err := git.GetDiff()
	if err != nil {
		return commitErrorMsg{err}
	}

	return commitReadyMsg{diff: diff}
}

type commitReadyMsg struct {
	diff string
}

type commitNoChangesMsg struct{}

type commitErrorMsg struct {
	err error
}

type commitGeneratedMsg struct {
	message string
}

type commitDoneMsg struct{}

func (m *CommitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg {
				return ReturnToMenuMsg{Message: "Cancelled", Type: "info"}
			}
		case "enter":
			// Submit on Enter
			if m.state == commitStateInput {
				return m.submitForm()
			}
			return m.handleEnter()

		case "alt+enter":
			// Newline on Alt+Enter
			if m.state == commitStateInput {
				if m.textArea.Focused() {
					m.textArea.InsertString("\n")
					return m, nil
				}
			}
		case "tab":
			// Switch between title and body in manual mode
			if m.state == commitStateInput && !m.useAI {
				if m.textInput.Focused() {
					m.textInput.Blur()
					m.textArea.Focus()
				} else {
					m.textArea.Blur()
					m.textInput.Focus()
				}
			}
		case "y", "Y":
			if m.state == commitStateConfirm {
				m.state = commitStateCommitting
				return m, m.doCommit
			}
		case "n", "N":
			if m.state == commitStateConfirm {
				return m, func() tea.Msg {
					return ReturnToMenuMsg{Message: "Commit cancelled", Type: "info"}
				}
			}
		case "e", "E":
			if m.state == commitStateConfirm {
				// Edit the message
				m.textInput.SetValue(strings.Split(m.commitMsg, "\n")[0])
				if parts := strings.SplitN(m.commitMsg, "\n\n", 2); len(parts) > 1 {
					m.textArea.SetValue(parts[1])
				}
				m.textInput.Focus()
				m.state = commitStateInput
				return m, textinput.Blink
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case commitReadyMsg:
		m.diff = msg.diff
		m.ready = true

		if m.useAI {
			// For AI commit, start generating immediately
			m.state = commitStateGenerating
			return m, m.generateMessage
		}
		// For manual commit, show input immediately
		m.state = commitStateInput
		return m, textinput.Blink

	case rendererMsg:
		m.renderer = msg.renderer
		return m, nil

	case commitNoChangesMsg:
		m.state = commitStateNoChanges
		return m, nil

	case commitGeneratedMsg:
		m.commitMsg = msg.message
		m.renderedMsg = m.renderMessage(msg.message)
		m.state = commitStateConfirm
		return m, nil

	case commitErrorMsg:
		m.state = commitStateError
		m.err = msg.err
		return m, nil

	case commitDoneMsg:
		m.state = commitStateDone
		return m, func() tea.Msg {
			return ReturnToMenuMsg{Message: "Commit successful!", Type: "success"}
		}
	}

	// Update text inputs when in input state
	if m.state == commitStateInput {
		var cmd tea.Cmd
		if m.textInput.Focused() {
			m.textInput, cmd = m.textInput.Update(msg)
		} else {
			m.textArea, cmd = m.textArea.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m *CommitModel) submitForm() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.textInput.Value())
	if title == "" {
		return m, nil
	}

	body := strings.TrimSpace(m.textArea.Value())
	if body != "" {
		m.commitMsg = title + "\n\n" + body
	} else {
		m.commitMsg = title
	}
	m.renderedMsg = m.renderMessage(m.commitMsg)
	m.state = commitStateConfirm
	return m, nil
}

func (m *CommitModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case commitStateNoChanges:
		return m, func() tea.Msg {
			return ReturnToMenuMsg{Message: "No staged changes to commit", Type: "info"}
		}

	case commitStateError:
		return m, func() tea.Msg {
			return ReturnToMenuMsg{Message: fmt.Sprintf("Error: %v", m.err), Type: "error"}
		}
	}

	return m, nil
}

func (m *CommitModel) generateMessage() tea.Msg {
	msg, err := ai.GenerateCommitMessage(m.diff, m.cfg)
	if err != nil {
		return commitErrorMsg{err}
	}
	return commitGeneratedMsg{msg}
}

func (m *CommitModel) doCommit() tea.Msg {
	if err := git.Commit(m.commitMsg); err != nil {
		return commitErrorMsg{err}
	}
	return commitDoneMsg{}
}

func (m *CommitModel) renderMessage(msg string) string {
	if m.renderer == nil {
		return msg // Fallback to plain text if renderer isn't ready yet
	}

	out, err := m.renderer.Render(msg)
	if err != nil {
		return msg
	}
	return out
}

func (m *CommitModel) View() string {
	var b strings.Builder

	// Header
	title := styles.Icons.Commit + " "
	if m.useAI {
		title += "AI Commit"
	} else {
		title += "Commit"
	}
	b.WriteString(styles.TitleStyle.Render(title))
	b.WriteString("\n\n")

	switch m.state {
	case commitStateInput:
		if !m.ready && !m.useAI {
			// Still loading, show spinner briefly
			b.WriteString(m.spinner.View() + " Checking status...")
		} else {
			b.WriteString("Enter your commit message:\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Purple).Render("Title:") + "\n")
			b.WriteString(m.textInput.View())
			b.WriteString("\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Purple).Render("Body (optional):") + "\n")
			b.WriteString(m.textArea.View())
			b.WriteString("\n\n")
			b.WriteString(styles.HelpStyle.Render("tab: switch fields • enter: commit • alt+enter: new line • esc: cancel"))
		}

	case commitStateGenerating:
		b.WriteString(m.spinner.View() + " Generating commit message with AI...")
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("This may take a few seconds..."))

	case commitStateNoChanges:
		b.WriteString(styles.WarningStyle.Render(styles.Icons.Warning + " No staged changes"))
		b.WriteString("\n\n")
		b.WriteString("You need to stage changes before committing.\n")
		b.WriteString("Use 'Stage All' (a) from the menu or 'git add <file>'.")
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Press enter or esc to go back"))

	case commitStateConfirm:
		b.WriteString("Commit message:\n")
		box := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Purple).
			Padding(1, 2).
			Render(m.renderedMsg)
		b.WriteString(box)
		b.WriteString("\n\n")
		b.WriteString(styles.InfoStyle.Render("Commit with this message?"))
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("y: confirm • n: cancel • e: edit"))

	case commitStateCommitting:
		b.WriteString(m.spinner.View() + " Committing changes...")

	case commitStateDone:
		b.WriteString(styles.RenderSuccess("Commit successful!"))

	case commitStateError:
		b.WriteString(styles.RenderError(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Press enter or esc to go back"))
	}

	return b.String()
}
