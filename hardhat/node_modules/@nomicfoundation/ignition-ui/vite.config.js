import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { viteSingleFile } from "vite-plugin-singlefile";
// https://vitejs.dev/config/
export default defineConfig({
    plugins: [react(), viteSingleFile()],
    optimizeDeps: {
        include: ["@nomicfoundation/ignition-core/ui-helpers"],
    },
    build: {
        commonjsOptions: {
            include: [/core/, /node_modules/],
        },
    },
});
