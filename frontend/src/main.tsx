import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'
import { setupBrowserMockIfNeeded } from './lib/browserMock'

// Provide safe fallback mock when running in standalone browser dev/preview modes
setupBrowserMockIfNeeded()

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>
)
