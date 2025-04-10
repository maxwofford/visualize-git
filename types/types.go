package types

type Author struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	FirstCommit  int64  `json:"firstCommit"`
	LastCommit   int64  `json:"lastCommit"`
	TotalCommits int    `json:"totalCommits"`
}

type Metadata struct {
	RepoName        string   `json:"repoName"`
	RepoURL         string   `json:"repoUrl"`
	FirstCommitDate int64    `json:"firstCommitDate"`
	LastCommitDate  int64    `json:"lastCommitDate"`
	TotalCommits    int      `json:"totalCommits"`
	Authors         []Author `json:"authors"`
}

type FileAction struct {
	Type          string `json:"type"` // "A", "M", or "D"
	Path          string `json:"path"`
	Timestamp     int64  `json:"timestamp"`
	Author        string `json:"author"`
	CommitHash    string `json:"commitHash"`
	CommitMessage string `json:"commitMessage"`
}

type FileNode struct {
	Path         string     `json:"path"`
	Type         string     `json:"type"` // "file" or "directory"
	Children     []FileNode `json:"children,omitempty"`
	LastModified int64      `json:"lastModified"`
	CreatedAt    int64      `json:"createdAt"`
	DeletedAt    *int64     `json:"deletedAt,omitempty"`
}

type RepoData struct {
	Metadata    Metadata     `json:"metadata"`
	FileActions []FileAction `json:"fileActions"`
	FinalTree   []FileNode   `json:"finalTree"`
}
