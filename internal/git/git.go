package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Status represents the current git repository status
type Status struct {
	IsRepo         bool
	Branch         string
	HasStaged      bool
	HasUnstaged    bool
	HasUntracked   bool
	Ahead          int
	Behind         int
	StagedFiles    []string
	ModifiedFiles  []string
	UntrackedFiles []string
	RemoteURL      string
}

// GetStatus returns the current git status
func GetStatus() (*Status, error) {
	status := &Status{}

	// Check if we're in a git repo
	if !IsRepo() {
		return status, nil
	}
	status.IsRepo = true

	// Get current branch
	branch, err := GetBranch()
	if err == nil {
		status.Branch = branch
	}

	// Get remote URL
	url, _ := GetRemoteURL()
	status.RemoteURL = url

	// Get porcelain status
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return status, nil
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		x := line[0]
		y := line[1]
		file := strings.TrimSpace(line[3:])

		// Staged changes (index)
		if x != ' ' && x != '?' {
			status.HasStaged = true
			status.StagedFiles = append(status.StagedFiles, file)
		}

		// Unstaged changes (worktree)
		if y != ' ' && y != '?' {
			status.HasUnstaged = true
			status.ModifiedFiles = append(status.ModifiedFiles, file)
		}

		// Untracked files
		if x == '?' && y == '?' {
			status.HasUntracked = true
			status.UntrackedFiles = append(status.UntrackedFiles, file)
		}
	}

	// Get ahead/behind counts
	aheadBehind, _ := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}").Output()
	if len(aheadBehind) > 0 {
		parts := strings.Fields(string(aheadBehind))
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &status.Ahead)
			fmt.Sscanf(parts[1], "%d", &status.Behind)
		}
	}

	return status, nil
}

// IsRepo checks if current directory is a git repository
func IsRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// Init initializes a new git repository
func Init() error {
	cmd := exec.Command("git", "init")
	return cmd.Run()
}

// GetBranch returns the current branch name
func GetBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "main", nil
	}
	return branch, nil
}

// Add stages files for commit
func Add(files ...string) error {
	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	return cmd.Run()
}

// AddAll stages all changes
func AddAll() error {
	return Add(".")
}

// Commit creates a commit with the given message
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}

// Push pushes to remote
func Push() error {
	cmd := exec.Command("git", "push")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", string(output), err)
	}
	return nil
}

// PushWithUpstream pushes and sets upstream
func PushWithUpstream(remote, branch string) error {
	cmd := exec.Command("git", "push", "-u", remote, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", string(output), err)
	}
	return nil
}

// Pull pulls from remote
func Pull() error {
	cmd := exec.Command("git", "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", string(output), err)
	}
	return nil
}

// Reset performs a hard reset
func Reset() error {
	cmd := exec.Command("git", "reset", "--hard")
	return cmd.Run()
}

// Rollback resets to previous commit
func Rollback() error {
	cmd := exec.Command("git", "reset", "--hard", "HEAD^")
	return cmd.Run()
}

// HasStagedChanges checks if there are any staged changes
func HasStagedChanges() bool {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()
	// Exit code 1 means differences were found (changes exist)
	// Exit code 0 means no differences (clean)
	return err != nil
}

// GetDiff returns the staged diff
func GetDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetFullDiff returns both staged and unstaged diff
func GetFullDiff() (string, error) {
	cmd := exec.Command("git", "diff", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetRemoteURL returns the origin remote URL
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SetConfig sets a git config value
func SetConfig(key, value string) error {
	cmd := exec.Command("git", "config", key, value)
	return cmd.Run()
}

// SetUser sets the user name and email
func SetUser(name, email string) error {
	if err := SetConfig("user.name", name); err != nil {
		return err
	}
	return SetConfig("user.email", email)
}

// GetBranches returns all branches
func GetBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, line := range strings.Split(string(output), "\n") {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" {
			branches = append(branches, branch)
		}
	}
	return branches, nil
}

// CreateBranch creates and checks out a new branch
func CreateBranch(name string) error {
	cmd := exec.Command("git", "checkout", "-b", name)
	return cmd.Run()
}

// Checkout switches to a branch
func Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	return cmd.Run()
}

// GetRepoName returns the repository name from the current directory
func GetRepoName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "repo"
	}
	return filepath.Base(cwd)
}

// HasRemote checks if a remote exists
func HasRemote(name string) bool {
	cmd := exec.Command("git", "remote", "get-url", name)
	return cmd.Run() == nil
}

// AddRemote adds a new remote
func AddRemote(name, url string) error {
	cmd := exec.Command("git", "remote", "add", name, url)
	return cmd.Run()
}

// Tag creates a new tag
func Tag(name string) error {
	cmd := exec.Command("git", "tag", name)
	return cmd.Run()
}

// TagAnnotated creates a new annotated tag with a message
func TagAnnotated(name, message string) error {
	var cmd *exec.Cmd
	if message == "" {
		cmd = exec.Command("git", "tag", name)
	} else {
		cmd = exec.Command("git", "tag", "-a", name, "-m", message)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", string(output), err)
	}
	return nil
}

// PushTags pushes all tags to remote
func PushTags() error {
	cmd := exec.Command("git", "push", "--tags")
	return cmd.Run()
}

// GetGitHubURL converts git URL to GitHub web URL
func GetGitHubURL() (string, error) {
	url, err := GetRemoteURL()
	if err != nil {
		return "", err
	}

	if !strings.Contains(url, "github.com") {
		return "", fmt.Errorf("not a GitHub repository")
	}

	// Convert SSH to HTTPS
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
		url = "https://" + url
	}

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	return url, nil
}

// OpenBrowser opens a URL in the default browser
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	// Try zen-browser first, then xdg-open
	if _, err := exec.LookPath("zen-browser"); err == nil {
		cmd = exec.Command("zen-browser", url)
	} else if _, err := exec.LookPath("zen"); err == nil {
		cmd = exec.Command("zen", url)
	} else {
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

// CheckDeps checks for required and optional dependencies
func CheckDeps() []string {
	var missing []string

	// Required
	if _, err := exec.LookPath("git"); err != nil {
		missing = append(missing, "git (required)")
	}

	// Optional
	if _, err := exec.LookPath("gh"); err != nil {
		missing = append(missing, "gh (optional, for publish)")
	}
	if _, err := exec.LookPath("lazygit"); err != nil {
		missing = append(missing, "lazygit (optional)")
	}

	return missing
}
