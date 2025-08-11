package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const projectRoot = "../frontend/src" // Path to your React project folder

// Ollama's direct API response
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"` // JSON string with edit actions
	Done      bool   `json:"done"`
}

// The AI's suggested file changes
type AIEditActions struct {
	Actions []struct {
		Type    string `json:"type"`              // "create", "update", "delete"
		Path    string `json:"path"`              // relative path in project
		Content string `json:"content,omitempty"` // new file content for create/update
	} `json:"actions"`
}

type FileJSON struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func main() {
	http.HandleFunc("/api/edit", handleEdit)

	fmt.Println("Backend running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Handle user edit requests
func handleEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Instructions string `json:"instructions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	contextJSON, err := gatherContextJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prompt := buildPrompt(req.Instructions, contextJSON)

	ollamaResp, err := callOllama(prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := applyEdits(ollamaResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// Reads project files into JSON array
func gatherContextJSON() (string, error) {
	files := []FileJSON{}

	err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".ts") ||
			strings.HasSuffix(path, ".jsx") || strings.HasSuffix(path, ".js") ||
			strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".html") {

			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			rel := strings.TrimPrefix(path, projectRoot+string(os.PathSeparator))
			files = append(files, FileJSON{
				Path:    rel,
				Content: string(b),
			})
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// Builds strict JSON edit prompt
func buildPrompt(instructions string, filesJSON string) string {
	return fmt.Sprintf(`You are a helpful AI programming assistant that edits a React + TypeScript project.

Input files are provided as a JSON array of objects with these fields:
[
  {
    "path": "src/App.tsx",
    "content": "<full file content here>"
  },
  {
    "path": "src/components/SidePanel.tsx",
    "content": "<full file content here>"
  }
  ...
]

Instructions:
- Follow the user instructions below precisely.
- Return ONLY a JSON object describing an array of actions.
- Do not modify the SidePanel.tsx file at all.
- You are allowed to create, update, or delete files.
- Do not return any other text, explanations, or comments.
- Do not return any other JSON fields, only "actions".
- Do not return thinking or reasoning steps.
- Each action must be a valid JSON object.
- Each action must have:
  - type: "create", "update", or "delete"
  - path: a relative file path inside the project (e.g. "src/App.tsx")
  - content: full file content (required for create and update; omit for delete)

Example output:

{
  "actions": [
    {
      "type": "update",
      "path": "src/App.tsx",
      "content": "<new file content>"
    },
    {
      "type": "delete",
      "path": "src/oldFile.tsx"
    },
    {
      "type": "create",
      "path": "src/newComponent.tsx",
      "content": "<file content>"
    }
  ]
}

User instructions:
%s

Project files (JSON array):
%s
`, instructions, filesJSON)
}

// Calls Ollama API
func callOllama(prompt string) (AIEditActions, error) {
	reqBody := map[string]any{
		"model": "qwen/qwen3-30b-a3b:free",
		"steam": false,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		return AIEditActions{}, err
	}

	fmt.Println("Sending request to Ollama with prompt length:", len(prompt))
	fmt.Println("Request body:", buf.String())

	// Make the HTTP request to Ollama API
	fmt.Println("Making HTTP POST request to Ollama API...")

	// read the API key from environment variable
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return AIEditActions{}, fmt.Errorf("OPENROUTER_API_KEY environment variable is not set")
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "openrouter.ai", Path: "/api/v1/chat/completions"},
		Header: http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + apiKey},
		},
	}

	req.Body = io.NopCloser(buf)
	fmt.Println("Sending request to Ollama API...")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AIEditActions{}, err
	}
	defer resp.Body.Close()

	bodyString := new(bytes.Buffer)
	if _, err := bodyString.ReadFrom(resp.Body); err != nil {
		return AIEditActions{}, err
	}

	// unmarshal the response
	var ollamaResp map[string]any
	if err := json.Unmarshal(bodyString.Bytes(), &ollamaResp); err != nil {
		return AIEditActions{}, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	// ollamaResp["choices"]["0"]["message"]["content"]
	if _, ok := ollamaResp["choices"]; !ok {
		return AIEditActions{}, fmt.Errorf("unexpected Ollama response format: %v", ollamaResp)
	}

	choices, ok := ollamaResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return AIEditActions{}, fmt.Errorf("no choices found in Ollama response")
	}

	message, ok := choices[0].(map[string]interface{})
	if !ok {
		return AIEditActions{}, fmt.Errorf("unexpected choice format: %v", choices[0])
	}

	content, ok := message["message"].(map[string]interface{})["content"].(string)
	if !ok {
		return AIEditActions{}, fmt.Errorf("unexpected message format: %v", message)
	}

	var edits AIEditActions
	if err := json.Unmarshal([]byte(content), &edits); err != nil {
		return AIEditActions{}, fmt.Errorf("failed to parse AI edits: %w", err)
	}

	// Log the AI's suggested actions
	log.Printf("AI suggested %d actions:\n", len(edits.Actions))
	// Print the actions in a readable format

	fmt.Printf("AI suggested %d actions:\n", len(edits.Actions))
	for _, act := range edits.Actions {
		fmt.Printf("- %s: %s\n", act.Type, act.Path)
		if act.Type == "create" || act.Type == "update" {
			fmt.Printf("  Content length: %d\n", len(act.Content))
		}
		if act.Type == "delete" {
			fmt.Printf("  Will delete file: %s\n", act.Path)
		}
	}
	if len(edits.Actions) == 0 {
		log.Println("AI returned no actions, nothing to apply.")
		return AIEditActions{}, fmt.Errorf("no actions returned by AI")
	}
	if len(edits.Actions) > 100 {
		log.Println("AI returned too many actions, limiting to 100")
		edits.Actions = edits.Actions[:100] // Limit to 100 actions
		fmt.Println("Limited actions to 100 due to API constraints.")
	} else if len(edits.Actions) > 50 {
		log.Println("AI returned a large number of actions, consider reviewing them carefully")
		fmt.Println("AI returned a large number of actions, consider reviewing them carefully")
	}
	if len(edits.Actions) == 0 {
		log.Println("AI returned no actions, nothing to apply.")
		return AIEditActions{}, fmt.Errorf("no actions returned by AI")
	}

	return edits, nil
}

// Applies the AI edits to local files
func applyEdits(edits AIEditActions) error {
	for _, act := range edits.Actions {
		fullPath := filepath.Join(projectRoot, "..", act.Path)

		switch act.Type {
		case "create", "update":
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return err
			}
			if err := ioutil.WriteFile(fullPath, []byte(act.Content), 0644); err != nil {
				return err
			}
			log.Printf("%s file: %s\n", strings.Title(act.Type), fullPath)
		case "delete":
			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			log.Printf("Deleted file: %s\n", fullPath)
		default:
			log.Printf("Unknown action type: %s\n", act.Type)
		}
	}
	return nil
}
