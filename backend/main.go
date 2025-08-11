package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const projectRoot = "../frontend/src" // Path to your React project folder

// Request from frontend
type EditRequest struct {
	Instructions string `json:"instructions"`
	Provider     string `json:"provider"` // "openrouter" or "ollama"
	Model        string `json:"model"`
}

// OpenRouter API response
type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Ollama API response
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
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
	// Enable CORS
	http.HandleFunc("/api/edit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handleEdit(w, r)
	})

	// Add models endpoint
	http.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		models := map[string][]string{
			"openrouter": {
				"qwen/qwen-2.5-7b-instruct:free",
				"qwen/qwen3-30b-a3b:free",
				"meta-llama/llama-3.1-8b-instruct:free",
				"anthropic/claude-3.5-sonnet",
				"openai/gpt-4o",
				"openai/gpt-4o-mini",
				"google/gemini-pro-1.5",
				"qwen/qwen-2.5-72b-instruct",
			},
			"ollama": {
				"llama3.2",
				"qwen2.5",
				"codellama",
				"deepseek-coder",
				"starcoder2",
			},
		}

		json.NewEncoder(w).Encode(models)
	})

	fmt.Println("Backend running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Handle user edit requests
func handleEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EditRequest
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

	var aiResponse string
	var parseErr error

	switch req.Provider {
	case "openrouter":
		aiResponse, parseErr = callOpenRouter(prompt, req.Model)
	case "ollama":
		aiResponse, parseErr = callOllama(prompt, req.Model)
	default:
		http.Error(w, "Invalid provider. Use 'openrouter' or 'ollama'", http.StatusBadRequest)
		return
	}

	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusInternalServerError)
		return
	}

	// Clean up the AI response before parsing
	cleanedResponse := cleanAIResponse(aiResponse)

	var edits AIEditActions
	if err := json.Unmarshal([]byte(cleanedResponse), &edits); err != nil {
		log.Printf("Failed to parse AI response as JSON: %v", err)
		log.Printf("Original response: %s", aiResponse)
		log.Printf("Cleaned response: %s", cleanedResponse)
		http.Error(w, fmt.Sprintf("Failed to parse AI response as JSON: %v\nOriginal Response: %s", err, aiResponse), http.StatusInternalServerError)
		return
	}

	if err := applyEdits(edits); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"applied": len(edits.Actions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	// Extract current file structure for the LLM
	fileStructure := extractFileStructure(filesJSON)

	return fmt.Sprintf(`You are a helpful AI programming assistant that edits a React + TypeScript project.

CURRENT PROJECT STRUCTURE:
%s

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

CRITICAL FILE PATH RULES:
- NEVER create nested src directories (src/src/... is WRONG)
- NEVER add project prefixes (frontend/src/... is WRONG) 
- ALL file paths must be relative to the project root
- Use EXACTLY these path patterns:
  * For main files: "src/App.tsx", "src/main.tsx", "src/styles.css"
  * For components: "src/components/ComponentName.tsx"
  * For component styles: "src/components/ComponentName.css"
- DO NOT add extra directories or change the existing structure
- DO NOT modify the SidePanel.tsx file under any circumstances
- When creating new React components, ALWAYS put them in "src/components/" directory
- EXAMPLES OF CORRECT PATHS: "src/App.tsx", "src/components/Counter.tsx", "src/components/TodoList.tsx"
- EXAMPLES OF WRONG PATHS: "src/src/App.tsx", "frontend/src/App.tsx", "components/Counter.tsx"

IMPORTANT INSTRUCTIONS:
- Follow the user instructions below precisely.
- Return ONLY a valid JSON object describing an array of actions.
- You are allowed to create, update, or delete files.
- Do not return any text, explanations, or comments outside the JSON.
- Do not return any other JSON fields, only "actions".
- Do not return thinking or reasoning steps.
- Use proper JSON escaping for newlines and quotes.
- Do NOT use HTML entities like \u003c or \u003e in your response.
- Make sure your JSON is properly formatted and parseable.
- Each action must be a valid JSON object.
- Each action must have:
  - type: "create", "update", or "delete"
  - path: a relative file path following the rules above
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
      "type": "create", 
      "path": "src/components/NewComponent.tsx",
      "content": "<file content>"
    }
  ]
}

User instructions:
%s

Project files (JSON array):
%s
`, fileStructure, instructions, filesJSON)
}

// Calls OpenRouter API
func callOpenRouter(prompt string, model string) (string, error) {
	godotenv.Load() // Load environment variables from .env file
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY environment variable is not set")
	}

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter API error %d: %s", resp.StatusCode, string(body))
	}

	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", fmt.Errorf("failed to parse OpenRouter response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenRouter response")
	}

	return cleanAIResponse(openRouterResp.Choices[0].Message.Content), nil
}

// Calls local Ollama API
func callOllama(prompt string, model string) (string, error) {
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "http://localhost:11434/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama (make sure it's running on localhost:11434): %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API error %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return cleanAIResponse(ollamaResp.Response), nil
}

// Clean up AI response to fix common JSON parsing issues
func cleanAIResponse(response string) string {
	// Remove any text before the first {
	startIndex := strings.Index(response, "{")
	if startIndex > 0 {
		response = response[startIndex:]
	}

	// Remove any text after the last }
	lastIndex := strings.LastIndex(response, "}")
	if lastIndex > 0 && lastIndex < len(response)-1 {
		response = response[:lastIndex+1]
	}

	// Fix common HTML entity escapes that break JSON
	response = strings.ReplaceAll(response, "\\u003c", "<")
	response = strings.ReplaceAll(response, "\\u003e", ">")
	response = strings.ReplaceAll(response, "\\u0026", "&")
	response = strings.ReplaceAll(response, "\\u0027", "'")
	response = strings.ReplaceAll(response, "\\u0022", "\"")

	// Fix malformed escape sequences
	response = strings.ReplaceAll(response, "\\un", "\\n")
	response = strings.ReplaceAll(response, ");n}", ");}")

	// Fix common malformed endings
	response = strings.ReplaceAll(response, "\\n}", "}")

	return response
}

// Applies the AI edits to local files
func applyEdits(edits AIEditActions) error {
	log.Printf("Applying %d edit actions", len(edits.Actions))

	for _, act := range edits.Actions {
		// Normalize the path to prevent incorrect nesting
		normalizedPath := normalizePath(act.Path)

		// Log path changes for debugging
		if normalizedPath != act.Path {
			log.Printf("Normalized path: %s -> %s", act.Path, normalizedPath)
		}

		// Prevent editing the SidePanel
		if strings.Contains(normalizedPath, "SidePanel") {
			log.Printf("Skipping SidePanel modification: %s", normalizedPath)
			continue
		}

		// Validate that we're not creating files outside the project
		if strings.Contains(normalizedPath, "..") || strings.HasPrefix(normalizedPath, "/") {
			log.Printf("Skipping potentially dangerous path: %s", normalizedPath)
			continue
		}

		// Build full path for file operations
		fullPath := filepath.Join(projectRoot, strings.TrimPrefix(normalizedPath, "src/"))

		switch act.Type {
		case "create", "update":
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return err
			}
			if err := ioutil.WriteFile(fullPath, []byte(act.Content), 0644); err != nil {
				return err
			}
			log.Printf("%s file: %s (normalized from: %s)", strings.Title(act.Type), fullPath, act.Path)
		case "delete":
			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			log.Printf("Deleted file: %s (normalized from: %s)", fullPath, act.Path)
		default:
			log.Printf("Unknown action type: %s", act.Type)
		}
	}
	return nil
}
