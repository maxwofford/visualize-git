package git

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"tree-it/tree"
	"tree-it/types"
	"tree-it/utils"

	"github.com/go-git/go-git/v5"
)

func ProcessRepo(repo *git.Repository, repoURL string) (types.RepoData, error) {
	data := types.RepoData{}

	w, err := repo.Worktree()
	if err != nil {
		return data, err
	}

	cmd := exec.Command("git", "log", "--pretty=format:hash:%H%nmessage:%s%nuser:%aN%n%ct", "--reverse", "--raw", "--encoding=UTF-8", "--no-renames", "--no-show-signature")
	cmd.Dir = w.Filesystem.Root()

	output, err := cmd.Output()
	if err != nil {
		return data, err
	}

	authors := make(map[string]*types.Author)
	var firstCommitDate, lastCommitDate int64
	totalCommits := 0
	fileActions := []types.FileAction{}
	var currentCommit struct {
		hash      string
		message   string
		author    string
		timestamp int64
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "hash:") {
			currentCommit.hash = strings.TrimPrefix(line, "hash:")
			continue
		}

		if strings.HasPrefix(line, "message:") {
			currentCommit.message = strings.TrimPrefix(line, "message:")
			continue
		}

		if strings.HasPrefix(line, "user:") {
			currentCommit.author = strings.TrimPrefix(line, "user:")
			totalCommits++
			continue
		}

		if strings.HasPrefix(line, ":") {
			// Parse file change line
			parts := strings.Fields(line)
			if len(parts) < 6 {
				continue
			}

			changeType := parts[4]
			path := parts[5]

			// Skip .git directory
			if strings.HasPrefix(path, ".git/") {
				continue
			}

			var actionType string
			switch changeType {
			case "A":
				actionType = "A"
			case "M":
				actionType = "M"
			case "D":
				actionType = "D"
			default:
				continue
			}

			fileActions = append(fileActions, types.FileAction{
				Type:          actionType,
				Path:          path,
				Timestamp:     currentCommit.timestamp,
				Author:        currentCommit.author,
				CommitHash:    currentCommit.hash,
				CommitMessage: currentCommit.message,
			})
			continue
		}

		// Must be a timestamp line
		timestamp, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			continue
		}
		currentCommit.timestamp = timestamp

		// Update time range
		if firstCommitDate == 0 || timestamp < firstCommitDate {
			firstCommitDate = timestamp
		}
		if timestamp > lastCommitDate {
			lastCommitDate = timestamp
		}

		// Update author stats
		author, exists := authors[currentCommit.author]
		if !exists {
			author = &types.Author{
				Name:         currentCommit.author,
				FirstCommit:  timestamp,
				LastCommit:   timestamp,
				TotalCommits: 0,
			}
			authors[currentCommit.author] = author
		}
		author.TotalCommits++

		if timestamp < author.FirstCommit {
			author.FirstCommit = timestamp
		}
		if timestamp > author.LastCommit {
			author.LastCommit = timestamp
		}
	}

	// Convert authors map to slice
	authorsList := make([]types.Author, 0, len(authors))
	for _, author := range authors {
		authorsList = append(authorsList, *author)
	}

	// Build metadata
	data.Metadata = types.Metadata{
		RepoName:        utils.GetRepoNameFromURL(repoURL),
		RepoURL:         repoURL,
		FirstCommitDate: firstCommitDate,
		LastCommitDate:  lastCommitDate,
		TotalCommits:    totalCommits,
		Authors:         authorsList,
	}

	data.FileActions = fileActions
	data.FinalTree = tree.BuildFinalTree(fileActions)

	return data, nil
}
