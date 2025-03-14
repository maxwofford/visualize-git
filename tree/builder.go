package tree

import (
	"strings"
	"tree-it/types"
)

func BuildFinalTree(actions []types.FileAction) []types.FileNode {
	nodeMap := make(map[string]*types.FileNode)
	
	for i := len(actions) - 1; i >= 0; i-- {
		action := actions[i]
		if strings.HasPrefix(action.Path, ".git/") {
			continue
		}
		
		switch action.Type {
		case "A", "M":
			CreateOrUpdateNode(nodeMap, action.Path, action.Timestamp)
		case "D":
			if node, exists := nodeMap[action.Path]; exists {
				timestamp := action.Timestamp
				node.DeletedAt = &timestamp
			}
		}
	}
	
	return buildRootNodes(nodeMap)
}
