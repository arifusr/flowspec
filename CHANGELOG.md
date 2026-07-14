# Changelog

Semua perubahan penting pada project `apitest` didokumentasikan di file ini.

Format berdasarkan [Keep a Changelog](https://keepachangelog.com/id-ID/1.1.0/).

---

## [0.2.0] — 2026-07-14

### Added

- **`log()` statement** — Print debug message ke console saat step dijalankan. Berguna untuk inspect variable dan data yang di-extract.
  ```flow
  step "Login" {
    run Login
    log("Token: {{access_token}}")
  }
  ```

- **JSONPath filter expression `$[?(@.field=='value')]`** — Search/filter di array JSON. Temukan item berdasarkan field value, lalu extract data darinya.
  ```flow
  let company_id = last.json("$.data[?(@.name=='PT ABC')].id")
  ```
  Operator yang didukung: `==`, `!=`, `>`, `>=`, `<`, `<=`.

- **`let x = last.json("$.path")`** — Extract value dari response terakhir langsung di step, termasuk support filter expression.
  ```flow
  step "Get users" {
    run GetUsers
    let admin = last.json("$[?(@.role=='admin')].name")
    log("Admin: {{admin}}")
  }
  ```

- **`let x = last.header("Name")`** — Extract response header inline di step.

- **`let x = last.status`** — Ambil status code response terakhir sebagai variable.

- **Auto-discovery request** — Flow otomatis menemukan request by name dari folder `requests/` dan `shared/` tanpa perlu `import` statement.

- **Project config directory loading** — `apitest` membaca `apitest.flow` dan memuat directory yang dideklarasikan (misal `requests from "custom-path/"`).

### Fixed

- **Flow tidak bisa resolve request by name** — Sebelumnya `run Login` dalam flow menghasilkan `unknown request 'Login'` meskipun request valid. Sekarang semua request di `requests/` dan `shared/` otomatis ter-load.

### Changed

- Versi naik ke 0.2.0

---

## [0.1.1] — 2026-07-14

### Fixed

- **Flow tidak bisa resolve request by name** — Sebelumnya, menjalankan flow yang mereferensi request dengan `run Login` menghasilkan error `unknown request 'Login'` meskipun request valid dan bisa dijalankan standalone. Sekarang `apitest` otomatis memuat semua request dari folder `requests/` dan `shared/` saat menjalankan flow. `import` statement tidak lagi wajib untuk project dengan struktur standar.

### Added

- **Auto-discovery request** — Flow bisa langsung `run NamaRequest` tanpa menulis `import` statement, selama request berada di folder `requests/` atau `shared/` dalam project.
- **Project config directory loading** — `apitest` membaca deklarasi directory di `apitest.flow` (misal `requests from "custom-path/"`) dan memuat file dari directory tersebut.
- **`import curl --file`** — Import multiple cURL commands dari satu file teks. Commands dipisahkan baris kosong, mendukung multi-line dengan backslash continuation, dan komentar `#`.
- **`import curl --output-dir`** — Tentukan output directory saat import dari file.
- **Link dokumentasi di binary** — `apitest --version` dan `apitest help` menampilkan link ke https://github.com/arifusr/flowspec.
- **`install.sh`** — Script install otomatis: build dari source dan copy ke `~/.local/bin` atau `/usr/local/bin`.

### Changed

- **Help text `apitest help run`** — Menambahkan section "Auto-discovery" yang menjelaskan bahwa requests di-load otomatis.
- **Dokumentasi tutorial (Bab 2, 7, 9)** — Update penjelasan tentang auto-discovery dan kapan `import` masih diperlukan.

---

## [0.1.0] — 2026-07-13

### Added — Initial Release (MVP)

**Core:**
- FlowSpec DSL parser (lexer + recursive descent parser)
- AST (Abstract Syntax Tree) untuk semua konstruk: request, flow, env, auth, fragment
- HTTP request engine (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
- Environment & variable system dengan scope hierarchy
- Variable interpolation `{{var}}` dan dynamic vars (`{{$uuid}}`, `{{$timestamp}}`, `{{$randomEmail}}`, `{{$randomInt}}`)
- `.env` file auto-loading

**Assertions (`expect`):**
- Status code: exact, range (`2xx`), list (`in [200, 201]`), negation (`!= 500`)
- JSON body via JSONPath: `==`, `!=`, `exists`, `not exists`, `is` (type check), `length`, `>=`, `<=`, `>`, `<`, `matches`, `contains`
- Response headers: `exists`, `contains`, `matches`, `==`
- Response time: `< 500ms`, `<= 2s`
- Response size: `> 100 bytes`, `< 1mb`

**Extract & Data Flow:**
- `extract { var from json "$.path" }` — dari JSON body
- `extract { var from header "Name" }` — dari response header
- `extract { var from cookie "NAME" }` — dari cookie
- Variable chaining antar step dalam flow

**Flow Execution:**
- Multi-step linear execution
- `when` / `unless` conditional step (skip jika kondisi tidak terpenuhi)
- `let` variable declarations dalam flow
- `run RequestName` — execute request by name
- `run RequestName(arg)` — pass parameter
- `run RequestName { body json { ... } }` — override inline (merge body/headers/expects)
- `teardown { ignore_fail; run X }` — cleanup yang selalu jalan

**CLI Commands:**
- `apitest init` — scaffold project baru
- `apitest run <path>` — execute request/flow/directory
- `apitest run --env <name>` — switch environment
- `apitest run --var key=value` — override variable dari CLI
- `apitest run --tags tag1,tag2` — filter by tag
- `apitest run --fail-fast` — stop on first failure
- `apitest run --report json,junit` — generate report files
- `apitest run -v / -vv` — verbose output
- `apitest run -q` — quiet mode (CI)
- `apitest run --no-color` — disable ANSI colors
- `apitest dsl lint <path>` — validate FlowSpec syntax
- `apitest dsl show <file> --env <name>` — dry run preview (resolved variables)
- `apitest import curl '<command>' --output <file>` — import cURL command
- `apitest help <command>` — built-in help
- `apitest --version` — show version

**Reporters:**
- Console reporter (colored, verbose levels, quiet mode)
- JSON report file (`reports/report-<timestamp>.json`)
- JUnit XML report file (`reports/report-<timestamp>.xml`)

**CI/CD:**
- Exit code 0 = all pass, 1 = failure, 2 = config error
- `--quiet` + `--report junit` untuk pipeline integration

---

## Roadmap

Lihat [BACKLOG.md](BACKLOG.md) untuk daftar fitur yang direncanakan:
- `repeat` / `for` / `retry` loop execution
- `extends` request inheritance
- `fragment` execution
- Data-driven testing (CSV)
- OpenAPI & Postman import
- `dsl fmt` formatter
- HTML report
- Request history
- OAuth 2.0
