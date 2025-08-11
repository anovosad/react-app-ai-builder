import React, { useState } from 'react';

export default function SidePanel() {
  const mainFile = 'src/App.tsx';  // fixed file to edit
  const [instructions, setInstructions] = useState('');
  const [status, setStatus] = useState('');

  const send = async () => {
    setStatus('Sending...');
    try {
      const res = await fetch('http://localhost:8080/api/edit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mainFile, instructions }),
      });
      const data = await res.json();
      setStatus('Applied: ' + (data.applied?.length || 0) + ' actions');
    } catch (e: any) {
      setStatus('Error: ' + String(e.message || e));
    }
  };

  return (
    <div
      style={{
        position: 'fixed',
        right: 0,
        top: 0,
        width: 320,
        height: '100vh',
        background: '#fff',
        borderLeft: '1px solid #ddd',
        padding: 12,
      }}
    >
      <h3>AI Editor (editing {mainFile})</h3>
      <textarea
        value={instructions}
        onChange={(e) => setInstructions(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault(); // prevent newline
            send();
          }
        }}
        style={{ width: '100%', height: 200, marginTop: 8 }}
      />
      <button onClick={send} style={{ marginTop: 8 }}>
        Apply
      </button>
      <div style={{ marginTop: 12 }}>{status}</div>
    </div>
  );
}
