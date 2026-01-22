package ui

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/0mykull/gitty/internal/config"
	"github.com/0mykull/gitty/internal/git"
	"github.com/0mykull/gitty/internal/styles"
)

// Action represents what a menu item does
type Action int

const (
	ActionNone Action = iota
	ActionAdd
	ActionCommit
	ActionAICommit
	ActionPush
	ActionPull
	ActionReset
	ActionRollback
	ActionPublish
	ActionOpen
	ActionLazygit
	ActionBranches
	ActionQuit
)

// menuItem implements list.Item
type menuItem struct {
	icon     string
	title    string
	desc     string
	shortcut string
	action   Action
}

func (i menuItem) Title() string       { return i.icon + "  " + i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

// Custom item delegate for beautiful rendering
type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(menuItem)
	if !ok {
		return
	}

	// Build the line
	var line string
	isSelected := index == m.Index()

	if isSelected {
		// Selected style: arrow + icon + title with pink color
		arrow := lipgloss.NewStyle().Foreground(styles.Pink).Render("  " + styles.Icons.Arrow + " ")
		icon := lipgloss.NewStyle().Foreground(styles.Purple).Render(i.icon)
		title := lipgloss.NewStyle().Foreground(styles.Pink).Bold(true).Render(" " + i.title)
		shortcut := lipgloss.NewStyle().Foreground(styles.Blue).Render(" [" + i.shortcut + "]")
		line = arrow + icon + title + shortcut
	} else {
		// Normal style
		space := "     "
		icon := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(i.icon)
		title := lipgloss.NewStyle().Foreground(styles.TextPrimary).Render(" " + i.title)
		shortcut := lipgloss.NewStyle().Foreground(styles.TextMuted).Render(" [" + i.shortcut + "]")
		line = space + icon + title + shortcut
	}

	fmt.Fprint(w, line)
}

// Model is the main menu model
type Model struct {
	list     list.Model
	items    []menuItem
	cfg      *config.Config
	status   *git.Status
	spinner  spinner.Model
	loading  bool
	message  string
	msgType  string // "success", "error", "info"
	width    int
	height   int
	quitting bool

	// Sub-models
	subModel  tea.Model
	inSubView bool
}

// NewModel creates a new menu model
func NewModel(cfg *config.Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	items := []menuItem{
		{icon: styles.Icons.Add, title: "Stage All", desc: "git add .", shortcut: "a", action: ActionAdd},
		{icon: styles.Icons.Commit, title: "Commit", desc: "Commit with message", shortcut: "c", action: ActionCommit},
		{icon: styles.Icons.AI, title: "AI Commit", desc: "Generate commit message with AI", shortcut: "i", action: ActionAICommit},
		{icon: styles.Icons.Push, title: "Push", desc: "Push to remote", shortcut: "p", action: ActionPush},
		{icon: styles.Icons.Pull, title: "Pull", desc: "Pull from remote", shortcut: "l", action: ActionPull},
		{icon: styles.Icons.Reset, title: "Reset", desc: "Reset changes (hard)", shortcut: "r", action: ActionReset},
		{icon: styles.Icons.Publish, title: "Publish", desc: "Publish to GitHub", shortcut: "u", action: ActionPublish},
		{icon: styles.Icons.Open, title: "Open Repo", desc: "Open repo in browser", shortcut: "o", action: ActionOpen},
		{icon: styles.Icons.Lazygit, title: "Lazygit", desc: "Open lazygit", shortcut: "g", action: ActionLazygit},
		{icon: styles.Icons.Branch, title: "Branches", desc: "View branches", shortcut: "b", action: ActionBranches},
		{icon: styles.Icons.Quit, title: "Quit", desc: "Exit gitty", shortcut: "q", action: ActionQuit},
	}

	// Convert to list.Item slice
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create list with custom delegate
	delegate := itemDelegate{}
	l := list.New(listItems, delegate, 50, len(items)+2)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	return Model{
		list:    l,
		items:   items,
		cfg:     cfg,
		spinner: s,
		width:   80,
		height:  24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.refreshStatus,
	)
}

// refreshStatus fetches git status
func (m Model) refreshStatus() tea.Msg {
	status, err := git.GetStatus()
	if err != nil {
		return statusMsg{err: err}
	}
	return statusMsg{status: status}
}

type statusMsg struct {
	status *git.Status
	err    error
}

type actionCompleteMsg struct {
	success bool
	message string
}

type clearMsgMsg struct{}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle sub-view updates
	if m.inSubView && m.subModel != nil {
		var cmd tea.Cmd
		m.subModel, cmd = m.subModel.Update(msg)

		// Check if sub-view wants to return
		if returnMsg, ok := msg.(ReturnToMenuMsg); ok {
			m.inSubView = false
			m.subModel = nil
			if returnMsg.Message != "" {
				m.message = returnMsg.Message
				m.msgType = returnMsg.Type
			}
			return m, tea.Batch(m.refreshStatus, clearMessageAfter())
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter", " ":
			if item, ok := m.list.SelectedItem().(menuItem); ok {
				return m.executeAction(item.action)
			}

		default:
			// Handle shortcut keys
			for _, item := range m.items {
				if msg.String() == item.shortcut {
					return m.executeAction(item.action)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case statusMsg:
		m.status = msg.status
		m.loading = false

	case actionCompleteMsg:
		m.loading = false
		m.message = msg.message
		if msg.success {
			m.msgType = "success"
		} else {
			m.msgType = "error"
		}
		return m, tea.Batch(m.refreshStatus, clearMessageAfter())

	case clearMsgMsg:
		m.message = ""
		m.msgType = ""
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func clearMessageAfter() tea.Cmd {
	return tea.Tick(time.Second*3, func(_ time.Time) tea.Msg {
		return clearMsgMsg{}
	})
}

func (m Model) executeAction(action Action) (tea.Model, tea.Cmd) {
	switch action {
	case ActionQuit:
		m.quitting = true
		return m, tea.Quit

	case ActionAdd:
		m.loading = true
		return m, func() tea.Msg {
			if err := git.AddAll(); err != nil {
				return actionCompleteMsg{false, fmt.Sprintf("Failed to add: %v", err)}
			}
			return actionCompleteMsg{true, "All files staged"}
		}

	case ActionPush:
		m.loading = true
		return m, func() tea.Msg {
			if err := git.Push(); err != nil {
				return actionCompleteMsg{false, fmt.Sprintf("Push failed: %v", err)}
			}
			return actionCompleteMsg{true, "Pushed to remote"}
		}

	case ActionPull:
		m.loading = true
		return m, func() tea.Msg {
			if err := git.Pull(); err != nil {
				return actionCompleteMsg{false, fmt.Sprintf("Pull failed: %v", err)}
			}
			return actionCompleteMsg{true, "Pulled from remote"}
		}

	case ActionReset:
		m.inSubView = true
		m.subModel = NewResetModel()
		return m, m.subModel.Init()

	case ActionCommit:
		m.inSubView = true
		m.subModel = NewCommitModel(m.cfg, false)
		return m, m.subModel.Init()

	case ActionAICommit:
		m.inSubView = true
		m.subModel = NewCommitModel(m.cfg, true)
		return m, m.subModel.Init()

	case ActionPublish:
		m.inSubView = true
		m.subModel = NewPublishModel(m.cfg)
		return m, m.subModel.Init()

	case ActionOpen:
		m.loading = true
		return m, func() tea.Msg {
			url, err := git.GetGitHubURL()
			if err != nil {
				return actionCompleteMsg{false, fmt.Sprintf("Not a GitHub repo: %v", err)}
			}
			if err := git.OpenBrowser(url); err != nil {
				return actionCompleteMsg{false, fmt.Sprintf("Failed to open: %v", err)}
			}
			return actionCompleteMsg{true, "Opened in browser"}
		}

	case ActionLazygit:
		c := exec.Command("lazygit")
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				return actionCompleteMsg{false, fmt.Sprintf("Lazygit error: %v", err)}
			}
			return actionCompleteMsg{true, ""}
		})

	case ActionBranches:
		m.loading = true
		return m, func() tea.Msg {
			branches, err := git.GetBranches()
			if err != nil {
				return actionCompleteMsg{false, fmt.Sprintf("Failed to get branches: %v", err)}
			}
			return actionCompleteMsg{true, fmt.Sprintf("Branches: %s", strings.Join(branches, ", "))}
		}
	}

	return m, nil
}

// View renders the menu
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Render sub-view if active
	if m.inSubView && m.subModel != nil {
		return m.subModel.View()
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(styles.Divider(m.width))
	b.WriteString("\n")

	// Menu list
	b.WriteString(m.list.View())

	// Message
	if m.message != "" {
		b.WriteString("\n")
		switch m.msgType {
		case "success":
			b.WriteString(styles.RenderSuccess(m.message))
		case "error":
			b.WriteString(styles.RenderError(m.message))
		default:
			b.WriteString(styles.RenderInfo(m.message))
		}
	}

	// Loading indicator
	if m.loading {
		b.WriteString("\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" Working...")
	}

	// Help
	b.WriteString("\n\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Pink).
		Render("gitty")

	// Separator
	separator := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Render(" | ")

	// Branch info (if in a repo)
	var branchInfo string
	if m.status != nil && m.status.IsRepo {
		branch := lipgloss.NewStyle().Foreground(styles.Cyan).Bold(true).Render(m.status.Branch)

		var statusParts []string
		if m.status.HasStaged {
			statusParts = append(statusParts, styles.SuccessStyle.Render(fmt.Sprintf("+%d", len(m.status.StagedFiles))))
		}
		if m.status.HasUnstaged {
			statusParts = append(statusParts, styles.WarningStyle.Render(fmt.Sprintf("~%d", len(m.status.ModifiedFiles))))
		}
		if m.status.HasUntracked {
			statusParts = append(statusParts, styles.InfoStyle.Render(fmt.Sprintf("?%d", len(m.status.UntrackedFiles))))
		}
		if m.status.Ahead > 0 {
			statusParts = append(statusParts, lipgloss.NewStyle().Foreground(styles.Blue).Render(fmt.Sprintf("↑%d", m.status.Ahead)))
		}
		if m.status.Behind > 0 {
			statusParts = append(statusParts, lipgloss.NewStyle().Foreground(styles.Yellow).Render(fmt.Sprintf("↓%d", m.status.Behind)))
		}
		if !m.status.HasStaged && !m.status.HasUnstaged && !m.status.HasUntracked {
			statusParts = append(statusParts, styles.SuccessStyle.Render(styles.Icons.Check))
		}

		branchInfo = branch
		if len(statusParts) > 0 {
			branchInfo += "  " + strings.Join(statusParts, " ")
		}
	} else {
		branchInfo = styles.WarningStyle.Render(styles.Icons.Warning + " Not a git repo")
	}

	// Join with pipe separator
	return title + separator + branchInfo
}

func (m Model) renderHelp() string {
	keyStyle := lipgloss.NewStyle().Foreground(styles.Purple)
	descStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)

	help := []string{
		keyStyle.Render("↑↓") + descStyle.Render(" navigate"),
		keyStyle.Render("enter") + descStyle.Render(" select"),
		keyStyle.Render("q") + descStyle.Render(" quit"),
	}
	return strings.Join(help, "  ")
}

// ReturnToMenuMsg signals return to main menu
type ReturnToMenuMsg struct {
	Message string
	Type    string
}
