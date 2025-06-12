// Provides additional Git actions
//
// The main focus is to work with a Git Repository by SSH authentication.
//
// The Git Actions module is used in following example repository: https://github.com/puzzle/dagger-module-gitops/

package main

import (
	"context"
	"dagger/git-actions/internal/dagger"
	"fmt"
	"time"
)

const WorkDir = "/tmp/repo/"

type GitActions struct {
}

type GitActionRepository struct {
	// URL of the Git repository
	RepoUrl string
	// SSH key with access credentials for the Git repository
	SshKey *dagger.File
}

// Configure Git repository access with ssh key
func (m *GitActions) WithRepository(
	// method call context
	ctx context.Context,
	// URL of the Git repository
	repoUrl string,
	// SSH key with access credentials for the Git repository
	sshKey *dagger.File,
) *GitActionRepository {
	return &GitActionRepository{
		RepoUrl: repoUrl,
		SshKey:  sshKey,
	}
}

// Clone Git repository using the SSH Key.
func (m *GitActionRepository) CloneSsh(
	// method call context
	ctx context.Context,
) (*dagger.Directory, error) {

	if m.RepoUrl == "" || m.SshKey == nil {
		return nil, fmt.Errorf("Repo URL and SSH Key must be set")
	}

	c, err := prepareContainer(m.SshKey, "", "").
		WithExec([]string{"git", "clone", m.RepoUrl, "."}).
		Sync(ctx)

	if err != nil {
		return nil, err
	}

	dir := c.Directory(WorkDir)

	return dir, nil
}

// Commit local changes to the Git repository using the SSH Key.
func (m *GitActionRepository) Push(
	// method call context
	ctx context.Context,
	// local dir with the Git repository and the changes
	dir *dagger.Directory,
	// Git branch to push to.
	// +optional
	// +default="main"
	prBranch string,
	// Commit message
	// +optional
	// +default="autocommit"
	commitMessage string,
	// Git user name
	// +optional
	// +default="dagger-bot"
	userName string,
	// Git user email
	// +optional
	// +default="cicd@puzzle.ch"
	userEmail string,
) error {

	c := prepareContainer(m.SshKey, userName, userEmail).
		WithDirectory(WorkDir, dir)

	if prBranch != "" {
		c = c.WithExec([]string{"git", "switch", "-c", prBranch})
	}

	_, err := c.WithExec([]string{"git", "add", "."}).
		WithExec([]string{"git", "commit", "-m", commitMessage}).
		WithExec([]string{"git", "push"}).
		Sync(ctx)

	return err
}

func prepareContainer(
	key *dagger.File,
	// +optional
	// +default="dagger-bot"
	userName string,
	// +optional
	// +default="cicd@puzzle.ch"
	userEmail string,
) *dagger.Container {
	return dag.Container().
		From("registry.puzzle.ch/cicd/alpine-base:latest").
		WithWorkdir(WorkDir).
		WithFile("/tmp/.ssh/id", key, dagger.ContainerWithFileOpts{Permissions: 0400}).
		WithEnvVariable("GIT_SSH_COMMAND", "ssh -i /tmp/.ssh/id -o StrictHostKeyChecking=no").
		WithEnvVariable("CACHE_BUSTER", time.Now().String()).
		WithExec([]string{"git", "config", "--global", "user.name", userName}).
		WithExec([]string{"git", "config", "--global", "user.email", userEmail}).
		WithExec([]string{"git", "config", "--global", "--add", "--bool", "push.autoSetupRemote", "true"})
}
