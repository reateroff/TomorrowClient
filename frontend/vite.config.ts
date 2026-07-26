import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // 27182 (e×10⁴). Chosen rather than left on Vite's 5173, which was already
    // taken on this machine: it sits below Windows' ephemeral range
    // (49152-65535), so the OS will never hand it to something else; it is not
    // the default of any common dev tool (3000/4200/5173/8080/9000); and it is
    // clear of the ports the client itself uses (19090 for the sing-box Clash
    // API) and of the Wails dev bridge on 34115.
    port: 27182,
    // Fail loudly instead of sliding to the next free port. Vite's default
    // fallback is exactly what left the dev server on 5174 while everything
    // else still expected 5173 — a fixed port is the whole point here.
    strictPort: true,
  },
})
