# Backlog Implementasi — apitest CLI

Dokumen ini berisi item-item yang belum diimplementasi, dikelompokkan per fase.
Gunakan sebagai panduan untuk melanjutkan development.

**Status saat ini:** MVP (Fase 1) selesai — binary `bin/apitest` berfungsi.

**Tanggal terakhir update:** 2026-07-13

---

## Status Fase 1 (MVP) — ✅ DONE

| Fitur | File | Status |
|---|---|---|
| FlowSpec Lexer | `src/internal/lexer/` | ✅ |
| FlowSpec Parser (recursive descent) | `src/internal/parser/parser.go` | ✅ |
| AST node types | `src/internal/ast/ast.go` | ✅ |
| HTTP request engine | `src/internal/runtime/http.go` | ✅ |
| Environment & variables | `src/internal/runtime/variables.go` | ✅ |
| Variable interpolation `{{var}}` | `src/internal/runtime/variables.go` | ✅ |
| Dynamic vars (`$uuid`, `$timestamp`, `$randomEmail`) | `src/internal/runtime/variables.go` | ✅ |
| `expect status` (exact, range, list, negation) | `src/internal/runtime/assertions.go` | ✅ |
| `expect json` (JSONPath: ==, exists, is, length, matches, >=) | `src/internal/runtime/assertions.go` | ✅ |
| `expect header` (exists, contains, matches) | `src/internal/runtime/assertions.go` | ✅ |
| `expect time` | `src/internal/runtime/assertions.go` | ✅ |
| `extract { var from json/header }` | `src/internal/runtime/engine.go` | ✅ |
| Flow execution (linear steps) | `src/internal/runtime/engine.go` | ✅ |
| `when` / `unless` conditions | `src/internal/runtime/engine.go` | ✅ |
| `let` variables | `src/internal/runtime/engine.go` | ✅ |
| `run` with override block | `src/internal/runtime/engine.go` | ✅ |
| Teardown (ignore_fail) | `src/internal/runtime/engine.go` | ✅ |
| `--fail-fast` | `src/internal/cli/run.go` | ✅ |
| `--tags` filter | `src/internal/cli/run.go` | ✅ |
| `--var key=value` override | `src/internal/cli/run.go` | ✅ |
| `apitest init` | `src/internal/cli/init.go` | ✅ |
| `apitest run` | `src/internal/cli/run.go` | ✅ |
| `apitest dsl lint` | `src/internal/cli/lint.go` | ✅ |
| `apitest dsl show` | `src/internal/cli/show.go` | ✅ |
| `apitest import curl` (inline + file) | `src/internal/cli/importcurl.go` | ✅ |
| Console reporter (colored, verbose, quiet) | `src/internal/reporter/console.go` | ✅ |
| JSON reporter | `src/internal/reporter/json.go` | ✅ |
| JUnit XML reporter | `src/internal/reporter/junit.go` | ✅ |
| Exit codes (0/1/2) | `src/cmd/main.go` | ✅ |
| .env file auto-loading | `src/internal/cli/run.go` | ✅ |
| `import` cross-file references | `src/internal/runtime/engine.go` | ✅ |
| `project` block parsing (skip) | `src/internal/parser/parser.go` | ✅ |

---

## Fase 2 — Automation & Team Ready

### P0 — Control Flow Execution (parser sudah ada, runtime belum)

#### BACK-001: `repeat N { ... }` execution
- **Apa:** Jalankan block N kali di runtime
- **Parser:** ✅ `parseRepeatAsStep()` sudah buat `RepeatDecl` di AST
- **Runtime:** ❌ `executeStep()` tidak handle `step.Repeat`
- **Lokasi:** `src/internal/runtime/engine.go` → `executeStep()`
- **Acceptance:** `repeat 3 { run CreatePost }` menghasilkan 3 request terkirim
- **Estimasi:** 30 menit

#### BACK-002: `for row in csv("path")` execution
- **Apa:** Baca CSV, iterasi per baris, set variable per row
- **Parser:** ✅ `parseForAsStep()` sudah buat `ForLoopDecl` di AST
- **Runtime:** ❌ `executeStep()` tidak handle `step.ForLoop`
- **Yang perlu dibuat:**
  1. CSV reader (gunakan `encoding/csv` stdlib)
  2. Loop execution: per row → set variables → execute inner steps
  3. Support `row.column_name` access
- **Lokasi:** `src/internal/runtime/engine.go` + file baru `src/internal/runtime/dataloader.go`
- **Acceptance:** File CSV 3 baris → 3 iterasi, variable per row tersedia
- **Estimasi:** 1-2 jam

#### BACK-003: `for item in [...]` inline array execution
- **Apa:** Iterasi array literal inline
- **Parser:** ✅ (array di-skip, perlu parse items)
- **Runtime:** ❌
- **Yang perlu:**
  1. Parser: parse inline array of objects `[{k:v}, {k:v}]` ke `ForLoopDecl.Items`
  2. Runtime: iterate items, set variables
- **Estimasi:** 1-2 jam

#### BACK-004: `retry N times every Xs until ... { run X }` proper execution
- **Apa:** Polling loop — jalankan request, cek kondisi, ulangi/stop
- **Parser:** ✅ `parseRetry()` sudah buat `RetryDecl`
- **Runtime:** ⚠️ Partial — loop ada tapi kondisi `until` tidak di-evaluate terhadap response
- **Yang perlu di-fix:**
  1. Setelah `retry.Run` execute, evaluate `retry.Condition` terhadap response terakhir
  2. Stop loop jika kondisi terpenuhi
  3. Proper error message jika exhausted
- **Lokasi:** `src/internal/runtime/engine.go` → `executeRetry()`
- **Acceptance:** `retry 5 times every 1s until json "$.status" == "done"` stops saat response memenuhi kondisi
- **Estimasi:** 1 jam

#### BACK-005: `wait Ns` execution
- **Apa:** Pause antara steps
- **Parser:** ✅
- **Runtime:** ⚠️ Perlu verifikasi — kode `time.Sleep` ada di `executeStep()` tapi perlu cek path-nya benar
- **Estimasi:** 15 menit (verify + fix jika perlu)

---

### P1 — Composition & Reuse

#### BACK-006: `extends` request inheritance
- **Apa:** `request B extends A { ... }` → merge headers, body, expects dari parent
- **Parser:** ✅ `req.Extends` di-populate
- **Runtime:** ❌ Tidak ada logic resolve parent
- **Yang perlu:**
  1. Saat `ExecuteRequest` atau `prepareRequest`: jika `req.Extends != ""`, lookup parent di `engine.Requests`
  2. Merge: method + URL dari parent (kecuali override), headers append, body fields merge, expects append
- **Lokasi:** `src/internal/runtime/engine.go` → `prepareRequest()` atau function baru `resolveExtends()`
- **Acceptance:** `CreateAdmin extends CreateUser { body json { role: "admin" } }` → body berisi field parent + role
- **Estimasi:** 1 jam

#### BACK-007: `fragment` execution
- **Apa:** `use fragment LoginSetup` → inject steps dari fragment ke flow
- **Parser:** ✅ Fragment parsed, `use fragment X` menjadi step placeholder
- **Runtime:** ❌ Step "use fragment X" tidak resolve ke actual steps
- **Yang perlu:**
  1. Di `ExecuteFlow()`, detect step.Name prefix "use fragment "
  2. Lookup fragment di `engine.Fragments`
  3. Execute fragment's steps sebagai bagian dari flow
- **Lokasi:** `src/internal/runtime/engine.go` → `ExecuteFlow()` atau `executeStep()`
- **Acceptance:** Fragment dengan 2 steps → kedua steps jalan saat `use fragment`
- **Estimasi:** 45 menit

#### BACK-008: `include flows/other.flow` execution
- **Apa:** Jalankan flow lain sebagai sub-flow
- **Parser:** ✅ `flow.Includes` populated
- **Runtime:** ❌ Includes tidak di-execute
- **Yang perlu:**
  1. Di `ExecuteFlow()`, process includes — load file, find flow, execute
  2. Results dari sub-flow digabung ke parent results
- **Estimasi:** 45 menit

---

### P2 — Import & Export

#### BACK-009: OpenAPI import → generate .flow files
- **Apa:** `apitest import openapi specs/openapi.yaml --output requests/`
- **Yang perlu:**
  1. Parse OpenAPI 3.0/3.1 YAML/JSON (gunakan library atau manual)
  2. Per operation → generate `.flow` file dengan method, URL, headers, body example, expect status
  3. Optional: generate contract expect assertions
- **Dependency:** Library OpenAPI parser (atau custom minimal parser)
- **Estimasi:** 4-6 jam

#### BACK-010: Postman Collection import → generate .flow files
- **Apa:** `apitest import postman collection.json --output requests/`
- **Yang perlu:**
  1. Parse Postman Collection v2.1 JSON format
  2. Map items → .flow request files
  3. Convert `{{variable}}` format (sudah kompatibel)
- **Estimasi:** 3-4 jam

---

### P3 — Developer Experience

#### BACK-011: `apitest dsl fmt` — auto-formatter
- **Apa:** Format/prettify .flow files ke style konsisten
- **Yang perlu:**
  1. Parse file ke AST
  2. Re-emit AST ke formatted string (indentation, spacing, newlines)
  3. `--write` flag untuk overwrite in-place, atau stdout
- **Lokasi:** `src/internal/cli/fmt.go` (baru)
- **Estimasi:** 3-4 jam

#### BACK-012: Request history
- **Apa:** Simpan setiap request yang dikirim ke local storage
- **Yang perlu:**
  1. Storage: JSON Lines file (`.apitest/history.jsonl`) atau SQLite
  2. Per entry: timestamp, method, URL, status, duration, ID
  3. CLI: `apitest history` (list), `apitest history replay <id>`, `apitest history clear`
- **Estimasi:** 2-3 jam

#### BACK-013: `apitest dsl migrate` — YAML to .flow converter
- **Apa:** Konversi file YAML legacy ke FlowSpec DSL
- **Yang perlu:**
  1. YAML parser (gopkg.in/yaml.v3)
  2. Map YAML structure → AST → emit .flow
- **Estimasi:** 3-4 jam

---

### P4 — Auth & Security

#### BACK-014: OAuth 2.0 Client Credentials flow
- **Apa:** Auto-fetch token sebelum request
- **Yang perlu:**
  1. Config di env: `oauth2_token_url`, `client_id`, `client_secret`
  2. Pre-request: POST ke token URL, extract `access_token`
  3. Cache token sampai expire
- **Estimasi:** 2-3 jam

#### BACK-015: Secret redaction di output
- **Apa:** Mask token/password di console & report
- **Status:** Sebagian sudah (`.env` loaded), tapi console output belum redact
- **Yang perlu:**
  1. Track variable keys yang mengandung "token", "password", "secret", "key"
  2. Ganti value dengan `***REDACTED***` di verbose output
  3. Respect `settings { redact [...] }` dari project config
- **Estimasi:** 1 jam

---

## Fase 3 — Advanced

#### BACK-016: HTML report
- **Reporter:** static HTML file dengan summary, timeline, detail per step
- **Estimasi:** 4-6 jam

#### BACK-017: `parallel { ... }` execution
- **Apa:** Jalankan beberapa request concurrent
- **Yang perlu:** goroutines + sync, collect results
- **Estimasi:** 2-3 jam

#### BACK-018: LSP / VS Code extension
- **Apa:** Syntax highlighting, autocomplete, go-to-definition untuk .flow files
- **Estimasi:** Besar — project terpisah

#### BACK-019: `apitest dsl repl`
- **Apa:** Interactive mode — ketik request, langsung execute
- **Estimasi:** 3-4 jam

#### BACK-020: `expect soft` (soft assertions)
- **Apa:** Assertion gagal tapi test lanjut, dilaporkan di akhir
- **Parser:** ✅ (field `Soft` di ExpectDecl)
- **Runtime:** ❌ Semua failure masih hard-stop step
- **Estimasi:** 30 menit

---

## Prioritas Rekomendasi (besok)

Urutan yang paling impactful untuk dikerjakan:

```
1. BACK-001  repeat execution         (30 min)  — unlock loop testing
2. BACK-005  wait execution verify    (15 min)  — quick win
3. BACK-004  retry/until proper       (1 hr)    — unlock polling
4. BACK-002  for/csv execution        (1-2 hr)  — unlock data-driven
5. BACK-006  extends inheritance      (1 hr)    — unlock composition
6. BACK-007  fragment execution       (45 min)  — unlock reuse
7. BACK-020  soft assertions          (30 min)  — quick win
8. BACK-015  secret redaction         (1 hr)    — security
```

Total estimasi untuk items 1-8: ~6-7 jam kerja.

---

## Cara Melanjutkan

```bash
# 1. Verify binary masih OK
cd /home/arif/Project/testing-cli
make build
./bin/apitest --version

# 2. Run existing tests sebagai baseline
cd examples && ../bin/apitest run flows/smoke.flow --env dev -v

# 3. Pilih item dari backlog, implement, test
# 4. Build ulang
make build

# 5. Test dengan contoh .flow yang relevan
```

---

## File Penting untuk Reference

| File | Isi |
|---|---|
| `src/internal/runtime/engine.go` | Execution engine — tempat utama implementasi runtime |
| `src/internal/parser/parser.go` | Parser — sudah handle semua syntax |
| `src/internal/ast/ast.go` | AST node definitions |
| `src/internal/runtime/assertions.go` | Assertion evaluation |
| `src/internal/runtime/variables.go` | Variable scoping & interpolation |
| `src/internal/runtime/jsonpath.go` | JSONPath evaluator (simple) |
| `src/internal/runtime/helpers.go` | Duration/size parsing utilities |
| `docs/flowspec-dsl.md` | Spec DSL lengkap — reference syntax |
| `requirement.txt` | Requirement document — acceptance criteria |
| `examples/` | Example .flow files untuk testing |
