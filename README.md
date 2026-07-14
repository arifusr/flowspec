# apitest — CLI API Testing Tool

Alat testing API berbasis terminal. Definisikan request, skenario, dan assertion dalam **FlowSpec DSL** — bahasa khusus untuk domain API testing — jalankan dari CLI, dan integrasikan ke CI/CD. Tanpa GUI.

> Dokumen ini menjelaskan **cara menggunakan produk akhir** dari sudut pandang end user.
> - Spesifikasi teknis: [`requirement.txt`](requirement.txt)
> - Spesifikasi bahasa FlowSpec: [`docs/flowspec-dsl.md`](docs/flowspec-dsl.md)

---

## Daftar Isi

- [Siapa yang menggunakan tool ini?](#siapa-yang-menggunakan-tool-ini)
- [FlowSpec DSL — Bahasa Khusus API Testing](#flowspec-dsl--bahasa-khusus-api-testing)
- [Instalasi](#instalasi)
- [Mulai Cepat (5 menit)](#mulai-cepat-5-menit)
- [Konsep Inti](#konsep-inti)
- [Workflow Sehari-hari](#workflow-sehari-hari)
- [Membuat Request](#membuat-request)
- [Membuat Skenario](#membuat-skenario)
- [Environment & Variabel](#environment--variabel)
- [Assertion & Validasi](#assertion--validasi)
- [Melakukan Perubahan](#melakukan-perubahan)
- [Import dari Sumber Lain](#import-dari-sumber-lain)
- [Data-Driven Testing](#data-driven-testing)
- [Integrasi CI/CD](#integrasi-cicd)
- [Kolaborasi Tim & Version Control](#kolaborasi-tim--version-control)
- [Debugging & Troubleshooting](#debugging--troubleshooting)
- [Referensi Command](#referensi-command)

---

## Siapa yang menggunakan tool ini?

| Peran | Kegiatan utama |
|---|---|
| **Backend Developer** | Debug endpoint cepat, uji request sebelum commit |
| **QA Engineer** | Susun regression suite, jalankan smoke test |
| **DevOps / SRE** | Pasang test otomatis di pipeline deploy |
| **Tech Lead** | Review coverage test lewat report & Git diff |

---

## FlowSpec DSL — Bahasa Khusus API Testing

Produk ini tidak hanya "YAML runner generik". Inti authoring experience adalah **FlowSpec** — DSL yang dirancang khusus untuk domain HTTP API testing.

### Mengapa DSL?

```flow
// Terbaca seperti spesifikasi, bukan konfigurasi
flow UserCRUD {
  let user_email = "test-{{$uuid}}@example.com"

  step "Create user" { run CreateUser }
  step "Get user"    { when user_id; run GetUser(user_id) }
  step "Delete user" { run DeleteUser(user_id) }

  expect json "$.data.email" == user_email   // assertion natural language
}
```

Dibanding YAML generik:

| YAML | FlowSpec DSL |
|---|---|
| `assert: - jsonpath: "$.data.id" exists: true` | `expect json "$.data.id" exists` |
| `steps: - request: collections/users/create-user.yaml` | `step "Create user" { run CreateUser }` |
| Error: "invalid YAML line 14" | Error: "Unknown request 'GetUsers' — did you mean 'GetUser'?" |

### Building blocks FlowSpec

```
request   →  Satu panggilan HTTP + expect + extract   (unit, reusable)
flow      →  Orkestrasi multi-step + control flow       (scenario bisnis)
env       →  Variable per target deployment
auth      →  Fragment autentikasi reusable
fragment  →  Mixin step yang bisa di-`use` ulang
```

### Contoh side-by-side

**Request** (`requests/users/create-user.flow`):

```flow
request CreateUser {
  POST "{{base_url}}/api/v1/users"
  header Authorization = "Bearer {{access_token}}"

  body json {
    name:  "{{user_name}}"
    email: "{{user_email}}"
  }

  expect status 201
  expect json "$.data.email" == "{{user_email}}"
  expect time < 2s

  extract { user_id from json "$.data.id" }
}
```

**Scenario** (`flows/user-crud.flow`):

```flow
@env(dev)
@tags(crud, smoke)

flow UserCRUD {
  let user_email = "test-{{$uuid}}@example.com"

  step "Create user"     { run CreateUser }
  step "Get user detail" { when user_id; run GetUser(user_id) }
  step "Delete user"     { run DeleteUser(user_id) }

  teardown { ignore_fail; run CleanupOrphans }
}
```

**Jalankan:**

```bash
apitest run flows/user-crud.flow --env dev
apitest dsl lint flows/              # validasi syntax
apitest dsl show flows/user-crud.flow --env dev   # preview resolved
```

### Fitur DSL yang membuat produk terasa spesifik

- **`expect`** — assertion natural language (`expect status 200`, `expect time < 500ms`)
- **`extract`** — data flow eksplisit antar step (`extract { user_id from json "$.data.id" }`)
- **`run X { override }`** — reuse request dengan override inline, tanpa duplikasi file
- **`when` / `unless`** — conditional step native
- **`for row in csv(...)`** — data-driven testing first-class
- **`retry N times every Xs until ...`** — polling/assertion retry built-in
- **`extends`** — inheritance request
- **`@tags` / `@env`** — metadata deklaratif

Spesifikasi lengkap: [`docs/flowspec-dsl.md`](docs/flowspec-dsl.md) · Contoh: [`examples/`](examples/)

---

## Instalasi

```bash
# Opsi 1: Binary (rekomendasi)
curl -fsSL https://example.com/install.sh | sh

# Opsi 2: Package manager
brew install apitest        # macOS
apt install apitest           # Debian/Ubuntu

# Opsi 3: Python (development)
pip install apitest-cli

# Verifikasi
apitest --version
```

---

## Mulai Cepat (5 menit)

### 1. Buat project baru

```bash
mkdir my-api-tests && cd my-api-tests
apitest init
```

Perintah ini membuat struktur folder siap pakai:

```
my-api-tests/
├── apitest.flow          # Konfigurasi global project (FlowSpec)
├── requests/             # Request reusable (.flow)
├── flows/                # Skenario / orchestration (.flow)
├── env/                  # Environment (dev/staging/prod)
├── shared/               # Auth, fragment, mixin
├── data/                 # Dataset untuk data-driven test
├── scripts/              # Pre/post-request hooks (JS)
├── specs/                # OpenAPI spec (contract testing)
└── reports/              # Output report (auto-generated, gitignored)
```

### 2. Atur environment

Edit `env/dev.flow`:

```flow
env dev {
  base_url     = "http://localhost:8080"
  access_token = env("API_TOKEN")      // baca dari env var sistem
}
```

Set token di terminal:

```bash
export API_TOKEN="your-dev-token"
```

### 3. Buat request pertama

Buat file `requests/users/list-users.flow`:

```flow
@tags(users, smoke)

request ListUsers {
  GET "{{base_url}}/api/v1/users"
  header Authorization = "Bearer {{access_token}}"

  expect status 200
  expect json "$.data" is array
}
```

### 4. Jalankan

```bash
# Debug single request
apitest run requests/users/list-users.flow --env dev

# Validasi syntax DSL
apitest dsl lint requests/
```

Output contoh:

```
✓ List Users                              142ms
  GET http://localhost:8080/api/v1/users
  Status: 200 OK
  Assertions: 2 passed

Summary: 1 passed, 0 failed (142ms)
```

---

## Konsep Inti

Produk ini dibangun dari empat building block yang saling terhubung:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│     env     │────▶│   request   │────▶│    flow     │
│  (dev/prod) │     │   (unit)    │     │  (scenario) │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                    ┌──────▼──────┐
                    │  FlowSpec   │
                    │  Parser →   │
                    │  apitest run│
                    └─────────────┘
```

| Konsep | Analogi | File |
|---|---|---|
| **request** | Satu panggilan API + assertion | `requests/users/create-user.flow` |
| **flow** | Alur bisnis multi-step | `flows/user-crud.flow` |
| **env** | Target server + secrets | `env/staging.flow` |
| **auth / fragment** | Building block reusable | `shared/auth.flow` |

**Prinsip utama:** semua definisi test ditulis dalam **FlowSpec DSL** (`.flow`) — file teks yang bisa di-commit ke Git, di-review lewat PR, dan dijalankan identik di laptop maupun CI. YAML legacy tetap didukung via `apitest dsl migrate`.

---

## Workflow Sehari-hari

### Developer: debug endpoint baru

```bash
# Import dari cURL (copy dari browser DevTools / dokumentasi API)
apitest import curl 'curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"John\",\"email\":\"john@example.com\"}"' \
  --output collections/users/create-user.yaml

# Preview request setelah variable di-resolve
apitest request show collections/users/create-user.yaml --env dev

# Kirim request, lihat response mentah
apitest request send collections/users/create-user.yaml --env dev -v
```

### QA: jalankan smoke test sebelum release

```bash
apitest run scenarios/smoke-test.yaml --env staging --tags smoke
```

### DevOps: gate deploy di CI

```bash
apitest run scenarios/ --env staging \
  --reporters junit,json \
  --output reports/ \
  --fail-fast \
  --quiet
```

Exit code `0` = lolos, `1` = ada test gagal, `2` = error konfigurasi.

---

## Membuat Request

### Struktur file request

```yaml
name: Create User                    # Nama tampilan di report
description: Membuat user baru       # Opsional
method: POST
url: "{{base_url}}/api/v1/users"

headers:
  Content-Type: application/json
  Authorization: "Bearer {{access_token}}"

body:
  type: json
  content:
    name: "{{user_name}}"
    email: "{{user_email}}"

# Ambil data dari response → simpan ke variable
extract:
  user_id: "$.data.id"
  user_email: "$.data.email"

assert:
  - status: 201
  - jsonpath: "$.data.email"
    equals: "{{user_email}}"
  - response_time_lt: 2000           # Maks 2 detik

tags: [users, crud, smoke]
```

### Tipe body yang didukung

```yaml
# JSON
body:
  type: json
  content:
    key: value

# Form URL-encoded
body:
  type: form
  content:
    username: admin
    password: secret

# Multipart (upload file)
body:
  type: multipart
  content:
    name: John
    avatar:
      file: ./fixtures/avatar.png

# Raw text
body:
  type: raw
  content: "<xml>...</xml>"
  content_type: application/xml
```

### Path parameter & query string

```yaml
# URL dengan path param
url: "{{base_url}}/api/v1/users/{{user_id}}"

# Query params
url: "{{base_url}}/api/v1/users"
query:
  page: 1
  limit: 20
  sort: created_at
```

---

## Membuat Skenario

Skenario mengorkestrasi beberapa request menjadi alur bisnis utuh. Data mengalir antar step lewat variable yang di-`extract` dari response sebelumnya.

### Skenario CRUD sederhana

Buat `scenarios/user-crud-flow.yaml`:

```yaml
name: User CRUD Flow
description: Create → Read → Update → Delete user
env: dev

# Variable awal (override environment jika perlu)
variables:
  user_name: "Test User"
  user_email: "test-{{$uuid}}@example.com"

steps:
  - name: Create user
    request: collections/users/create-user.yaml
    # user_id otomatis tersedia dari extract di create-user.yaml

  - name: Get user detail
    request: collections/users/get-user.yaml
    if: "{{user_id}}"                # Skip jika create gagal

  - name: Update user
    request: collections/users/update-user.yaml

  - name: Delete user
    request: collections/users/delete-user.yaml

teardown:
  - name: Cleanup orphaned data
    request: collections/users/cleanup.yaml
    ignore_fail: true                # Teardown tidak fail-kan scenario
```

Jalankan:

```bash
apitest run scenarios/user-crud-flow.yaml --env dev
```

Output:

```
Scenario: User CRUD Flow
──────────────────────────────────────────────
  ✓ Step 1: Create user                   201  89ms
  ✓ Step 2: Get user detail               200  45ms
  ✓ Step 3: Update user                   200  52ms
  ✓ Step 4: Delete user                   204  38ms
  ✓ Teardown: Cleanup orphaned data       200  31ms

Summary: 5 passed, 0 failed (255ms)
```

### Skenario dengan kondisi & loop

```yaml
name: Bulk User Import
env: staging

steps:
  - name: Login admin
    request: collections/auth/login.yaml

  - name: Import each user from list
    for_each: "{{user_list}}"          # Variable berisi array
    steps:
      - request: collections/users/create-user.yaml
        variables:
          user_name: "{{item.name}}"
          user_email: "{{item.email}}"

  - name: Wait for async processing
    wait: 3000                         # Pause 3 detik

  - name: Verify total count
    request: collections/users/list-users.yaml
    assert:
      - jsonpath: "$.meta.total"
        gte: 10
```

### Skenario smoke test (kumpulan request paralel logic)

```yaml
name: Smoke Test
description: Quick sanity check — harus selesai < 30 detik
env: staging
tags: [smoke]

steps:
  - request: collections/health/check.yaml
  - request: collections/auth/login.yaml
  - request: collections/users/list-users.yaml
  - request: collections/products/list-products.yaml
  - request: collections/orders/list-orders.yaml
```

Jalankan hanya smoke test:

```bash
apitest run scenarios/ --env staging --tags smoke
```

---

## Environment & Variabel

### Hierarki variable (prioritas dari rendah ke tinggi)

```
Global (apitest.yaml)
  └── Environment (environments/staging.yaml)
        └── Collection (collections/users/folder.yaml)
              └── Scenario (scenarios/user-crud-flow.yaml)
                    └── CLI flag (--var key=value)
```

Variable di level lebih tinggi menimpa yang di bawahnya.

### Definisi environment

`environments/staging.yaml`:

```yaml
name: staging
variables:
  base_url: https://staging-api.example.com
  access_token: "{{$env.STAGING_API_TOKEN}}"
  admin_email: admin@staging.example.com
```

`environments/prod.yaml`:

```yaml
name: prod
variables:
  base_url: https://api.example.com
  access_token: "{{$env.PROD_API_TOKEN}}"
```

### Dynamic variables built-in

| Variable | Hasil |
|---|---|
| `{{$timestamp}}` | Unix timestamp saat ini |
| `{{$uuid}}` | UUID v4 random |
| `{{$randomEmail}}` | Email random untuk test |
| `{{$randomInt}}` | Integer random |
| `{{$env.VAR_NAME}}` | Baca dari environment variable OS |

### Secret management

```bash
# Jangan simpan secret di file YAML yang di-commit
# Gunakan env var sistem:

export STAGING_API_TOKEN="sk-staging-xxxxx"
apitest run scenarios/ --env staging

# Atau file .env lokal (gitignored):
echo "STAGING_API_TOKEN=sk-staging-xxxxx" >> .env
apitest run scenarios/ --env staging
```

Secret otomatis di-redact di log dan report:

```
Authorization: Bearer ***REDACTED***
```

---

## Assertion & Validasi

### Assertion dasar

```yaml
assert:
  # Status code
  - status: 200
  - status: [200, 201]               # Salah satu dari list
  - status_range: [200, 299]         # Range 2xx

  # Response time
  - response_time_lt: 1000           # Kurang dari 1 detik

  # Header
  - header: Content-Type
    contains: application/json

  # JSON body via JSONPath
  - jsonpath: "$.data.id"
    exists: true
  - jsonpath: "$.data.email"
    equals: "john@example.com"
  - jsonpath: "$.data.roles"
    type: array
  - jsonpath: "$.data.roles"
    length: 2
  - jsonpath: "$.message"
    matches: "^User created"
```

### Contract testing (validasi terhadap OpenAPI)

Import spec OpenAPI terlebih dahulu:

```bash
apitest import openapi specs/openapi.yaml --output collections/
```

Aktifkan contract validation di request:

```yaml
name: Get User
method: GET
url: "{{base_url}}/api/v1/users/{{user_id}}"
contract:
  spec: specs/openapi.yaml
  operation: getUserById           # operationId dari OpenAPI
  validate:
    status: true
    schema: true
    headers: false
```

Jika implementasi API tidak sesuai dokumentasi, test gagal dengan pesan jelas:

```
✗ Get User
  Contract violation: response field 'email' expected type 'string', got 'null'
  Spec: specs/openapi.yaml → getUserById → 200 → schema.properties.email
```

---

## Melakukan Perubahan

Semua perubahan dilakukan dengan mengedit file YAML — tidak perlu buka GUI.

### Menambah endpoint baru

```bash
# 1. Buat file request baru
cat > collections/orders/create-order.yaml << 'EOF'
name: Create Order
method: POST
url: "{{base_url}}/api/v1/orders"
...
EOF

# 2. Validasi syntax
apitest collection validate collections/orders/

# 3. Test manual
apitest request send collections/orders/create-order.yaml --env dev

# 4. Tambahkan ke scenario yang relevan
# Edit scenarios/order-flow.yaml, tambahkan step baru
```

### Mengubah assertion setelah API berubah

```yaml
# Sebelum (API return field "name")
- jsonpath: "$.data.name"
  equals: "{{user_name}}"

# Sesudah (API rename field ke "full_name")
- jsonpath: "$.data.full_name"
  equals: "{{user_name}}"
```

### Menambah step di scenario existing

Edit `scenarios/user-crud-flow.yaml`, sisipkan step:

```yaml
steps:
  - request: collections/users/create-user.yaml

  # Step baru: verifikasi email terkirim
  - name: Verify welcome email queued
    request: collections/notifications/check-email.yaml
    if: "{{user_id}}"

  - request: collections/users/get-user.yaml
  ...
```

### Override sementara tanpa edit file

```bash
# Ganti base URL untuk test lokal
apitest run scenarios/smoke-test.yaml \
  --env dev \
  --var base_url=http://localhost:3000

# Ganti data test
apitest run scenarios/user-crud-flow.yaml \
  --env dev \
  --var user_email=qa-test@example.com
```

### Workflow perubahan dengan Git

```bash
# Buat branch untuk perubahan test
git checkout -b test/add-order-scenario

# Edit file YAML
# ...

# Validasi sebelum commit
apitest collection validate .
apitest run scenarios/order-flow.yaml --env dev

# Commit & PR
git add collections/orders/ scenarios/order-flow.yaml
git commit -m "test: add order creation scenario"
git push origin test/add-order-scenario
```

---

## Import dari Sumber Lain

### Dari cURL (command langsung)

```bash
apitest import curl \
  'curl -X GET "https://api.example.com/users" -H "Authorization: Bearer xxx"' \
  --output requests/users/list-users.flow
```

### Dari cURL (file)

Simpan satu atau beberapa cURL command ke file teks, lalu import sekaligus:

`curls/user-api.txt`:

```bash
# List users
curl -X GET https://api.example.com/users \
  -H "Authorization: Bearer {{access_token}}" \
  -H "Accept: application/json"

# Create user
curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {{access_token}}" \
  -d '{"name":"John","email":"john@example.com"}'

# Get single user
curl https://api.example.com/users/1 \
  -H "Authorization: Bearer {{access_token}}"
```

Import seluruh file:

```bash
apitest import curl --file curls/user-api.txt --output-dir requests/users/
```

Output:

```
Found 3 curl command(s) in curls/user-api.txt

✓ Imported to requests/users/get-users.flow
  GET https://api.example.com/users
✓ Imported to requests/users/create-users.flow
  POST https://api.example.com/users
✓ Imported to requests/users/get1.flow
  GET https://api.example.com/users/1
```

**Format file cURL:**
- Satu command per blok, dipisahkan baris kosong
- Multi-line dengan backslash (`\`) continuation
- Komentar dengan `#` atau `//`
- Mendukung semua flag umum: `-X`, `-H`, `-d`, `--data-raw`, `-u` (basic auth)

Review dan sesuaikan file yang di-generate (tambah assertion, ganti hardcoded value ke `{{variable}}`).

### Dari OpenAPI spec (fase 2)

```bash
# Generate collection + contract assertions otomatis
apitest import openapi specs/openapi.yaml --output requests/generated/
```

### Dari Postman Collection (fase 2)

```bash
apitest import postman exports/my-api.postman_collection.json \
  --output requests/imported/
```

Variable Postman (`{{baseUrl}}`) dikonversi ke format `{{base_url}}`.

---

## Data-Driven Testing

Jalankan scenario yang sama dengan banyak dataset.

### Siapkan data

`data/users.csv`:

```csv
user_name,user_email,expected_role
Alice,alice@example.com,admin
Bob,bob@example.com,user
Charlie,charlie@example.com,user
```

### Scenario dengan data-driven

`scenarios/create-users-batch.yaml`:

```yaml
name: Create Users Batch
env: dev
data: data/users.csv              # Kolom CSV → variable otomatis

steps:
  - request: collections/users/create-user.yaml
    assert:
      - jsonpath: "$.data.role"
        equals: "{{expected_role}}"
```

Jalankan:

```bash
apitest run scenarios/create-users-batch.yaml --env dev
```

Output:

```
Scenario: Create Users Batch (3 iterations)
──────────────────────────────────────────────
  Iteration 1/3: Alice     ✓ 201  78ms
  Iteration 2/3: Bob       ✓ 201  65ms
  Iteration 3/3: Charlie   ✗ 422  42ms
    Assertion failed: $.data.role
    Expected: "user"
    Actual:   "guest"

Summary: 2 passed, 1 failed (185ms)
```

---

## Integrasi CI/CD

### GitHub Actions

`.github/workflows/api-test.yml`:

```yaml
name: API Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  api-test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Install apitest
        run: curl -fsSL https://example.com/install.sh | sh

      - name: Run smoke tests
        env:
          STAGING_API_TOKEN: ${{ secrets.STAGING_API_TOKEN }}
        run: |
          apitest run scenarios/ \
            --env staging \
            --tags smoke \
            --reporters junit,json \
            --output reports/ \
            --fail-fast \
            --quiet

      - name: Upload test report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: api-test-report
          path: reports/
```

### GitLab CI

`.gitlab-ci.yml`:

```yaml
api-test:
  stage: test
  image: alpine:latest
  before_script:
    - curl -fsSL https://example.com/install.sh | sh
  script:
    - apitest run scenarios/ --env staging --tags smoke,regression --reporters junit --output reports/ --quiet
  artifacts:
    when: always
    reports:
      junit: reports/junit.xml
    paths:
      - reports/
  variables:
    STAGING_API_TOKEN: $STAGING_API_TOKEN
```

### Strategi integrasi per trigger

| Trigger | Command | Tujuan |
|---|---|---|
| Setiap PR | `--tags smoke` | Cepat, < 1 menit |
| Merge ke `develop` | `--tags smoke,regression` | Coverage lebih luas |
| Sebelum deploy prod | `scenarios/` full suite | Gate deploy |
| Scheduled (cron) | `--env prod --tags monitoring` | Health check produksi |

### Pre-deploy gate

```bash
#!/bin/bash
# scripts/pre-deploy-check.sh

set -e

echo "Running API tests against staging..."
apitest run scenarios/ \
  --env staging \
  --reporters json \
  --output reports/ \
  --fail-fast \
  --quiet

if [ $? -eq 0 ]; then
  echo "✓ All API tests passed. Proceeding with deploy."
else
  echo "✗ API tests failed. Deploy aborted."
  exit 1
fi
```

---

## Kolaborasi Tim & Version Control

### Struktur repo yang direkomendasikan

```
my-api-tests/                   # Repo terpisah, atau folder di monorepo
├── apitest.yaml
├── collections/                # Di-commit ✓
├── scenarios/                  # Di-commit ✓
├── environments/
│   ├── dev.yaml                # Di-commit ✓ (tanpa secret)
│   ├── staging.yaml            # Di-commit ✓ (tanpa secret)
│   └── prod.yaml               # Di-commit ✓ (tanpa secret)
├── data/                       # Di-commit ✓
├── specs/openapi.yaml          # Di-commit ✓
├── scripts/                    # Di-commit ✓
├── reports/                    # .gitignore ✗
├── .env                        # .gitignore ✗
└── .gitignore
```

`.gitignore`:

```
reports/
.env
*.local.yaml
.apitest/history.db
```

### Konvensi penamaan

```
collections/<domain>/<action>.yaml
  users/create-user.yaml
  users/get-user.yaml
  orders/create-order.yaml

scenarios/<scope>-<purpose>.yaml
  smoke-test.yaml
  user-crud-flow.yaml
  order-checkout-flow.yaml
  regression-auth.yaml
```

### Tag strategy

| Tag | Kapan dijalankan |
|---|---|
| `smoke` | Setiap commit / PR |
| `regression` | Nightly / sebelum release |
| `crud` | Saat domain terkait berubah |
| `auth` | Saat auth module berubah |
| `slow` | Hanya manual / scheduled |

### Review checklist (PR)

Saat review perubahan test di Pull Request, periksa:

- [ ] Assertion memvalidasi behavior bisnis, bukan hanya status 200
- [ ] Tidak ada secret hardcoded di file YAML
- [ ] Variable `extract` di step N tersedia di step N+1
- [ ] Tag scenario sudah benar (`smoke` vs `regression`)
- [ ] `apitest collection validate .` lolos
- [ ] Test dijalankan lokal sebelum push

---

## Debugging & Troubleshooting

### Preview request sebelum dikirim

```bash
# Lihat URL, headers, body setelah variable di-resolve
apitest request show collections/users/create-user.yaml --env dev
```

### Verbose output

```bash
# Level 1: tampilkan request + response headers
apitest run scenarios/smoke-test.yaml --env dev -v

# Level 2: tampilkan full request + response body
apitest run scenarios/smoke-test.yaml --env dev -vv
```

### Simpan response ke file

```bash
apitest request send collections/users/list-users.yaml \
  --env dev \
  --output-dir ./debug/
# Hasil: ./debug/list-users-response.json
```

### History & replay

```bash
# Lihat 10 request terakhir
apitest history

# Replay request spesifik
apitest history replay abc123 --env dev
```

### Error umum

| Error | Penyebab | Solusi |
|---|---|---|
| `Variable 'user_id' not found` | Step sebelumnya gagal extract | Cek assertion step sebelumnya, tambahkan `if:` guard |
| `Connection refused` | Server tidak jalan / salah `base_url` | Cek `--env`, pastikan service up |
| `401 Unauthorized` | Token expired / salah | Cek `$API_TOKEN`, refresh token |
| `Schema validation failed` | Response API berubah | Update assertion atau laporkan breaking change |
| `Config error: invalid YAML` | Syntax error di file | Jalankan `apitest collection validate` |

---

## Referensi Command

```
apitest init                                          Buat project baru
apitest run <path> [--env] [--tags] [--var]           Jalankan test
apitest request send <file> [--env] [-v]              Kirim single request
apitest request show <file> [--env]                     Preview resolved request
apitest collection list                               List semua collection
apitest collection validate [path]                    Validasi syntax YAML
apitest env list                                      List environment
apitest import curl "<command>" --output <file>       Import dari cURL command
apitest import curl --file <path> --output-dir <dir>  Import dari cURL file
apitest import openapi <file> --output <dir>          Import OpenAPI spec
apitest import postman <file> --output <dir>          Import Postman collection
apitest history [replay <id>]                         Request history
apitest config show [--env]                           Tampilkan config efektif
apitest --version                                     Versi CLI
apitest help <command>                                Bantuan per command
```

### Flag `apitest run`

| Flag | Deskripsi |
|---|---|
| `--env <name>` | Pilih environment (dev/staging/prod) |
| `--var key=value` | Override variable |
| `--tags tag1,tag2` | Filter by tag |
| `--reporters json,junit,console` | Format output |
| `--output <dir>` | Direktori output report |
| `--fail-fast` | Stop pada failure pertama |
| `--timeout <ms>` | Global timeout |
| `--quiet` | Output minimal (untuk CI) |
| `-v` / `-vv` | Verbose output |
| `--no-color` | Disable colored output |

---

## Alur Lengkap: Dari Nol sampai CI

```
1. apitest init
       ↓
2. Import OpenAPI / buat request manual
       ↓
3. apitest request send ... (debug)
       ↓
4. Tambah assertion
       ↓
5. Susun scenario multi-step
       ↓
6. apitest run scenarios/ --env dev
       ↓
7. Commit ke Git
       ↓
8. Integrasi ke GitHub Actions / GitLab CI
       ↓
9. Scheduled monitoring (opsional)
```

---

## Dokumen Terkait

- [`docs/flowspec-dsl.md`](docs/flowspec-dsl.md) — Spesifikasi bahasa FlowSpec DSL
- [`examples/`](examples/) — Contoh file `.flow` siap pakai
- [`requirement.txt`](requirement.txt) — Spesifikasi fungsional & non-fungsional lengkap
- Roadmap implementasi — lihat section 8 di `requirement.txt`
