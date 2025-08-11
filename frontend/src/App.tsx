import React, { useState, useEffect } from "react"
import SidePanel from "./components/SidePanel"
import Counter from "./components/Counter"

export default function App(){
  const [darkMode, setDarkMode] = useState(true);
  
  useEffect(() => {
    if (darkMode) {
      document.body.classList.add('dark-mode');
      document.body.classList.remove('light-mode');
    } else {
      document.body.classList.add('light-mode');
      document.body.classList.remove('dark-mode');
    }
  }, [darkMode]);

  return (
    <>
      {/* Main content area with right margin for side panel */}
      <div style={{
        padding: 20,
        marginRight: 380, // Width of side panel
        minHeight: '100vh',
        boxSizing: 'border-box'
      }}>
        <h1>AI-Editable React App</h1>
        <p>
          Welcome to your AI-powered React development environment! 
          Use the side panel on the right to make changes to your application using natural language.
        </p>
        
        <div style={{ marginBottom: 24 }}>
          <h2>Features</h2>
          <ul>
            <li>✨ AI-powered code editing with multiple LLM providers</li>
            <li>🔄 Real-time file updates that reflect instantly in the browser</li>
            <li>🎯 Support for both OpenRouter API and local Ollama models</li>
            <li>🛡️ Protected side panel that won't be modified by AI</li>
          </ul>
        </div>

        <div style={{ marginBottom: 24 }}>
          <h2>Settings</h2>
          <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={darkMode}
              onChange={() => setDarkMode(!darkMode)}
            />
            Dark Mode
          </label>
        </div>

        <div style={{ marginBottom: 24 }}>
          <h2>Getting Started</h2>
          <ol>
            <li>Choose your preferred AI provider (OpenRouter or Ollama) in the side panel</li>
            <li>Select a model that suits your needs</li>
            <li>Type your instructions in natural language</li>
            <li>Press "Apply Changes" or use Ctrl+Enter to execute</li>
            <li>Watch your changes appear instantly!</li>
          </ol>
        </div>

        <div style={{ 
          padding: 16, 
          borderRadius: 8, 
          backgroundColor: darkMode ? '#2d3748' : '#f7fafc',
          border: `1px solid ${darkMode ? '#4a5568' : '#e2e8f0'}`
        }}>
          <h3>Example Instructions</h3>
          <p>Try asking the AI to:</p>
          <ul>
            <li>"Add a counter component with increment and decrement buttons"</li>
            <li>"Create a todo list with add, delete, and toggle complete functionality"</li>
            <li>"Add a responsive navigation bar with multiple menu items"</li>
            <li>"Create a contact form with validation"</li>
          </ul>
        </div>

        <Counter />
      </div>

      {/* Side panel - always rendered last to ensure it's on top */}
      <SidePanel />
    </>
  )
}