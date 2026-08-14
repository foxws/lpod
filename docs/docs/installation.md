---
sidebar_position: 2
---

# Installation

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/latest/download/lpod
chmod +x ~/.local/bin/lpod
```

Pin a specific release instead of `latest` for reproducibility:

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/download/v0.1.0/lpod
chmod +x ~/.local/bin/lpod
```

Or clone the repo and copy/symlink `lpod` onto your `PATH` yourself — note this gets the unreleased `main` branch, whose `lpod --version` reports `dev` rather than a tagged version. It's a single dependency-free script — no PHP or Composer required.

## Optional: shell alias

Resolves `lpod` relative to your current directory, so one alias works across every project:

```sh
# Bash/Zsh, in ~/.bashrc or ~/.zshrc
alias lpod='[ -f lpod ] && bash lpod || bash "$(git rev-parse --show-toplevel)/lpod"'
```
