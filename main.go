package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "time"
    "runtime/pprof"
    "testing"

    "github.com/go-git/go-git/v5"
)

type Author struct {
    Name        string `json:"name"`
    Email       string `json:"email"`
    FirstCommit int64  `json:"firstCommit"`
    LastCommit  int64  `json:"lastCommit"`
    TotalCommits int   `json:"totalCommits"`
}

type Metadata struct {
    RepoName       string   `json:"repoName"`
    RepoURL        string   `json:"repoUrl"`
    FirstCommitDate int64   `json:"firstCommitDate"`
    LastCommitDate  int64   `json:"lastCommitDate"`
    TotalCommits    int     `json:"totalCommits"`
    Authors         []Author `json:"authors"`
}

type FileAction struct {
    Type      string `json:"type"` // "A", "M", or "D"
    Path      string `json:"path"`
    Timestamp int64  `json:"timestamp"`
    CommitHash string `json:"commitHash"`
    Author    string `json:"author"`
}

type FileNode struct {
    Path        string     `json:"path"`
    Type        string     `json:"type"` // "file" or "directory"
    Children    []FileNode `json:"children,omitempty"`
    LastModified int64     `json:"lastModified"`
    CreatedAt    int64     `json:"createdAt"`
    DeletedAt    *int64    `json:"deletedAt,omitempty"`
}

type RepoData struct {
    Metadata    Metadata     `json:"metadata"`
    FileActions []FileAction `json:"fileActions"`
    FinalTree   []FileNode   `json:"finalTree"`
}

func main() {
    // Start CPU profiling
    f, err := os.Create("cpu.prof")
    if err != nil {
        fmt.Printf("Could not create CPU profile: %v\n", err)
        os.Exit(1)
    }
    defer f.Close()
    if err := pprof.StartCPUProfile(f); err != nil {
        fmt.Printf("Could not start CPU profile: %v\n", err)
        os.Exit(1)
    }
    defer pprof.StopCPUProfile()

    startTime := time.Now()
    
    if len(os.Args) != 2 {
        fmt.Println("Usage: go run main.go <repo_url>")
        os.Exit(1)
    }

    repoURL := os.Args[1]
    // Create output filename in outputs directory
    outputFile := fmt.Sprintf("outputs/%s.json", getRepoNameFromURL(repoURL))

    // Create outputs directory if it doesn't exist
    err = os.MkdirAll("outputs", 0755)
    if err != nil {
        fmt.Printf("Failed to create outputs directory: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Processing repository %s...\n", repoURL)
    cloneStart := time.Now()
    
    // Get or clone repository
    repo, err := getOrCloneRepo(repoURL)
    if err != nil {
        fmt.Printf("Repository error: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("Repository ready in %v\n", time.Since(cloneStart))

    fmt.Println("Processing repository...")
    processStart := time.Now()
    
    data := processRepo(repo, repoURL)
    
    fmt.Printf("Processing completed in %v\n", time.Since(processStart))
    
    fmt.Println("Writing output file...")
    writeStart := time.Now()
    
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        fmt.Printf("Failed to marshal JSON: %v\n", err)
        os.Exit(1)
    }

    err = os.WriteFile(outputFile, jsonData, 0644)
    if err != nil {
        fmt.Printf("Failed to write output file: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Write completed in %v\n", time.Since(writeStart))
    fmt.Printf("Total time: %v\n", time.Since(startTime))

    // Memory profile at the end
    f2, err := os.Create("mem.prof")
    if err != nil {
        fmt.Printf("Could not create memory profile: %v\n", err)
        os.Exit(1)
    }
    defer f2.Close()
    if err := pprof.WriteHeapProfile(f2); err != nil {
        fmt.Printf("Could not write memory profile: %v\n", err)
        os.Exit(1)
    }

    timeProcessingCommits := time.Now()
    // process commits...
    fmt.Printf("Commit processing took: %v\n", time.Since(timeProcessingCommits))

    timeBuildingTree := time.Now()
    // build tree...
    fmt.Printf("Tree building took: %v\n", time.Since(timeBuildingTree))
}

func processRepo(repo *git.Repository, repoURL string) RepoData {
    data := RepoData{}
    
    // Get the worktree to run git commands
    w, err := repo.Worktree()
    if err != nil {
        fmt.Printf("Failed to get worktree: %v\n", err)
        os.Exit(1)
    }

    // Run the git log command
    cmd := exec.Command("git", "log", "--pretty=format:user:%aN%n%ct", "--reverse", "--raw", "--encoding=UTF-8", "--no-renames", "--no-show-signature")
    cmd.Dir = w.Filesystem.Root()
    
    output, err := cmd.Output()
    if err != nil {
        fmt.Printf("Failed to run git log: %v\n", err)
        os.Exit(1)
    }

    // Process the output
    authors := make(map[string]*Author)
    var firstCommitDate, lastCommitDate int64
    totalCommits := 0
    fileActions := []FileAction{}

    scanner := bufio.NewScanner(bytes.NewReader(output))
    var currentAuthor string
    var currentTimestamp int64

    for scanner.Scan() {
        line := scanner.Text()
        
        if strings.HasPrefix(line, "user:") {
            currentAuthor = strings.TrimPrefix(line, "user:")
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
            
            fileActions = append(fileActions, FileAction{
                Type:      actionType,
                Path:      path,
                Timestamp: currentTimestamp,
                Author:    currentAuthor,
            })
            continue
        }
        
        // Must be a timestamp line
        timestamp, err := strconv.ParseInt(line, 10, 64)
        if err != nil {
            continue
        }
        currentTimestamp = timestamp
        
        // Update time range
        if firstCommitDate == 0 || timestamp < firstCommitDate {
            firstCommitDate = timestamp
        }
        if timestamp > lastCommitDate {
            lastCommitDate = timestamp
        }
        
        // Update author stats
        author, exists := authors[currentAuthor]
        if !exists {
            author = &Author{
                Name:         currentAuthor,
                FirstCommit:  timestamp,
                LastCommit:   timestamp,
                TotalCommits: 0,
            }
            authors[currentAuthor] = author
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
    authorsList := make([]Author, 0, len(authors))
    for _, author := range authors {
        authorsList = append(authorsList, *author)
    }

    // Build metadata
    data.Metadata = Metadata{
        RepoName:        getRepoNameFromURL(repoURL),
        RepoURL:         repoURL,
        FirstCommitDate: firstCommitDate,
        LastCommitDate:  lastCommitDate,
        TotalCommits:    totalCommits,
        Authors:         authorsList,
    }

    data.FileActions = fileActions
    data.FinalTree = buildFinalTree(fileActions)

    return data
}

func buildFinalTree(actions []FileAction) []FileNode {
    // Map to track all files/directories and their current state
    nodeMap := make(map[string]*FileNode)
    
    // Process actions in chronological order (reverse our list since it's newest first)
    for i := len(actions) - 1; i >= 0; i-- {
        action := actions[i]
        path := action.Path
        
        // Skip only .git directory
        if strings.HasPrefix(path, ".git/") {
            continue
        }
        
        // Handle the action based on type
        switch action.Type {
        case "A", "M":
            // Create or update file node
            createOrUpdateNode(nodeMap, path, action.Timestamp)
        case "D":
            // Mark file as deleted
            if node, exists := nodeMap[path]; exists {
                timestamp := action.Timestamp
                node.DeletedAt = &timestamp
            }
        }
    }
    
    // Build the tree structure
    root := make([]FileNode, 0)
    for path, node := range nodeMap {
        // Skip deleted files
        if node.DeletedAt != nil {
            continue
        }
        
        // Skip if this is not a top-level node
        if strings.Contains(path, "/") {
            continue
        }
        
        // Add to root if it's a top-level node
        root = append(root, *node)
    }
    
    // Sort root nodes alphabetically
    sort.Slice(root, func(i, j int) bool {
        return root[i].Path < root[j].Path
    })
    
    return root
}

func createOrUpdateNode(nodeMap map[string]*FileNode, path string, timestamp int64) {
    // Create or update the node itself
    if node, exists := nodeMap[path]; exists {
        node.LastModified = timestamp
        node.DeletedAt = nil // Clear deletion if file is recreated
    } else {
        nodeMap[path] = &FileNode{
            Path:         path,
            Type:        "file",
            CreatedAt:    timestamp,
            LastModified: timestamp,
        }
    }
    
    // Create all parent directories
    parts := strings.Split(path, "/")
    for i := 0; i < len(parts)-1; i++ {
        dirPath := strings.Join(parts[:i+1], "/")
        
        if dir, exists := nodeMap[dirPath]; exists {
            // Update existing directory's last modified time
            dir.LastModified = timestamp
            dir.DeletedAt = nil // Clear deletion if directory is recreated
        } else {
            // Create new directory
            nodeMap[dirPath] = &FileNode{
                Path:         dirPath,
                Type:        "directory",
                CreatedAt:    timestamp,
                LastModified: timestamp,
                Children:    make([]FileNode, 0),
            }
        }
    }
    
    // Build parent-child relationships
    for path, node := range nodeMap {
        if node.Type == "directory" {
            // Clear existing children to rebuild
            node.Children = make([]FileNode, 0)
            
            // Find all immediate children
            prefix := path + "/"
            for childPath, childNode := range nodeMap {
                if strings.HasPrefix(childPath, prefix) && 
                   !strings.Contains(childPath[len(prefix):], "/") &&
                   childNode.DeletedAt == nil {
                    node.Children = append(node.Children, *childNode)
                }
            }
            
            // Sort children alphabetically
            sort.Slice(node.Children, func(i, j int) bool {
                return node.Children[i].Path < node.Children[j].Path
            })
        }
    }
}

func getRepoNameFromURL(url string) string {
    // Remove .git suffix if present
    url = strings.TrimSuffix(url, ".git")
    
    // Split by '/' and get the last two parts (owner/repo)
    parts := strings.Split(url, "/")
    if len(parts) < 2 {
        return parts[len(parts)-1]
    }
    
    // Return "owner_repo"
    return fmt.Sprintf("%s_%s", parts[len(parts)-2], parts[len(parts)-1])
}

func getOrCloneRepo(repoURL string) (*git.Repository, error) {
    // Create cache directory if it doesn't exist
    cacheDir := "repos-cache"
    err := os.MkdirAll(cacheDir, 0755)
    if err != nil {
        return nil, fmt.Errorf("failed to create cache directory: %v", err)
    }

    repoName := getRepoNameFromURL(repoURL)
    repoCachePath := filepath.Join(cacheDir, repoName)

    // Check if repo exists in cache
    if _, err := os.Stat(repoCachePath); err == nil {
        fmt.Printf("Found cached repo at %s, updating...\n", repoCachePath)
        
        // Open existing repo
        repo, err := git.PlainOpen(repoCachePath)
        if err != nil {
            return nil, fmt.Errorf("failed to open cached repo: %v", err)
        }

        // Get the worktree
        w, err := repo.Worktree()
        if err != nil {
            return nil, fmt.Errorf("failed to get worktree: %v", err)
        }

        // Pull latest changes
        err = w.Pull(&git.PullOptions{
            RemoteName: "origin",
        })
        if err != nil && err != git.NoErrAlreadyUpToDate {
            return nil, fmt.Errorf("failed to pull latest changes: %v", err)
        }

        return repo, nil
    }

    // Clone fresh if not in cache
    fmt.Printf("Cloning %s into %s...\n", repoURL, repoCachePath)
    repo, err := git.PlainClone(repoCachePath, false, &git.CloneOptions{
        URL: repoURL,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to clone repository: %v", err)
    }

    return repo, nil
}

func BenchmarkProcessRepo(b *testing.B) {
    repo, _ := getOrCloneRepo("https://github.com/hackclub/airbridge")
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        processRepo(repo, "https://github.com/hackclub/airbridge")
    }
}