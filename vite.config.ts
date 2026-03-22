import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  publicDir: false,
  base: "/build/",
  build: {
    outDir: "assets/public/build",
    emptyOutDir: true,
    manifest: "manifest.json",
    rollupOptions: {
      input: "./cmd/admin-tools/resources/js/app.tsx",
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    hmr: {
      port: 5173,
    },
  },
});
