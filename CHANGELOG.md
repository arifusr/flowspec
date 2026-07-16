# Changelog

Semua perubahan penting pada project `apitest` didokumentasikan di file ini.

Format berdasarkan [Keep a Changelog](https://keepachangelog.com/id-ID/1.1.0/).

---

## [0.4.0] — 2026-07-16

### Added

- **`before` / `after` hooks in request definitions** — Execute logic before sending request and after receiving response:
  ```flow
  request CreateOrder {
    before {
      set request_id = "{{$uuid}}"
      log("Sending: {{request_id}}")
    }

    POST "{{base_url}}/orders"
    body json { ref: "{{request_id}}" }

    extract { order_id from json "$.id" }

    after {
      log("Created: id={{order_id}}, status={{last.status}}")
    }
  }
  ```

  Supported hook statements:
  | Statement | When | What it does |
  |-----------|------|--------------|
  | `set key = "value"` | before | Set variable before request (available in headers/body) |
  | `log("message")` | before/after | Print debug message with interpolation |
  | `script "path.js"` | (planned) | Execute external script |

  Execution order: `before` → send request → assertions → `extract` → `after`

### Changed

- Versi naik ke 0.4.0 (minor bump — new DSL construct)

---

## [0.3.5] — 2026-07-16

### Fixed

- **`{{last.status}}` and `{{last.body}}` now resolve inside `log()`** — Previously these rendered as literal strings. Now all `last.*` accessors work in string interpolation:
  ```flow
  step "Call API" {
    run GetUsers
    log("Status: {{last.status}}")                   // → "Status: 200"
    log("Name: {{last.json(\"$[0].name\")}}")       // → "Name: Leanne Graham"
  }
  ```
  No need for intermediate `let` variables anymore (though they still work).

### Changed

- Versi naik ke 0.3.5

---

## [0.3.4] — 2026-07-16

### Fixed

- **Multiple `run` commands in a single step** — Steps with multiple `run` + `let` now execute in declaration order. Each `let x = last.json(...)` captures data from the immediately preceding `run`, not just the last one in the step.
  ```flow
  step "Get dropdowns" {
    run GetCompany
    let company_id = last.json("$.data[0].id")    // ← from GetCompany

    run GetCountry
    let country_id = last.json("$.data[0].id")    // ← from GetCountry

    run GetCategory
    let category = last.json("$.data[0].id")      // ← from GetCategory
  }
  ```
  Previously only the last `run`'s response was accessible. Now all variables are captured correctly.

### Changed

- Versi naik ke 0.3.4

---

## [0.3.3] — 2026-07-16

### Fixed

- **`body from schema` fails with raw JSON data files** — Files containing `"items"` array (or any non-schema JSON) were incorrectly parsed as JSON Schema, causing unmarshal errors. Now `body from schema/file` auto-detects: if the file has `"type"` + `"properties"` → parse as JSON Schema; otherwise → load as raw JSON data template.

### Added

- **`body from file "path.json"`** — Explicit syntax to load raw JSON data as request body (alias for `body from schema` with auto-detection):
  ```flow
  body from file "data/payload.json" {
    set formulation_id = "{{my_id}}"
    set items[0].qty = "500"
  }
  ```

- **Variable interpolation in data templates** — `{{variable}}` placeholders inside JSON data files are now resolved at runtime.

### Changed

- Versi naik ke 0.3.3

---

## [0.3.2] — 2026-07-14

### Fixed

- **Numeric coercion bug** — String values di `body json { ... }` yang terlihat seperti angka (misalnya `"320301"`) tidak lagi dikonversi ke JSON number. Sebelumnya `material: "320301"` dikirim sebagai `{"material":320301}` (number), sekarang dikirim sebagai `{"material":"320301"}` (string). Fix ini juga diterapkan pada `set` overrides di `body from schema`.

  **Behavior baru:**
  | DSL | JSON terkirim |
  |-----|---------------|
  | `field: "320301"` | `"field": "320301"` (string) |
  | `active: true` | `"active": true` (boolean) |
  | `active: false` | `"active": false` (boolean) |

- **Cookie extraction** — Perbaikan parsing `Set-Cookie` header yang mengandung `=` di value (misalnya JWT token). Extraction sekarang menggunakan prefix matching (`name=`) bukan generic split.

### Changed

- Versi naik ke 0.3.2

---

## [0.3.1] — 2026-07-14

### Added

- **`write` statement** — Simpan response (atau bagian response) ke file. Output di-pretty-print otomatis.
  ```flow
  step "Get data" {
    run GetCompany
    write last.body to "reports/company-response.json"
    write last.json("$.data[0]") to "reports/first-company.json"
    write last.json("$[?(@.name=='PT ABC')].id") to "reports/abc-id.txt"
  }
  ```

  Variasi yang didukung:
  | Source | Hasil |
  |--------|-------|
  | `write last.body to "path"` | Full response body (pretty JSON) |
  | `write last.json("$.path") to "path"` | Subset JSON via JSONPath + filter |
  | `write last.header("Name") to "path"` | Header value |
  | `write last.status to "path"` | Status code |
  | `write "{{var}}" to "path"` | Variable value |

- **Append mode** — Tambahkan ke file tanpa overwrite:
  ```flow
  write last.body to "reports/all-responses.jsonl" append
  ```

### Changed

- Versi naik ke 0.3.1

---

## [0.3.0] — 2026-07-14

### Added

- **Schema-driven payload (`body from schema`)** — Generate request body dari JSON Schema file. Schema menggunakan `default` dan `example` values untuk menghasilkan payload lengkap.
  ```flow
  request StoreBom {
    POST "{{base_url}}/bom/product/store"
    body from schema "schemas/bom-product.schema.json"
    expect status 200
  }
  ```

- **Deep path override (`set`)** — Override field di generated payload, termasuk nested array:
  ```flow
  run StoreBom {
    body from schema "schemas/bom-product.schema.json" {
      set company = "{{company_id}}"
      set items[0].materialId = "312404"
      set items[0].components[0].qty = "500"
    }
  }
  ```

- **Schema response validation (`expect schema`)** — Validate response body terhadap JSON Schema:
  ```flow
  expect schema "schemas/user-response.schema.json"
  ```
  Error message menunjukkan violation spesifik (missing required field, wrong type, dll).

- **`$ref` resolution** — Schema bisa referensi schema lain via `$ref`, otomatis di-resolve:
  ```json
  { "items": { "type": "array", "items": { "$ref": "bom-item.schema.json" } } }
  ```

- **`apitest schema generate <file>`** — CLI command untuk preview generated JSON dari schema.

### Changed

- Versi naik ke 0.3.0

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

- **Parameterized request** — Request bisa menerima parameter dan dipanggil dengan literal atau variable:
  ```flow
  request GetUserRole(role_id) {
    GET "{{base_url}}/combo/user-role?q=&role={{role_id}}"
  }

  // Pemanggilan dari flow:
  step "Product role"   { run GetUserRole("8") }
  step "Packaging role" { run GetUserRole("9") }
  step "Dynamic"        { run GetUserRole(extracted_var) }
  ```

- **Query override di `run` block** — Override/tambah query parameter tanpa buat file request baru:
  ```flow
  step "Search" {
    run ListUsers {
      query role = "admin"
      query page = "2"
    }
  }
  ```

### Fixed

- **Flow tidak bisa resolve request by name** — Sebelumnya `run Login` dalam flow menghasilkan `unknown request 'Login'`. Sekarang semua request di `requests/` dan `shared/` otomatis ter-load.

- **`let` sebelum `run` tidak ter-resolve saat dipakai sebagai parameter** — `let x = "5"` lalu `run GetUser(x)` sekarang bekerja dengan benar.

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
