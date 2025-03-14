package git

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"tree-it/utils"
)

func GetOrCloneRepo(repoURL string) (*git.Repository, error) {
	cacheDir := "repos-cache"
	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %v", err)
	}

	repoName := utils.GetRepoNameFromURL(repoURL)
	repoCachePath := filepath.Join(cacheDir, repoName)

	if _, err := os.Stat(repoCachePath); err == nil {
		fmt.Printf("Found cached repo at %s, updating...\n", repoCachePath)
		
		repo, err := git.PlainOpen(repoCachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open cached repo: %v", err)
		}

		w, err := repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("failed to get worktree: %v", err)
		}

		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, fmt.Errorf("failed to pull latest changes: %v", err)
		}

		return repo, nil
	}

	fmt.Printf("Cloning %s into %s...\n", repoURL, repoCachePath)
	return git.PlainClone(repoCachePath, false, &git.CloneOptions{URL: repoURL})
}
