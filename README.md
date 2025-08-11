# AI Sidepanel + Ollama Prototype

This scaffold provides a working prototype that lets you:
- Run a local Ollama model (e.g. Qwen).
- Run a React dev server (Vite) with a right-hand AI side panel in the browser.
- Run a Go backend that collects project context, sends it to Ollama, and applies create/update/delete file actions inside the frontend folder.

## Requirements
- Go 1.20+
- Node 18+
- Ollama installed and a local model available (e.g. `qwen:14b`).

## Quick start

1. Start Ollama (example):
```bash
ollama run qwen:14b
```

2. Start backend:
```bash
cd backend
go mod tidy
go run main.go
```

3. Start frontend:
```bash
cd frontend
npm install
npm run dev
```

4. Open the Vite URL (likely http://localhost:5173). Use the right-hand panel to pick a file and type instructions. The backend will call Ollama and apply the returned JSON actions directly inside `frontend/`.

**Important:** This prototype writes files directly. Use Git or backups. Consider enabling automatic commits or an undo endpoint before heavy use.
