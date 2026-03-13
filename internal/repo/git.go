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
