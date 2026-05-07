import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { editorApiPlugin } from "./server/api";

export default defineConfig(() => ({
  plugins: [
    react(),
    editorApiPlugin({
      rawTomlPath: "../assets/metadata/entities/raw/raw.toml",
      palettesDir: "../assets/levels/palettes",
      layoutsDir: "../assets/levels/layouts",
      chunkDirs: [
        "../assets/levels/layouts",
        "../assets/levels/facilities",
        "../assets/levels/chunks",
      ],
      assetsDir: "../assets",
    }),
  ],
  server: {
    port: 3000,
  },
  test: {
    globals: true,
    environment: "happy-dom",
    setupFiles: ["./src/setupTests.ts"],
  },
}));
