import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import { Toaster } from 'sonner'
import { ThemeProvider } from 'next-themes'
import { NextUIProvider } from "@nextui-org/react";

const container = document.getElementById('root')

const root = ReactDOM.createRoot(container!)

root.render(
  <React.StrictMode>
    <NextUIProvider>
      <ThemeProvider enableSystem={true} attribute="class">
        <div className="min-h-screen flex flex-col">
          <Toaster richColors position="top-center" />
          <main className="flex-1">
            <App />
          </main>
        </div>
      </ThemeProvider>
    </NextUIProvider>
  </React.StrictMode>,
)
