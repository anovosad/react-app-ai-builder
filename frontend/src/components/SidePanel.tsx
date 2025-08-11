import React, { useState, useEffect } from 'react';

interface Models {
  openrouter: string[];
  ollama: string[];
}

export default function SidePanel() {
  const [instructions, setInstructions] = useState('');
  const [status, setStatus] = useState('');
  const [provider, setProvider] = useState<'openrouter' | 'ollama'>('openrouter');
  const [models, setModels] = useState<Models>({ openrouter: [], ollama: [] });
  const [selectedModel, setSelectedModel] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  // Fetch available models on component mount
  useEffect(() => {
    fetchModels();
  }, []);

  // Update selected model when provider changes
  useEffect(() => {
    if (models[provider]?.length > 0) {
      setSelectedModel(models[provider][0]);
    }
  }, [provider, models]);

  const fetchModels = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/models');
      const data: Models = await response.json();
      setModels(data);
      
      // Set default model
      if (data.openrouter?.length > 0) {
        setSelectedModel(data.openrouter[0]);
      }
    } catch (error) {
      console.error('Failed to fetch models:', error);
      setStatus('Failed to fetch models');
    }
  };

  const send = async () => {
    if (!instructions.trim()) {
      setStatus('Please enter instructions');
      return;
    }

    if (!selectedModel) {
      setStatus('Please select a model');
      return;
    }

    setIsLoading(true);
    setStatus('Sending...');
    
    try {
      const res = await fetch('http://localhost:8080/api/edit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          instructions, 
          provider, 
          model: selectedModel 
        }),
      });
      
      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(`HTTP ${res.status}: ${errorText}`);
      }
      
      const data = await res.json();
      setStatus(`Applied: ${data.applied || 0} actions`);
      setInstructions(''); // Clear instructions after successful execution
    } catch (e: any) {
      setStatus('Error: ' + String(e.message || e));
    } finally {
      setIsLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      send();
    }
  };

  return (
    <div
      style={{
        position: 'fixed',
        right: 0,
        top: 0,
        width: 380,
        height: '100vh',
        backgroundColor: '#f8f9fa',
        borderLeft: '1px solid #dee2e6',
        padding: 16,
        boxSizing: 'border-box',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
        fontSize: 14,
        zIndex: 9999,
        // Reset any potential inherited styles
        color: '#212529',
        lineHeight: '1.4',
        textAlign: 'left',
      }}
    >
      <div style={{ marginBottom: 16 }}>
        <h3 style={{ 
          margin: '0 0 16px 0', 
          fontSize: 18, 
          fontWeight: 600,
          color: '#212529'
        }}>
          AI Code Editor
        </h3>
        
        {/* Provider Selection */}
        <div style={{ marginBottom: 12 }}>
          <label style={{ 
            display: 'block', 
            marginBottom: 6, 
            fontWeight: 500,
            color: '#495057'
          }}>
            Provider:
          </label>
          <div style={{ display: 'flex', gap: 8 }}>
            <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
              <input
                type="radio"
                value="openrouter"
                checked={provider === 'openrouter'}
                onChange={(e) => setProvider(e.target.value as 'openrouter')}
                style={{ marginRight: 6 }}
              />
              OpenRouter
            </label>
            <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
              <input
                type="radio"
                value="ollama"
                checked={provider === 'ollama'}
                onChange={(e) => setProvider(e.target.value as 'ollama')}
                style={{ marginRight: 6 }}
              />
              Ollama
            </label>
          </div>
        </div>

        {/* Model Selection */}
        <div style={{ marginBottom: 16 }}>
          <label style={{ 
            display: 'block', 
            marginBottom: 6, 
            fontWeight: 500,
            color: '#495057'
          }}>
            Model:
          </label>
          <select
            value={selectedModel}
            onChange={(e) => setSelectedModel(e.target.value)}
            style={{
              width: '100%',
              padding: '8px 12px',
              border: '1px solid #ced4da',
              borderRadius: 4,
              backgroundColor: 'white',
              fontSize: 14,
              color: '#495057'
            }}
          >
            {models[provider]?.map((model) => (
              <option key={model} value={model}>
                {model}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Instructions */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <label style={{ 
          display: 'block', 
          marginBottom: 8, 
          fontWeight: 500,
          color: '#495057'
        }}>
          Instructions:
        </label>
        <textarea
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Describe what you want to change in your React app...&#10;&#10;Press Ctrl+Enter (Cmd+Enter on Mac) to send"
          style={{
            flex: 1,
            minHeight: 120,
            padding: 12,
            border: '1px solid #ced4da',
            borderRadius: 4,
            backgroundColor: 'white',
            fontSize: 14,
            fontFamily: 'inherit',
            resize: 'none',
            outline: 'none',
            color: '#495057'
          }}
        />
      </div>

      {/* Action Button */}
      <button
        onClick={send}
        disabled={isLoading || !instructions.trim() || !selectedModel}
        style={{
          marginTop: 12,
          padding: '12px 16px',
          backgroundColor: isLoading || !instructions.trim() || !selectedModel ? '#6c757d' : '#007bff',
          color: 'white',
          border: 'none',
          borderRadius: 4,
          fontSize: 14,
          fontWeight: 500,
          cursor: isLoading || !instructions.trim() || !selectedModel ? 'not-allowed' : 'pointer',
          transition: 'background-color 0.2s'
        }}
      >
        {isLoading ? 'Applying Changes...' : 'Apply Changes'}
      </button>

      {/* Status */}
      {status && (
        <div 
          style={{ 
            marginTop: 12,
            padding: 8,
            borderRadius: 4,
            backgroundColor: status.includes('Error') ? '#f8d7da' : '#d4edda',
            color: status.includes('Error') ? '#721c24' : '#155724',
            fontSize: 13,
            border: `1px solid ${status.includes('Error') ? '#f5c6cb' : '#c3e6cb'}`,
            wordBreak: 'break-word'
          }}
        >
          {status}
        </div>
      )}

      {/* Help Text */}
      <div style={{ 
        marginTop: 12, 
        fontSize: 12, 
        color: '#6c757d',
        lineHeight: '1.3'
      }}>
        💡 This AI can create, update, or delete files in your React project. 
        Changes will be applied instantly and reflected in your browser.
        <br />
        <strong>Note:</strong> Make sure to use Git or have backups!
      </div>
    </div>
  );
}