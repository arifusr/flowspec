# FlowSpec DSL — Domain Specific Language untuk API Testing

FlowSpec adalah bahasa khusus (DSL) yang dirancang **hanya untuk satu domain**: mendefinisikan, mengorkestrasi, dan memvalidasi HTTP API test.

Berbeda dengan YAML generik, setiap keyword FlowSpec memiliki makna di domain API testing — `request`, `flow`, `expect`, `extract`, `given env` — sehingga file test terbaca seperti spesifikasi bisnis, bukan konfigurasi infrastruktur.

---

## Mengapa DSL, Bukan YAML?

| Aspek | YAML generik | FlowSpec DSL |
|---|---|---|
| **Keterbacaan** | Banyak boilerplate key-value | Syntax deklaratif, mirip pseudo-code |
| **Domain fit** | Struktur data umum | Keyword native: `POST`, `expect`, `extract` |
| **Validasi error** | "invalid YAML" | "line 12: expect requires status or jsonpath" |
| **Composability** | Include/import manual | `use`, `import`, `extend request` first-class |
| **Refactoring** | Rename manual | `apitest dsl rename`, symbol-aware |
| **IDE support** | Schema YAML basic | Syntax highlight, autocomplete, lint khusus |
| **Identitas produk** | "YAML runner" | "FlowSpec — bahasa testing API" |

---

## Filosofi Desain

1. **Read like a spec, run like a program** — file `.flow` terbaca QA dan developer
2. **Progressive disclosure** — request sederhana = 5 baris; scenario kompleks = tambah block
3. **Explicit over magic** — data flow antar step terlihat jelas via `extract` / `let`
4. **Compile to IR** — DSL di-parse ke Intermediate Representation (JSON AST), dieksekusi engine yang sama
5. **YAML as fallback** — file YAML lama tetap didukung; `apitest compile` konversi ke FlowSpec

---

## File & Struktur Project

```
my-api-tests/
├── apitest.flow          # Konfigurasi global (DSL)
├── env/
│   ├── dev.flow
│   ├── staging.flow
│   └── prod.flow
├── requests/             # Request reusable (unit)
│   ├── users/
│   │   ├── list-users.flow
│   │   └── create-user.flow
│   └── auth/
│       └── login.flow
├── flows/                # Skenario / orchestration
│   ├── smoke.flow
│   ├── user-crud.flow
│   └── checkout.flow
├── shared/
│   ├── auth.flow         # Fragment & mixin
│   └── assertions.flow
└── data/
    └── users.csv
```

Ekstensi file: `.flow`

---

## Lexical Rules

```
# Comentar
// komentar satu baris
/* komentar
   multi baris */

# String
"double quoted"
'tsingle quoted'

# Interpolasi
{{base_url}}              # variable
{{$uuid}}                 # built-in dynamic
{{$env.API_TOKEN}}        # OS env var

# Identifier
user_id, CreateUser, smoke

# Literals
200, 201, true, false, 1500ms, 2s
```

---

## 1. Konfigurasi Project

File `apitest.flow` di root project:

```flow
project "My API Tests" {
  version = "1.0"
  default_env = dev

  env dev     from "env/dev.flow"
  env staging from "env/staging.flow"
  env prod    from "env/prod.flow"

  spec openapi from "specs/openapi.yaml"

  settings {
    timeout     = 30s
    fail_fast   = false
    redact      [Authorization, X-Api-Key, password]
    report_dir  = "reports/"
  }
}
```

---

## 2. Environment

File `env/dev.flow`:

```flow
env dev {
  base_url     = "http://localhost:8080"
  access_token = env("API_TOKEN")        // baca dari OS env var
  admin_email  = "admin@dev.local"
}
```

File `env/staging.flow`:

```flow
env staging {
  base_url     = "https://staging-api.example.com"
  access_token = env("STAGING_API_TOKEN")
}
```

Switch environment saat run:

```bash
apitest run flows/smoke.flow --env staging
```

---

## 3. Request Definition (Unit Test)

Request adalah **blok reusable** — satu panggilan HTTP + assertion.

### Request minimal

```flow
// requests/users/list-users.flow

@tags(users, smoke)

request ListUsers {
  GET "{{base_url}}/api/v1/users"

  header Authorization = "Bearer {{access_token}}"

  expect status 200
  expect json "$.data" is array
  expect time < 500ms
}
```

### Request dengan body & extract

```flow
// requests/users/create-user.flow

@tags(users, crud)

request CreateUser {
  POST "{{base_url}}/api/v1/users"

  header Content-Type = "application/json"
  header Authorization = "Bearer {{access_token}}"

  body json {
    name:  "{{user_name}}"
    email: "{{user_email}}"
  }

  expect status 201
  expect json "$.data.email" == "{{user_email}}"
  expect time < 2s

  extract {
    user_id    from json "$.data.id"
    user_email from json "$.data.email"
  }
}
```

### Request dengan parameter

```flow
// requests/users/get-user.flow

request GetUser(user_id) {
  GET "{{base_url}}/api/v1/users/{{user_id}}"

  header Authorization = "Bearer {{access_token}}"

  expect status 200
  expect json "$.data.id" == "{{user_id}}"
}
```

### Auth block (reusable)

```flow
// shared/auth.flow

auth BearerAuth {
  header Authorization = "Bearer {{access_token}}"
}

auth ApiKeyAuth {
  query api_key = "{{api_key}}"
}
```

Pemakaian di request:

```flow
request ListUsers {
  use auth BearerAuth

  GET "{{base_url}}/api/v1/users"
  expect status 200
}
```

---

## 4. Flow / Scenario (Orchestration)

Flow mengorkestrasi beberapa request menjadi alur bisnis.

### Flow linear

```flow
// flows/user-crud.flow

@tags(crud, regression)
@env(dev)

flow UserCRUD {
  description "Create → Read → Update → Delete user"

  let user_name  = "Test User"
  let user_email = "test-{{$uuid}}@example.com"

  step "Create user" {
    run CreateUser
  }

  step "Get user detail" {
    when user_id                    // skip jika variable tidak ada
    run GetUser(user_id)
  }

  step "Update user" {
    run UpdateUser(user_id) {
      body json { name: "Updated Name" }
    }
  }

  step "Delete user" {
    run DeleteUser(user_id)
  }

  teardown "Cleanup" {
    ignore_fail
    run CleanupOrphans
  }
}
```

### Override inline saat `run`

```flow
step "Create VIP user" {
  run CreateUser {
    body json {
      name:  "VIP User"
      email: "vip-{{$uuid}}@example.com"
      role:  "admin"
    }
    expect status 201
  }
}
```

Child block `{ ... }` **merge/override** definisi request asli — tidak perlu duplikasi file.

---

## 5. Expect — Assertion DSL

Syntax `expect` adalah core DSL — dibaca natural language:

```flow
# Status
expect status 200
expect status in [200, 201, 204]
expect status 2xx

# JSON body
expect json "$.data.id" exists
expect json "$.data.email" == "john@example.com"
expect json "$.data.roles" is array
expect json "$.data.roles" length 3
expect json "$.message" matches "^User created"
expect json "$.meta.total" >= 10

# Headers
expect header Content-Type contains "application/json"
expect header X-Request-Id exists

# Performance
expect time < 500ms
expect time <= 2s
expect size > 100 bytes
expect size < 1mb

# Contract (OpenAPI)
expect contract "getUserById" status
expect contract "getUserById" schema
```

Negasi:

```flow
expect json "$.error" not exists
expect status != 500
```

Soft assertion (lanjut meski gagal — fase 2):

```flow
expect soft json "$.data.optional_field" exists
```

---

## 6. Extract — Data Flow

```flow
extract {
  user_id   from json "$.data.id"
  token     from header "X-Auth-Token"
  session   from cookie "SESSION_ID"
  item_count from json "$.data" length
}
```

Atau inline setelah step:

```flow
step "Login" {
  run Login
  let access_token = last.json("$.token")
  let expires_at   = last.header("X-Token-Expires")
}
```

Keyword `last` merujuk response step terakhir — data flow eksplisit dan terbaca.

---

## 7. Control Flow

### Kondisi

```flow
step "Verify email" {
  when status_code == 201 && user_id
  run CheckEmailQueued(user_id)
}

step "Handle conflict" {
  unless status_code == 409
  run CreateUser
}
```

### Loop

```flow
flow BulkCreate {
  repeat 5 {
    let user_email = "bulk-{{$uuid}}@example.com"
    run CreateUser
  }
}

flow ImportFromCSV {
  for row in data("data/users.csv") {
    let user_name  = row.name
    let user_email = row.email
    run CreateUser
    expect json "$.data.role" == row.expected_role
  }
}
```

### Delay

```flow
step "Wait for async job" {
  wait 3s
}

step "Poll until ready" {
  retry 10 times every 2s until json "$.status" == "ready" {
    run GetJobStatus(job_id)
  }
}
```

### Parallel (fase 2)

```flow
step "Fetch all resources" {
  parallel {
    run ListUsers
    run ListProducts
    run ListOrders
  }
}
```

---

## 8. Composition & Reuse

### Import request dari file lain

```flow
import requests/users/create-user.flow
import requests/users/get-user.flow
import shared/auth.flow
```

### Extend request (inheritance)

```flow
request CreateAdminUser extends CreateUser {
  body json {
    role: "admin"
  }
  expect status 201
  expect json "$.data.role" == "admin"
}
```

### Fragment / mixin

```flow
// shared/common-steps.flow

fragment AuthenticatedSetup {
  step "Login" {
    run Login
    let access_token = last.json("$.token")
  }
}
```

Pemakaian:

```flow
flow OrderCheckout {
  use fragment AuthenticatedSetup

  step "Create order" {
    run CreateOrder
  }
}
```

### Include flow sebagai sub-flow

```flow
flow FullRegression {
  include flows/smoke.flow
  include flows/user-crud.flow
  include flows/order-checkout.flow
}
```

---

## 9. Hooks & Scripts

### Pre/post hook di level request

```flow
request CreateUser {
  before {
    set user_email = "{{$randomEmail}}"
    set timestamp  = "{{$timestamp}}"
  }

  POST "{{base_url}}/api/v1/users"
  body json { name: "Test", email: "{{user_email}}" }

  after {
    log("Created user: {{user_id}}")
  }

  expect status 201
  extract { user_id from json "$.data.id" }
}
```

### Script eksternal (JavaScript — kompatibel Postman)

```flow
request Login {
  before script "scripts/generate-signature.js"

  POST "{{base_url}}/auth/login"
  ...
}
```

---

## 10. Contract Testing

```flow
request GetUser {
  GET "{{base_url}}/api/v1/users/{{user_id}}"

  expect contract "getUserById" {
    status
    schema
    // headers  — optional
  }
}
```

Atau di level flow:

```flow
flow APIContractSuite {
  contract spec "specs/openapi.yaml"

  run GetUser
  run ListUsers
  run CreateUser
}
```

---

## 11. Data-Driven Testing

### Inline table

```flow
flow CreateUsersMatrix {
  for row in [
    { name: "Alice",   email: "alice@example.com",   role: "admin" },
    { name: "Bob",     email: "bob@example.com",     role: "user"  },
    { name: "Charlie", email: "charlie@example.com", role: "user"  }
  ] {
    let user_name  = row.name
    let user_email = row.email
    run CreateUser
    expect json "$.data.role" == row.role
  }
}
```

### External file

```flow
flow CreateUsersBatch {
  for row in csv("data/users.csv") {
    let user_name       = row.user_name
    let user_email      = row.user_email
    let expected_role   = row.expected_role
    run CreateUser
    expect json "$.data.role" == expected_role
  }
}
```

---

## 12. Perbandingan: YAML vs FlowSpec

### YAML (sebelum)

```yaml
name: User CRUD Flow
env: dev
variables:
  user_name: "Test User"
  user_email: "test-{{$uuid}}@example.com"
steps:
  - name: Create user
    request: collections/users/create-user.yaml
  - name: Get user detail
    request: collections/users/get-user.yaml
    if: "{{user_id}}"
  - request: collections/users/delete-user.yaml
teardown:
  - request: collections/users/cleanup.yaml
    ignore_fail: true
```

### FlowSpec (sesudah)

```flow
@env(dev)
@tags(crud)

flow UserCRUD {
  let user_name  = "Test User"
  let user_email = "test-{{$uuid}}@example.com"

  step "Create user"       { run CreateUser }
  step "Get user detail"   { when user_id; run GetUser(user_id) }
  step "Delete user"       { run DeleteUser(user_id) }

  teardown { ignore_fail; run CleanupOrphans }
}
```

Lebih ringkas, data flow terbaca, tidak perlu navigasi antar file untuk skenario sederhana.

---

## 13. CLI untuk DSL

```bash
# Jalankan file .flow
apitest run flows/user-crud.flow
apitest run flows/ --env staging --tags smoke

# Validasi syntax DSL (lint)
apitest dsl lint flows/
apitest dsl lint requests/users/create-user.flow

# Format / prettify
apitest dsl fmt flows/ --write

# Compile DSL → IR JSON (debug, CI cache)
apitest dsl compile flows/user-crud.flow --output build/user-crud.ir.json

# Konversi YAML legacy → FlowSpec
apitest dsl migrate collections/ --output requests/

# Preview resolved flow (dry run)
apitest dsl show flows/user-crud.flow --env dev

# Import cURL command → .flow
apitest import curl 'curl -X GET https://api.example.com/users' --output requests/get-users.flow

# Import dari file berisi banyak cURL commands
apitest import curl --file curls/api-calls.txt --output-dir requests/users/

# Autocomplete di shell (fase 2)
apitest dsl complete --shell bash >> ~/.bashrc
```

---

## 14. Error Messages (DSL-aware)

YAML generik:
```
Error: yaml.scanner.ScannerError: mapping values are not allowed here
  in "scenarios/user-crud.yaml", line 14, column 7
```

FlowSpec DSL:
```
Error [flows/user-crud.flow:18:5]
  Unknown request 'GetUsers' — did you mean 'GetUser'?
  Available: CreateUser, GetUser, UpdateUser, DeleteUser

Error [requests/create-user.flow:12:3]
  expect: 'status' expects integer or range, got string "ok"

Error [flows/checkout.flow:24:7]
  Variable 'order_id' used before assignment
  Hint: 'order_id' is extracted in step "Create order" (line 15) — ensure that step runs first
```

---

## 15. Architecture: Parse → IR → Execute

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  .flow file  │────▶│   Lexer +    │────▶│     IR       │────▶│   Runtime    │
│  (FlowSpec)  │     │   Parser     │     │   (JSON AST) │     │   Engine     │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                            │                    │
                     apitest dsl lint       apitest dsl compile
                     apitest dsl fmt        (cacheable di CI)
```

Semua format input (FlowSpec, YAML, Postman import) dikonvergi ke **IR yang sama** — engine executor tunggal, tidak duplikasi logic.

---

## 16. Grammar (EBNF Ringkas)

```ebnf
file        = { import | env | auth | request | flow | fragment } ;
import      = "import" string ;
env         = "env" IDENT "{" { assignment } "}" ;
request     = [ "@" tag_list ] "request" IDENT [ "(" params ")" ] [ "extends" IDENT ] block ;
flow        = [ "@" tag_list ] [ "@" env_tag ] "flow" IDENT block ;
block       = "{" { statement } "}" ;
statement   = step | let | expect | extract | run | wait | retry | for | repeat | teardown ;
step        = "step" string block ;
run         = "run" IDENT [ "(" args ")" ] [ override_block ] ;
expect      = "expect" expectation ;
let         = "let" IDENT "=" expression ;
```

Parser implementasi: ANTLR, Tree-sitter, atau hand-written recursive descent.

---

## 17. Roadmap DSL

| Fase | Fitur |
|---|---|
| **MVP** | Syntax request + expect + extract + flow linear |
| **Fase 2** | Import, extends, for/csv, retry/until, dsl lint/fmt |
| **Fase 3** | LSP (VS Code extension), parallel, soft expect, migrate YAML |
| **Fase 4** | REPL interaktif: `apitest dsl repl` |

---

## 18. Contoh Lengkap: E-Commerce Checkout

```flow
// flows/checkout.flow

@import
import requests/auth/login.flow
import requests/products/list-products.flow
import requests/orders/create-order.flow
import requests/orders/pay-order.flow

@tags(checkout, regression)
@env(staging)

flow CheckoutHappyPath {
  description "Login → browse → order → pay"

  // --- Setup ---
  let test_email = "checkout-{{$uuid}}@example.com"

  step "Register & login" {
    run Login {
      body json { email: test_email, password: "Test1234!" }
    }
    let access_token = last.json("$.token")
  }

  step "Browse products" {
    run ListProducts
    let product_id = last.json("$.data[0].id")
    expect json "$.data" length >= 1
  }

  step "Create order" {
    run CreateOrder {
      body json {
        product_id: product_id
        quantity:   2
      }
    }
    let order_id = last.json("$.data.id")
    expect status 201
  }

  step "Process payment" {
    run PayOrder(order_id) {
      body json {
        method: "credit_card"
        token:  "tok_test_{{$uuid}}"
      }
    }
    expect status 200
    expect json "$.data.status" == "paid"
  }

  step "Verify order status" {
    retry 5 times every 1s until json "$.data.status" == "confirmed" {
      run GetOrder(order_id)
    }
  }

  teardown "Cancel if unpaid" {
    ignore_fail
    when order_id
    run CancelOrder(order_id)
  }
}
```

Jalankan:

```bash
apitest run flows/checkout.flow --env staging -v
```

---

## Referensi Inspirasi

| Tool | DSL / Format | Yang diadopsi |
|---|---|---|
| Karate DSL | Gherkin-like | `request` + `match` pattern |
| REST Assured | Java fluent | `expect` chain |
| Hurl | Plain text HTTP | Syntax request ringkas |
| Gherkin/Cucumber | Given/When/Then | Readability untuk QA |
| Postman | JSON collection | Variable & script model |

FlowSpec menggabungkan **keterbacaan Gherkin**, **ringkasnya Hurl**, dan **orkestrasi Postman** — dalam satu bahasa kohesif.
