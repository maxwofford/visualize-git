package tree

import (
	"fmt"
	"sort"
	"strings"
	"tree-it/types"
)

func CreateOrUpdateNode(nodeMap map[string]*types.FileNode, path string, timestamp int64) {
	// Create or update the node itself
	if node, exists := nodeMap[path]; exists {
		node.LastModified = timestamp
		node.DeletedAt = nil // Clear deletion if file is recreated
	} else {
		nodeMap[path] = &types.FileNode{
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
			nodeMap[dirPath] = &types.FileNode{
				Path:         dirPath,
				Type:        "directory",
				CreatedAt:    timestamp,
				LastModified: timestamp,
				Children:    make([]types.FileNode, 0),
			}
		}
	}
}

func buildRootNodes(nodeMap map[string]*types.FileNode) []types.FileNode {
	root := make([]types.FileNode, 0)
	
	// First pass: collect all root nodes
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
	
	// Second pass: build parent-child relationships
	for path, node := range nodeMap {
		if node.Type == "directory" {
			// Clear existing children to rebuild
			node.Children = make([]types.FileNode, 0)
			
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
	
	// Sort root nodes alphabetically
	sort.Slice(root, func(i, j int) bool {
		return root[i].Path < root[j].Path
	})
	
	return root
}

func getRepoNameFromURL(url string) string {
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return parts[len(parts)-1]
	}
	return fmt.Sprintf("%s_%s", parts[len(parts)-2], parts[len(parts)-1])
}
