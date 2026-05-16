package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inhuman/memorialiste/codesearch"
	"github.com/inhuman/memorialiste/provider"
)

// SearchCodeSchema is the tool descriptor for the search_code function
// exposed to the LLM during a tool-call loop.
var SearchCodeSchema = provider.ToolSchema{
	Name:        "search_code",
	Description: "Search the repository for Go declarations by regex name match. Returns matched function, method, type, const, or var declarations with their source code, file path, and line ranges.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regex pattern matched against identifier names (Go regexp syntax).",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Repo-relative path scope. Empty or '.' searches the whole repo.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of hits to return. Defaults to 20.",
			},
		},
		"required": []string{"pattern"},
	},
}

// dispatchSearchCode parses tool arguments, runs codesearch.Search, and
// returns a JSON-encoded result ready to send back to the model.
func dispatchSearchCode(ctx context.Context, call provider.ToolCall, repoRoot string, parseTimeout time.Duration) provider.ToolResult {
	var req codesearch.SearchRequest
	if err := json.Unmarshal([]byte(call.Arguments), &req); err != nil {
		return provider.ToolResult{CallID: call.ID, Content: errorJSON(fmt.Sprintf("invalid arguments: %v", err))}
	}
	req.RepoRoot = repoRoot
	req.ParseTimeout = parseTimeout

	result, err := codesearch.Search(ctx, req)
	if err != nil {
		return provider.ToolResult{CallID: call.ID, Content: errorJSON(err.Error())}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return provider.ToolResult{CallID: call.ID, Content: errorJSON(fmt.Sprintf("encode result: %v", err))}
	}
	return provider.ToolResult{CallID: call.ID, Content: string(encoded)}
}

func errorJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}
