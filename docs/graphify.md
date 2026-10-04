# Graphify — Knowledge Graph Tooling

[Graphify](https://github.com/Graphify-Labs/graphify) memetakan seluruh codebase (Go, TypeScript, Python, SQL, dll.) menjadi *knowledge graph* lokal yang dapat di-query — tanpa mengirim kode ke server eksternal. Semua parsing dilakukan lokal via tree-sitter AST; edge diberi tag `EXTRACTED` (impor/ref eksplisit) atau `INFERRED` (ko-okurensi).

## Instalasi CLI

```bash
# Direkomendasikan (uv) — sudah terpasang di sandbox Freebuff
uv tool install "graphifyy[sql]"

# Alternatif
pipx install "graphifyy[sql]"
```

> Extra `[sql]` dibutuhkan karena repo ini berisi banyak migrasi SQL
> (`backend/go-core/migrations/`, `schema_pakasir.sql`).

## Generate Knowledge Graph

```bash
npm run graph        # == graphify . --code-only  (tanpa API key, deterministik)
npm run graph:full   # == graphify .  (termasuk ekstraksi semantik dokumen — butuh LLM key)
npm run graph:update # == graphify update .  (refresh AST-only setelah perubahan kode, tanpa biaya)
```

Output ditulis ke `graphify-out/`:

| File | Fungsi |
|---|---|
| `graph.html` | Visualisasi interaktif (filter node, god nodes, komunitas) — buka di browser |
| `graph.json` | Graph mentah yang dapat di-query programatik |
| `GRAPH_REPORT.md` | Ringkasan arsitektur: god nodes, koneksi lintas-file, pertanyaan saran |

`graphify-out/` di-**.gitignore** — selalu di-regenerate dari kode, tidak di-commit.

## Query

```bash
graphify query  "what connects auth to the database?"
graphify path   "AuthHandler" "UserRepository"     # jalur terpendek antar node
graphify explain "RateLimiter"                     # penjelasan node + tetangganya
```

## Catatan: Ekstraksi Dokumen (opsional)

Mode default `graphify .` ingin mengekstrak 131+ file non-kode (docs, ADR, spec)
secara semantik via LLM. Tanpa API key, gunakan `--code-only` (default `npm run graph`).
Untuk ekstraksi penuh, set salah satu key berikut lalu jalankan `npm run graph:full`:

- `GEMINI_API_KEY` / `GOOGLE_API_KEY` (gemini)
- `ANTHROPIC_API_KEY` (claude)
- `OPENAI_API_KEY` (openai)
- `DEEPSEEK_API_KEY` (deepseek)
- `MOONSHOT_API_KEY` (kimi)

## Perawatan

Setelah perubahan kode yang signifikan, jalankan `npm run graph:update`
(AST-only, tanpa API cost) agar graph tetap segar.
