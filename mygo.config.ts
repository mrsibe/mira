import { defineConfig } from "mygo-cli";

export default defineConfig({
  name: "Mira",
  identifier: "com.mrsibe.mira",
  version: "0.3.3",
  copyright: "© 2026 MrSibe",
  publisher: "MrSibe",
  updates: {
    publicKey: "2cdI3cyRcrHa2YNZUIsojIgAhXutvNa11Cj/4OhMnT8=",
    github: "mrsibe/mira",
  },
  // `mygo dev` runs devCommand and loads devUrl; `mygo build` runs
  // buildCommand and embeds frontendDist into the app.
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",
});
