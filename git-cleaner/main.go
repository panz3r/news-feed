package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	// Error out if fetching feeds takes longer than a minute
	timeout = time.Minute
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// Load local repository
	repo, err := git.PlainOpen("../")
	if err != nil {
		return err
	}

	// retrieves the branch pointed by HEAD
	ref, err := repo.Head()
	if err != nil {
		return err
	}

	// retrieves the commit history
	cIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return err
	}

	// ref to the oldest auto-commit
	var oldestAutoCommitRef *plumbing.Hash = nil

	// iterates over the commits, looking for the oldest auto-commit
	commit, cErr := cIter.Next()
	for commit != nil {
		fmt.Println("ref", commit.Hash, "message", commit.Message)

		oldestAutoCommitRef = &commit.Hash

		if !strings.EqualFold(strings.TrimSpace(commit.Message), "chore(news): Update news (automated)") {
			break
		}

		commit, cErr = cIter.Next()
	}
	if cErr != nil {
		return err
	}

	if oldestAutoCommitRef == nil {
		return errors.New("no auto-commit found")
	}

	fmt.Println("Oldest auto-commit hash:", oldestAutoCommitRef)

	// retrieve work tree for the repo
	workTree, err := repo.Worktree()
	if err != nil {
		return err
	}

	// try to soft-reset the repository to the oldest auto-commit
	err = workTree.Reset(&git.ResetOptions{
		Commit: *oldestAutoCommitRef,
		Mode:   git.SoftReset,
	})
	if err != nil {
		return err
	}

	// commit everything as a new auto-commit
	hash, err := workTree.Commit("chore(news): Update news (automated)", &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  "GitHub Actions",
			Email: "actions@github.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	fmt.Println("new auto-commit hash", hash)

	return nil
}
