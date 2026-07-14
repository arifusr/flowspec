# Bab 12 — Project Lengkap

**Estimasi waktu:** 40 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Bangun test suite lengkap dari nol
- Terapkan semua konsep bab 1–11
- Struktur project production-ready
- Pattern yang umum di real project

---

## 12.1 Skenario: API E-Commerce

Kita akan buat test suite untuk API e-commerce sederhana dengan endpoint:

| Method | Endpoint | Fungsi |
|---|---|---|
| POST | `/auth/register` | Register user baru |
| POST | `/auth/login` | Login, dapat token |
| GET | `/products` | List semua produk |
| GET | `/products/:id` | Detail produk |
| POST | `/orders` | Buat order |
| GET | `/orders/:id` | Detail order |
| POST | `/orders/:id/pay` | Bayar order |
| DELETE | `/orders/:id` | Cancel order |

---

## 12.2 Struktur Project

```
ecommerce-tests/
├── apitest.flow
├── .gitignore
├── env/
│   ├── dev.flow
│   └── staging.flow
├── shared/
│   ├── auth.flow
│   └── common-steps.flow
├── requests/
│   ├── auth/
│   │   ├── register.flow
│   │   └── login.flow
│   ├── products/
│   │   ├── list-products.flow
│   │   └── get-product.flow
│   └── orders/
│       ├── create-order.flow
│       ├── get-order.flow
│       ├── pay-order.flow
│       └── cancel-order.flow
├── flows/
│   ├── smoke.flow
│   ├── auth-flow.flow
│   ├── order-checkout.flow
│   ├── order-cancel.flow
│   └── regression.flow
├── data/
│   └── products.csv
├── reports/
└── .github/
    └── workflows/
        └── api-test.yml
```

---

## 12.3 Konfigurasi Global

```flow
// apitest.flow

project "E-Commerce API Tests" {
  version     = "1.0"
  default_env = dev

  env dev     from "env/dev.flow"
  env staging from "env/staging.flow"

  settings {
    timeout    = 30s
    fail_fast  = false
    redact     [Authorization, X-Api-Key, password]
    report_dir = "reports/"
  }
}
```

---

## 12.4 Environment Files

```flow
// env/dev.flow

env dev {
  base_url      = "http://localhost:3000"
  admin_email   = "admin@dev.local"
  admin_password = env("ADMIN_PASSWORD")
}
```

```flow
// env/staging.flow

env staging {
  base_url      = "https://staging-api.ecommerce.com"
  admin_email   = "admin@staging.ecommerce.com"
  admin_password = env("STAGING_ADMIN_PASSWORD")
}
```

---

## 12.5 Shared: Auth Block & Fragment

```flow
// shared/auth.flow

auth BearerAuth {
  header Authorization = "Bearer {{access_token}}"
}
```

```flow
// shared/common-steps.flow

import requests/auth/login.flow

fragment LoginAsAdmin {
  step "Login as admin" {
    run Login {
      body json {
        email:    "{{admin_email}}"
        password: "{{admin_password}}"
      }
    }
    let access_token = last.json("$.token")
    let current_user_id = last.json("$.user.id")
  }
}

fragment LoginAsNewUser {
  step "Register new user" {
    run Register {
      body json {
        name:     "Test User {{$uuid}}"
        email:    "test-{{$uuid}}@example.com"
        password: "TestPass123!"
      }
    }
    let user_email = last.json("$.data.email")
  }

  step "Login as new user" {
    run Login {
      body json {
        email:    user_email
        password: "TestPass123!"
      }
    }
    let access_token = last.json("$.token")
    let current_user_id = last.json("$.user.id")
  }
}
```

---

## 12.6 Request Files

### Auth

```flow
// requests/auth/register.flow

request Register {
  POST "{{base_url}}/auth/register"

  header Content-Type = "application/json"

  body json {
    name:     "{{user_name}}"
    email:    "{{user_email}}"
    password: "{{user_password}}"
  }

  expect status 201
  expect json "$.data.email" == "{{user_email}}"
  expect time < 3s

  extract {
    user_id    from json "$.data.id"
    user_email from json "$.data.email"
  }
}
```

```flow
// requests/auth/login.flow

request Login {
  POST "{{base_url}}/auth/login"

  header Content-Type = "application/json"

  body json {
    email:    "{{login_email}}"
    password: "{{login_password}}"
  }

  expect status 200
  expect json "$.token" exists
  expect time < 2s

  extract {
    access_token from json "$.token"
    user_id      from json "$.user.id"
  }
}
```

### Products

```flow
// requests/products/list-products.flow

request ListProducts {
  GET "{{base_url}}/products"

  use auth BearerAuth

  expect status 200
  expect json "$.data" is array
  expect json "$.data" length >= 1
  expect time < 1s
}
```

```flow
// requests/products/get-product.flow

request GetProduct(product_id) {
  GET "{{base_url}}/products/{{product_id}}"

  use auth BearerAuth

  expect status 200
  expect json "$.data.id" == "{{product_id}}"
  expect json "$.data.name" exists
  expect json "$.data.price" is number
}
```

### Orders

```flow
// requests/orders/create-order.flow

request CreateOrder {
  POST "{{base_url}}/orders"

  use auth BearerAuth
  header Content-Type = "application/json"

  body json {
    product_id: "{{product_id}}"
    quantity:   "{{order_quantity}}"
  }

  expect status 201
  expect json "$.data.id" exists
  expect json "$.data.status" == "pending"
  expect time < 2s

  extract {
    order_id     from json "$.data.id"
    order_total  from json "$.data.total"
    order_status from json "$.data.status"
  }
}
```

```flow
// requests/orders/get-order.flow

request GetOrder(order_id) {
  GET "{{base_url}}/orders/{{order_id}}"

  use auth BearerAuth

  expect status 200
  expect json "$.data.id" == "{{order_id}}"
}
```

```flow
// requests/orders/pay-order.flow

request PayOrder(order_id) {
  POST "{{base_url}}/orders/{{order_id}}/pay"

  use auth BearerAuth
  header Content-Type = "application/json"

  body json {
    method: "credit_card"
    token:  "tok_test_{{$uuid}}"
  }

  expect status 200
  expect json "$.data.status" == "paid"
  expect time < 5s

  extract {
    payment_id from json "$.data.payment_id"
  }
}
```

```flow
// requests/orders/cancel-order.flow

request CancelOrder(order_id) {
  DELETE "{{base_url}}/orders/{{order_id}}"

  use auth BearerAuth

  expect status 200
  expect json "$.data.status" == "cancelled"
}
```

---

## 12.7 Flow Files

### Smoke Test

```flow
// flows/smoke.flow

import shared/auth.flow
import shared/common-steps.flow
import requests/products/list-products.flow

@tags(smoke)

flow SmokeTest {
  description "Sanity check — login + list products"

  use fragment LoginAsAdmin

  step "List products" {
    run ListProducts
    expect json "$.data" length >= 1
  }
}
```

### Auth Flow

```flow
// flows/auth-flow.flow

import shared/auth.flow
import requests/auth/register.flow
import requests/auth/login.flow

@tags(auth, regression)

flow AuthFlow {
  description "Register → Login → Verify token"

  let user_name     = "Auth Test {{$uuid}}"
  let user_email    = "auth-{{$uuid}}@example.com"
  let user_password = "SecurePass123!"

  step "Register new user" {
    run Register
    expect status 201
  }

  step "Login with new credentials" {
    run Login {
      body json {
        email:    user_email
        password: user_password
      }
    }
    let access_token = last.json("$.token")
    expect json "$.token" exists
  }

  step "Verify token works" {
    run ListProducts
    expect status 200
  }
}
```

### Order Checkout (Happy Path)

```flow
// flows/order-checkout.flow

import shared/auth.flow
import shared/common-steps.flow
import requests/products/list-products.flow
import requests/products/get-product.flow
import requests/orders/create-order.flow
import requests/orders/get-order.flow
import requests/orders/pay-order.flow
import requests/orders/cancel-order.flow

@tags(orders, checkout, regression)
@env(staging)

flow OrderCheckout {
  description "Login → Browse → Order → Pay → Verify"

  // --- Setup: Login ---
  use fragment LoginAsNewUser

  // --- Browse products ---
  step "List products" {
    run ListProducts
    let product_id = last.json("$.data[0].id")
    expect json "$.data" length >= 1
  }

  step "Get product detail" {
    when product_id
    run GetProduct(product_id)
    let product_price = last.json("$.data.price")
  }

  // --- Create order ---
  step "Create order" {
    when product_id
    let order_quantity = "2"
    run CreateOrder
    expect status 201
    expect json "$.data.status" == "pending"
  }

  step "Verify order created" {
    when order_id
    run GetOrder(order_id)
    expect json "$.data.product_id" == product_id
    expect json "$.data.quantity" == 2
  }

  // --- Payment ---
  step "Pay order" {
    when order_id
    run PayOrder(order_id)
    expect status 200
    expect json "$.data.status" == "paid"
  }

  step "Verify order paid" {
    when order_id
    retry 5 times every 1s until json "$.data.status" == "confirmed" {
      run GetOrder(order_id)
    }
  }

  // --- Teardown ---
  teardown "Cancel if not paid" {
    ignore_fail
    when order_id
    run CancelOrder(order_id)
  }
}
```

### Order Cancel

```flow
// flows/order-cancel.flow

import shared/auth.flow
import shared/common-steps.flow
import requests/products/list-products.flow
import requests/orders/create-order.flow
import requests/orders/get-order.flow
import requests/orders/cancel-order.flow

@tags(orders, cancel, regression)

flow OrderCancel {
  description "Create order → Cancel → Verify cancelled"

  use fragment LoginAsNewUser

  step "List products" {
    run ListProducts
    let product_id = last.json("$.data[0].id")
  }

  step "Create order" {
    when product_id
    let order_quantity = "1"
    run CreateOrder
  }

  step "Cancel order" {
    when order_id
    run CancelOrder(order_id)
    expect json "$.data.status" == "cancelled"
  }

  step "Verify cancelled" {
    when order_id
    run GetOrder(order_id)
    expect json "$.data.status" == "cancelled"
  }
}
```

### Full Regression

```flow
// flows/regression.flow

@tags(regression)

flow FullRegression {
  description "Semua scenario regression"

  include flows/smoke.flow
  include flows/auth-flow.flow
  include flows/order-checkout.flow
  include flows/order-cancel.flow
}
```

---

## 12.8 CI/CD Configuration

```yaml
# .github/workflows/api-test.yml

name: E-Commerce API Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install apitest
        run: curl -fsSL https://example.com/install.sh | sh
      - name: Lint
        run: apitest dsl lint .

  smoke-test:
    needs: lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install apitest
        run: curl -fsSL https://example.com/install.sh | sh
      - name: Smoke tests
        env:
          ADMIN_PASSWORD: ${{ secrets.STAGING_ADMIN_PASSWORD }}
        run: apitest run flows/smoke.flow --env staging --report junit
      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: smoke-report
          path: reports/

  regression:
    needs: smoke-test
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install apitest
        run: curl -fsSL https://example.com/install.sh | sh
      - name: Full regression
        env:
          STAGING_ADMIN_PASSWORD: ${{ secrets.STAGING_ADMIN_PASSWORD }}
        run: apitest run flows/regression.flow --env staging --report junit,html
      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: regression-report
          path: reports/
```

---

## 12.9 .gitignore

```gitignore
# Reports (auto-generated)
reports/

# Secrets
.env
*.local.flow

# OS files
.DS_Store
Thumbs.db
```

---

## 12.10 Menjalankan Project

```bash
# Development (lokal)
apitest dsl lint .
apitest run flows/smoke.flow --env dev
apitest run flows/order-checkout.flow --env dev -v

# Staging (sebelum deploy ke prod)
apitest run flows/regression.flow --env staging --report html

# Hanya test tertentu
apitest run flows/ --env staging --tags orders
apitest run flows/ --env staging --tags smoke
```

---

## 12.11 Checklist Production-Ready

Sebelum project dianggap "selesai":

- [ ] Semua request punya minimal `expect status`
- [ ] Flow punya teardown untuk cleanup
- [ ] Secret via `env()`, bukan hardcode
- [ ] `.gitignore` ada dan benar
- [ ] CI pipeline: lint → smoke → regression
- [ ] Report artifact di-upload
- [ ] README di root project menjelaskan cara run
- [ ] Tag `smoke` di request kritis (≤ 2 menit)
- [ ] Tag `regression` di semua flow

---

## 12.12 Evolusi Project

Setelah fondasi siap, kamu bisa menambah:

| Fase | Tambahan |
|---|---|
| **Bulan 1** | Smoke + CRUD flow utama |
| **Bulan 2** | Data-driven (CSV negative cases) |
| **Bulan 3** | Contract testing (OpenAPI) |
| **Bulan 4** | Performance baseline (expect time) |
| **Bulan 5** | Multi-environment (dev/staging/prod) |
| **Bulan 6** | Custom scripts + webhook notification |

---

## Ringkasan Bab 12

Project FlowSpec production-ready terdiri dari:

1. **`apitest.flow`** — config global
2. **`env/`** — variable per environment
3. **`shared/`** — auth block & fragment
4. **`requests/`** — building block per endpoint
5. **`flows/`** — scenario bisnis
6. **`data/`** — CSV untuk data-driven
7. **CI/CD** — lint → smoke → regression
8. **Reports** — artifact untuk review

---

## Latihan Bab 12

**1.** Buat project FlowSpec dari nol untuk API yang kamu kenal (bisa JSONPlaceholder).

**2.** Minimal: 3 request, 1 flow smoke, 1 flow CRUD dengan teardown.

**3.** Buat file CI (GitHub Actions atau GitLab CI) — lint + smoke.

**4.** Jalankan `--report html`, buka report dan review.

---

**Lanjut →** [Bab 13 — Jawaban Latihan](13-jawaban-latihan.md)
