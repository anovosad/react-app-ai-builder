import React, { useState, useEffect } from "react"
import SidePanel from "./components/SidePanel"

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
    <div style={{padding:20}}>
      <h1>AI-editable React app</h1>
      <p>Open the right-hand panel to apply AI changes (it writes files directly).</p>
      <label>
        <input
          type="checkbox"
          checked={darkMode}
          onChange={() => setDarkMode(!darkMode)}
        />
        Dark Mode
      </label>
      <SidePanel />
    </div>
  )
}
