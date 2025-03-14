package main

import (
    "fmt"
    "html/template"
    "log"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "github.com/gorilla/mux"
    "tree-it/git"
    "tree-it/types"
)

type ProcessStatus string

const (
    StatusQueued     ProcessStatus = "queued"
    StatusCloning    ProcessStatus = "cloning"
    StatusProcessing ProcessStatus = "processing"
    StatusComplete   ProcessStatus = "complete"
    StatusError      ProcessStatus = "error"
)

type RepoProcess struct {
    Status      ProcessStatus
    StartedAt   time.Time
    Subscribers []*websocket.Conn
    Done        chan bool
    Data        *types.RepoData
    Error       error
    mu          sync.Mutex
}

type ProcessManager struct {
    processes map[string]*RepoProcess
    mu        sync.RWMutex
}

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins for development
    },
}

func NewProcessManager() *ProcessManager {
    return &ProcessManager{
        processes: make(map[string]*RepoProcess),
    }
}

func (pm *ProcessManager) GetOrCreateProcess(repoKey string) *RepoProcess {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    if process, exists := pm.processes[repoKey]; exists {
        return process
    }

    process := &RepoProcess{
        Status:    StatusQueued,
        StartedAt: time.Now(),
        Done:      make(chan bool),
    }
    pm.processes[repoKey] = process
    return process
}

func (p *RepoProcess) AddSubscriber(conn *websocket.Conn) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.Subscribers = append(p.Subscribers, conn)
}

func (p *RepoProcess) RemoveSubscriber(conn *websocket.Conn) {
    p.mu.Lock()
    defer p.mu.Unlock()
    for i, sub := range p.Subscribers {
        if sub == conn {
            p.Subscribers = append(p.Subscribers[:i], p.Subscribers[i+1:]...)
            break
        }
    }
}

func (p *RepoProcess) BroadcastStatus() {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    message := map[string]interface{}{
        "status": p.Status,
        "time":   time.Since(p.StartedAt).Seconds(),
    }

    if p.Error != nil {
        message["error"] = p.Error.Error()
    }

    if p.Status == StatusComplete && p.Data != nil {
        message["data"] = p.Data
    }

    for _, conn := range p.Subscribers {
        conn.WriteJSON(message)
    }
}

type TemplateData struct {
    Org     string
    Name    string
    RepoKey string
}

func main() {
    pm := NewProcessManager()
    r := mux.NewRouter()

    // Parse templates
    tmpl, err := template.ParseFiles("templates/index.html")
    if err != nil {
        log.Fatal("Failed to parse templates:", err)
    }

    // Redirect /:org/:name to /github/:org/:name
    r.HandleFunc("/{org}/{name}", func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        http.Redirect(w, r, fmt.Sprintf("/github/%s/%s", vars["org"], vars["name"]), http.StatusFound)
    })

    // Main handler for GitHub repos
    r.HandleFunc("/github/{org}/{name}", func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        repoKey := fmt.Sprintf("github/%s/%s", vars["org"], vars["name"])
        
        data := TemplateData{
            Org:     vars["org"],
            Name:    vars["name"],
            RepoKey: repoKey,
        }

        if err := tmpl.Execute(w, data); err != nil {
            log.Printf("Template error: %v", err)
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        }
    })

    // WebSocket handler
    r.HandleFunc("/ws/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
        repoKey := mux.Vars(r)["path"]
        process := pm.GetOrCreateProcess(repoKey)

        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Printf("WebSocket upgrade error: %v", err)
            return
        }
        defer conn.Close()

        process.AddSubscriber(conn)
        defer process.RemoveSubscriber(conn)

        // Start processing if this is the first subscriber
        if len(process.Subscribers) == 1 {
            go func() {
                process.Status = StatusCloning
                process.BroadcastStatus()

                repoURL := fmt.Sprintf("https://github.com/%s", repoKey[7:]) // Remove "github/" prefix
                repo, err := git.GetOrCloneRepo(repoURL)
                if err != nil {
                    process.Status = StatusError
                    process.Error = err
                    process.BroadcastStatus()
                    return
                }

                process.Status = StatusProcessing
                process.BroadcastStatus()

                data, err := git.ProcessRepo(repo, repoURL)
                if err != nil {
                    process.Status = StatusError
                    process.Error = err
                    process.BroadcastStatus()
                    return
                }

                process.Status = StatusComplete
                process.Data = &data
                process.BroadcastStatus()
            }()
        }

        // Keep connection alive and handle disconnection
        for {
            _, _, err := conn.ReadMessage()
            if err != nil {
                break
            }
        }
    })

    log.Println("Server starting on :3000")
    log.Fatal(http.ListenAndServe(":3000", r))
} 