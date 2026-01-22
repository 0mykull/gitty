package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/0mykull/gitty/internal/config"
	"github.com/0mykull/gitty/internal/git"
	"github.com/0mykull/gitty/internal/styles"
)

type publishState int

const (
	publishStateInit publishState = iota
	publishStateCheckRepo
	publishStateForm
	publishStateConfirm
	publishStateWorking
	publishStateDone
	publishStateError
)

// PublishModel handles the GitHub publish flow
type PublishModel struct {
	cfg         *config.Config
	state       publishState
	spinner     spinner.Model
	form        *huh.Form
	repoName    string
	description string
	visibility  string
	commitMsg   string
	addTag      bool
	tagName     string
	hasRemote   bool
	branch      string
	err         error
	repoURL     string

	// Text inputs for step-by-step
	nameInput textinput.Model
	descInput textinput.Model
	tagInput  textinput.Model
}

// NewPublishModel creates a new publish model
func NewPublishModel(cfg *config.Config) *PublishModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	// Get default repo name from directory
	defaultName := git.GetRepoName()

	ni := textinput.New()
	ni.Placeholder = defaultName
	ni.SetValue(defaultName)
	ni.Focus()
	ni.CharLimit = 100
	ni.Width = 40

	di := textinput.New()
	di.Placeholder = "Optional description..."
	di.CharLimit = 200
	di.Width = 40

	ti := textinput.New()
	ti.Placeholder = "v1.0.0"
	ti.CharLimit = 20
	ti.Width = 20

	return &PublishModel{
		cfg:        cfg,
		state:      publishStateInit,
		spinner:    s,
		visibility: cfg.GitHub.DefaultVisibility,
		nameInput:  ni,
		descInput:  di,
		tagInput:   ti,
	}
}

func (m *PublishModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.checkRepo,
	)
}

func (m *PublishModel) checkRepo() tea.Msg {
	// Check if we're in a git repo
	if !git.IsRepo() {
		// Initialize git
		if err := git.Init(); err != nil {
			return publishErrorMsg{fmt.Errorf("failed to initialize git: %w", err)}
		}
	}

	// Get current branch
	branch, _ := git.GetBranch()
	if branch == "" {
		branch = "main"
	}

	// Check if remote exists
	hasRemote := git.HasRemote("origin")

	return publishRepoCheckedMsg{
		branch:    branch,
		hasRemote: hasRemote,
	}
}

type publishRepoCheckedMsg struct {
	branch    string
	hasRemote bool
}

type publishErrorMsg struct{ err error }
type publishDoneMsg struct{ url string }

func (m *PublishModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg {
				return ReturnToMenuMsg{Message: "", Type: ""}
			}
		case "enter":
			// Only handle Enter manually if we are NOT in the form
			if m.state != publishStateForm {
				return m.handleEnter()
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case publishRepoCheckedMsg:
		m.branch = msg.branch
		m.hasRemote = msg.hasRemote

		if msg.hasRemote {
			// Already has remote, just push
			m.state = publishStateWorking
			return m, m.pushToRemote
		}

		// Show form for new repo
		m.state = publishStateForm
		return m, m.initForm()

	case publishErrorMsg:
		m.state = publishStateError
		m.err = msg.err
		return m, nil

	case publishDoneMsg:
		m.state = publishStateDone
		m.repoURL = msg.url
		return m, func() tea.Msg {
			return ReturnToMenuMsg{
				Message: fmt.Sprintf("Published to %s", msg.url),
				Type:    "success",
			}
		}
	}

	// Update form if in form state
	if m.state == publishStateForm && m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			m.state = publishStateConfirm
			return m, nil
		}

		return m, cmd
	}

	return m, nil
}

func (m *PublishModel) initForm() tea.Cmd {
	defaultName := git.GetRepoName()

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Repository name").
				Value(&m.repoName).
				Placeholder(defaultName),

			huh.NewInput().
				Title("Description (optional)").
				Value(&m.description).
				Placeholder("A brief description..."),

			huh.NewSelect[string]().
				Title("Visibility").
				Options(
					huh.NewOption("Public", "public"),
					huh.NewOption("Private", "private"),
				).
				Value(&m.visibility),

			huh.NewInput().
				Title("Commit message").
				Value(&m.commitMsg).
				Placeholder("Initial commit"),

			huh.NewConfirm().
				Title("Add version tag?").
				Value(&m.addTag),
		),
	).WithTheme(huh.ThemeCharm())

	// Set defaults
	if m.repoName == "" {
		m.repoName = defaultName
	}
	if m.commitMsg == "" {
		m.commitMsg = "Initial commit"
	}

	return m.form.Init()
}

func (m *PublishModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case publishStateConfirm:
		m.state = publishStateWorking
		return m, m.doPublish

	case publishStateError:
		return m, func() tea.Msg {
			return ReturnToMenuMsg{Message: fmt.Sprintf("Error: %v", m.err), Type: "error"}
		}

	case publishStateDone:
		return m, func() tea.Msg {
			return ReturnToMenuMsg{
				Message: fmt.Sprintf("Published to %s", m.repoURL),
				Type:    "success",
			}
		}
	}

	return m, nil
}

func (m *PublishModel) pushToRemote() tea.Msg {
	// Stage and commit any changes
	status, _ := git.GetStatus()
	if status.HasUnstaged || status.HasUntracked {
		git.AddAll()
		git.Commit("Update")
	}

	// Push
	if err := git.PushWithUpstream("origin", m.branch); err != nil {
		return publishErrorMsg{err}
	}

	url, _ := git.GetGitHubURL()
	return publishDoneMsg{url}
}

func (m *PublishModel) doPublish() tea.Msg {
	// Configure git user if specified
	if m.cfg.Git.UserName != "" && m.cfg.Git.UserEmail != "" {
		git.SetUser(m.cfg.Git.UserName, m.cfg.Git.UserEmail)
	}

	// Stage all changes
	if err := git.AddAll(); err != nil {
		return publishErrorMsg{fmt.Errorf("failed to stage changes: %w", err)}
	}

	// Check if there are changes to commit
	status, _ := git.GetStatus()
	if status.HasStaged {
		if err := git.Commit(m.commitMsg); err != nil {
			return publishErrorMsg{fmt.Errorf("failed to commit: %w", err)}
		}
	}

	// Add tag if requested
	if m.addTag && m.tagName != "" {
		if err := git.Tag(m.tagName); err != nil {
			// Tag might already exist, ignore error
		}
	}

	// Create GitHub repo using gh CLI
	args := []string{"repo", "create", m.repoName, "--" + m.visibility, "--source=.", "--remote=origin", "--push"}
	if m.description != "" {
		args = append(args, fmt.Sprintf("--description=%s", m.description))
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir, _ = os.Getwd()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return publishErrorMsg{fmt.Errorf("gh cli error: %s - %w", string(output), err)}
	}

	// Get the URL
	url, _ := git.GetGitHubURL()
	if url == "" {
		// Try to construct it
		user := os.Getenv("GITHUB_USER")
		if user == "" {
			user = "user"
		}
		url = fmt.Sprintf("https://github.com/%s/%s", user, m.repoName)
	}

	return publishDoneMsg{url}
}

func (m *PublishModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(styles.TitleStyle.Render(styles.Icons.Publish + " Publish to GitHub"))
	b.WriteString("\n\n")

	switch m.state {
	case publishStateInit, publishStateCheckRepo:
		b.WriteString(m.spinner.View() + " Checking repository...")

	case publishStateForm:
		if m.form != nil {
			b.WriteString(m.form.View())
		}

	case publishStateConfirm:
		b.WriteString("Ready to publish:\n\n")

		info := []string{
			fmt.Sprintf("  %s Repository: %s", styles.Icons.Folder, m.repoName),
			fmt.Sprintf("  %s Visibility: %s", styles.Icons.Git, m.visibility),
			fmt.Sprintf("  %s Branch: %s", styles.Icons.Branch, m.branch),
		}
		if m.description != "" {
			info = append(info, fmt.Sprintf("  %s Description: %s", styles.Icons.File, m.description))
		}
		if m.addTag {
			info = append(info, fmt.Sprintf("  %s Tag: %s", styles.Icons.Star, m.tagName))
		}

		b.WriteString(strings.Join(info, "\n"))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Press enter to publish, esc to cancel"))

	case publishStateWorking:
		b.WriteString(m.spinner.View() + " Publishing to GitHub...")
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("Creating repository and pushing code..."))

	case publishStateDone:
		b.WriteString(styles.RenderSuccess("Published successfully!"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.Icons.Open, m.repoURL))
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("Press enter to continue"))

	case publishStateError:
		b.WriteString(styles.RenderError(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")

		// Check for common issues
		if strings.Contains(m.err.Error(), "gh") {
			b.WriteString(styles.WarningStyle.Render("Make sure you have the GitHub CLI (gh) installed and authenticated."))
			b.WriteString("\n")
			b.WriteString(styles.HelpStyle.Render("Run: gh auth login"))
		}
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("Press enter to go back"))
	}

	return b.String()
}
