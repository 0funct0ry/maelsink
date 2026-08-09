import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.tsx'
import './index.css'
import { basePath } from './lib/apiBase.ts'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter basename={basePath()}>
      <App />
    </BrowserRouter>
  </StrictMode>,
)
