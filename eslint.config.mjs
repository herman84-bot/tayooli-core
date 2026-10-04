import { defineConfig } from "eslint/config";
import next from "eslint-config-next";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export default defineConfig([
  {
    ignores: [
      ".claude/**",
      "coverage/**",
      "frontend/**",
      ".next/**",
      "node_modules/**",
    ],
  },
  {
    extends: [...next],
    rules: {
      "react-hooks/set-state-in-effect": "warn",
      "react-hooks/purity": "warn",
      "react-hooks/immutability": "warn",
      "react-hooks/incompatible-library": "warn",
    },
  },
]);
