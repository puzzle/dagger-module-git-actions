package main

import (
	"context"
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
	SshKey  *File
}

// Configure Git repository access with ssh key
func (m *GitActions) WithRepository(
	// method call context
	ctx context.Context,

	// URL of the Git repository
	repoUrl string,

	// SSH key with access credentials for the Git repository
	sshKey *File,
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
) (*Directory, error) {

	if m.RepoUrl == "" || m.SshKey == nil {
		return nil, fmt.Errorf("Repo URL and SSH Key must be set")
	}

	c, err := prepareContainer(m.SshKey).
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
	dir *Directory,

	// Git branch to push to.
	// +optional
	// +default='main'
	prBranch Optional[string],
) error {

	c := prepareContainer(m.SshKey).
		WithDirectory(WorkDir, dir)

	if prBranch.isSet {
		c = c.WithExec([]string{"git", "switch", "-c", prBranch.GetOr("main")})
	}

	_, err := c.WithExec([]string{"git", "add", "."}).
		WithExec([]string{"git", "commit", "-m", "autocommit"}).
		WithExec([]string{"git", "push"}).
		Sync(ctx)

	return err
}

func prepareContainer(key *File) *Container {
	return dag.Container().
		Pipeline("Prepare Git container").
		From("registry.puzzle.ch/cicd/alpine-base:latest").
		WithWorkdir(WorkDir).
		WithFile("/tmp/.ssh/id", key, ContainerWithFileOpts{Permissions: 0400}).
		WithEnvVariable("GIT_SSH_COMMAND", "ssh -i /tmp/.ssh/id -o StrictHostKeyChecking=no").
		WithEnvVariable("CACHE_BUSTER", time.Now().String()).
		WithExec([]string{"git", "config", "--global", "user.name", "dagger-bot"}).
		WithExec([]string{"git", "config", "--global", "user.email", "cicd@puzzle.ch"}).
		WithExec([]string{"git", "config", "--global", "--add", "--bool", "push.autoSetupRemote", "true"})
}
