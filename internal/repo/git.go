package repo

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type GitRunner interface {
	Clone(url, dest string, depth int) error
	Pull(repoPath string) error
	GetRemoteURL(repoPath string) (string, error)
	GetCurrentCommit(repoPath string) (string, error)
	Checkout(repoPath, ref string) error
	VerifyRepoAccess(url string) error
}

type DefaultGitRunner struct{}

func NewGitRunner() GitRunner {
	return &DefaultGitRunner{}
}

func (g *DefaultGitRunner) Clone(url, dest string, depth int) error {
	args := []string{"clone"}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}
	args = append(args, url, dest)

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %s: %w", stderr.String(), err)
	}
	return nil
}

func (g *DefaultGitRunner) Pull(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "pull", "--ff-only")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %s: %w", stderr.String(), err)
	}
	return nil
}

func (g *DefaultGitRunner) GetRemoteURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git remote get-url failed: %s: %w", stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (g *DefaultGitRunner) GetCurrentCommit(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse failed: %s: %w", stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (g *DefaultGitRunner) Checkout(repoPath, ref string) error {
	cmd := exec.Command("git", "-C", repoPath, "checkout", ref)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %s: %w", stderr.String(), err)
	}
	return nil
}

// VerifyRepoAccess checks if a repository URL is accessible (public or accessible with current credentials)
func (g *DefaultGitRunner) VerifyRepoAccess(url string) error {
	cmd := exec.Command("git", "ls-remote", "--exit-code", "--heads", url)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := stderr.String()
		if strings.Contains(errStr, "Repository not found") || strings.Contains(errStr, "does not exist") {
			return fmt.Errorf("repository not found or not accessible: %s", url)
		}
		if strings.Contains(errStr, "Authentication failed") || strings.Contains(errStr, "403") {
			return fmt.Errorf("repository requires authentication: %s", url)
		}
		return fmt.Errorf("failed to access repository: %s: %w", errStr, err)
	}
	return nil
}
