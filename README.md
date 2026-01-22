# gitty

**gitty** is a beautiful, modern, and fast Git CLI tool built with Go and the [Charm](https://charm.sh) ecosystem (BubbleTea, Lip Gloss). It provides a simplified interface for common git operations, featuring AI-powered commit messages and a stunning terminal UI.

## Features

- **Beautiful UI**: Styled with Lip Gloss for a modern terminal aesthetic (Pink/Purple/Blue theme).
- **Fast & Responsive**: Optimized for speed, with instant status checks and background processing.
- **AI Commit Messages**: Generate context-aware commit messages using OpenAI or Anthropic (Claude).
- **Quick Actions**: Stage, commit, push, pull, and reset with single keystrokes.
- **GitHub Publishing**: Create and push new repositories to GitHub directly from the CLI.
- **Configurable**: YAML configuration for AI settings, user details, and UI preferences.

## Installation

### From Source

```bash
git clone https://github.com/0mykull/gitty.git
cd gitty
make install
```

This will build and install the binary to `~/.local/bin/gitty`. Ensure this directory is in your `$PATH`.

## Usage

Run `gitty` in any git repository:

```bash
gitty
```

### Key Bindings

| Key | Action | Description |
|-----|--------|-------------|
| `a` | **Stage All** | `git add .` |
| `c` | **Commit** | Open manual commit interface |
| `i` | **AI Commit** | Generate commit message with AI |
| `p` | **Push** | `git push` |
| `l` | **Pull** | `git pull` |
| `r` | **Reset** | Hard reset changes (requires confirmation) |
| `u` | **Publish** | Create & push repo to GitHub |
| `o` | **Open Repo** | Open repository in browser |
| `g` | **Lazygit** | Launch lazygit (if installed) |
| `b` | **Branches** | View branches |
| `q` | **Quit** | Exit gitty |

## Configuration

Gitty uses a YAML configuration file located at `~/.config/gitty/config.yaml`.

Run `make config` to generate a default configuration file, or create it manually:

```yaml
# ~/.config/gitty/config.yaml

git:
  editor: "vim"

ai:
  provider: "openai" # or "anthropic"
  model: "gpt-4o-mini"
  api_key: "your-api-key-here" # or use OPENAI_API_KEY env var
  temperature: 0.7

ui:
  show_icons: true

github:
  default_visibility: "public"
```

## License

MIT

---

Built with ❤️ by [0mykull](https://github.com/0mykull)
