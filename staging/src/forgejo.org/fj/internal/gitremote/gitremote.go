// Package gitremote parses git config to discover the Forgejo remote
// for the current working directory.
package gitremote

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Remote holds the parsed host + owner/repo from a git remote URL.
type Remote struct {
	Host string
	Owner string
	Repo  string
}

// Discover finds the Forgejo remote from .git/config in the current or
// parent directories. It checks the named remote first (if given), then
// falls back to "origin", then the first remote that looks like a forge.
func Discover(namedRemote string) (*Remote, error) {
	gitDir, err := findGitDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(gitDir, "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading git config: %w", err)
	}
	remotes := parseGitConfig(string(data))
	if len(remotes) == 0 {
		return nil, fmt.Errorf("no git remotes found")
	}

	// Try named remote, then origin, then first available
	remoteOrder := []string{namedRemote, "origin", ""}
	for _, name := range remoteOrder {
		if name != "" {
			if url, ok := remotes[name]; ok {
				return parseRemoteURL(url)
			}
		}
	}
	// Fall back to first remote
	for _, url := range remotes {
		return parseRemoteURL(url)
	}
	return nil, fmt.Errorf("no suitable git remote found")
}

func findGitDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return gitDir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a git repository")
		}
		dir = parent
	}
}

func parseGitConfig(config string) map[string]string {
	remotes := map[string]string{}
	var currentRemote string
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[remote ") {
			// [remote "origin"]
			name := strings.TrimPrefix(line, "[remote ")
			name = strings.TrimSuffix(name, "]")
			name = strings.Trim(name, "\" ")
			currentRemote = name
		} else if currentRemote != "" && strings.HasPrefix(line, "url = ") {
			url := strings.TrimPrefix(line, "url = ")
			remotes[currentRemote] = url
			currentRemote = ""
		}
	}
	return remotes
}

func parseRemoteURL(raw string) (*Remote, error) {
	// SSH: git@host:owner/repo.git
	if strings.HasPrefix(raw, "git@") || strings.Contains(raw, ":") && !strings.Contains(raw, "://") {
		// git@git.example.com:owner/repo.git
		rest := strings.SplitN(raw, ":", 2)
		if len(rest) != 2 {
			return nil, fmt.Errorf("invalid SSH remote URL: %s", raw)
		}
		hostPart := strings.TrimPrefix(rest[0], "git@")
		path := strings.TrimSuffix(rest[1], ".git")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid remote path: %s", path)
		}
		return &Remote{Host: hostPart, Owner: parts[0], Repo: parts[1]}, nil
	}
	// HTTPS: https://host/owner/repo.git
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		// Strip scheme
		rest := strings.SplitN(raw, "://", 2)[1]
		// Split host and path
		idx := strings.Index(rest, "/")
		if idx < 0 {
			return nil, fmt.Errorf("invalid HTTPS remote URL: %s", raw)
		}
		host := rest[:idx]
		path := strings.TrimSuffix(rest[idx+1:], ".git")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid remote path: %s", path)
		}
		return &Remote{Host: host, Owner: parts[0], Repo: parts[1]}, nil
	}
	return nil, fmt.Errorf("unsupported remote URL format: %s", raw)
}
