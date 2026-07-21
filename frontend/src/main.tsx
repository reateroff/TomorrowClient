import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'

// Make the webview feel native: no right-click menu, no dragging elements or
// images around, and no accidental text selection outside inputs.
document.addEventListener('contextmenu', (e) => e.preventDefault())
document.addEventListener('dragstart', (e) => e.preventDefault())
document.addEventListener('drop', (e) => e.preventDefault())

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
)
