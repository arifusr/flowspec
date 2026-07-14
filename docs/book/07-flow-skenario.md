# Bab 7 — Flow & Skenario

**Estimasi waktu:** 30 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Beda `request` (unit) vs `flow` (scenario)
- Susun multi-step scenario
- `run`, override inline, teardown
- Tag & filter scenario

---

## 7.1 Request vs Flow

| | `request` | `flow` |
|---|---|---|
| **Isi** | 1 HTTP call | Banyak step |
| **Analogi** | Resep 1 hidangan | Menu lengkap |
| **File** | `requests/users/create-user.flow` | `flows/user-crud.flow` |
| **Jalankan** | `apitest run requests/...` | `apitest run flows/...` |

**Aturan praktis:**
- Buat **request** dulu (building block)
- Gabungkan jadi **flow** (scenario bisnis)

---

## 7.2 Flow Linear Sederhana

```flow
// flows/post-crud.flow

@import
import requests/posts/create-post.flow
import requests/posts/get-post.flow
import requests/posts/delete-post.flow

@tags(posts, crud)
@env(dev)

flow PostCRUD {
  description "Create → Read → Delete post"

  let post_title = "FlowSpec Tutorial {{$uuid}}"

  step "Create post" {
    run CreatePost
  }

  step "Get post" {
    when post_id
    run GetPost(post_id)
    expect json "$.title" == post_title
  }

  step "Delete post" {
    when post_id
    run DeletePost(post_id)
    expect status 204
  }
}
```

Jalankan:

```bash
apitest run flows/post-crud.flow --env dev
```

Output:

```
Scenario: PostCRUD
──────────────────────────────────────────
  ✓ Step 1: Create post     201  95ms
  ✓ Step 2: Get post        200  42ms
  ✓ Step 3: Delete post     204  38ms

Summary: 3 passed, 0 failed (175ms)
```

---

## 7.3 Anatomi Blok `flow`

```flow
@tags(smoke, users)          // metadata: filter tag
@env(staging)                // metadata: default env

flow NamaFlow {              // nama flow (PascalCase)
  description "..."          // opsional: deskripsi

  let var = "value"          // variable lokal flow

  step "Label step" {        // step dengan label human-readable
    run SomeRequest          // jalankan request
    expect ...               // assertion tambahan (opsional)
  }

  teardown "Cleanup" {       // opsional: jalan meski test gagal
    ignore_fail
    run CleanupRequest
  }
}
```

---

## 7.4 `run` — Menjalankan Request

### Run basic

```flow
step "List users" {
  run ListUsers
}
```

### Run dengan parameter

```flow
step "Get user 42" {
  run GetUser(42)
}

step "Get created user" {
  run GetUser(user_id)       // user_id dari extract step sebelumnya
}
```

### Run dengan override inline

Override body/assertion tanpa buat file request baru:

```flow
step "Create VIP user" {
  run CreateUser {
    body json {
      name:  "VIP Customer"
      email: "vip-{{$uuid}}@example.com"
      role:  "admin"
    }
    expect status 201
    expect json "$.data.role" == "admin"
  }
}
```

💡 **Tip:** Override inline = DRY (Don't Repeat Yourself). Satu `CreateUser`, banyak variasi.

---

## 7.5 Teardown — Cleanup Setelah Test

```flow
flow CreateOrder {
  step "Create" { run CreateOrder }

  teardown "Cancel unpaid order" {
    ignore_fail              // gagal cleanup tidak fail-kan scenario
    when order_id
    run CancelOrder(order_id)
  }
}
```

Teardown **selalu dijalankan** — meski ada step yang fail — mirip `finally` di programming.

---

## 7.6 Smoke Test — Flow dari Banyak Request

```flow
// flows/smoke.flow

@import
import requests/health/check.flow
import requests/auth/login.flow
import requests/users/list-users.flow
import requests/posts/list-posts.flow

@tags(smoke)
@env(staging)

flow SmokeTest {
  description "Quick sanity check — harus selesai < 30 detik"

  step "Health check"  { run HealthCheck }
  step "Login"         { run Login }
  step "List users"    { run ListUsers }
  step "List posts"    { run ListPosts }
}
```

Jalankan hanya smoke:

```bash
apitest run flows/ --env staging --tags smoke
```

---

## 7.7 Include Flow — Gabungkan Scenario

```flow
// flows/full-regression.flow

flow FullRegression {
  description "Semua test regression"

  include flows/smoke.flow
  include flows/user-crud.flow
  include flows/post-crud.flow
  include flows/order-checkout.flow
}
```

Satu perintah, jalankan semua:

```bash
apitest run flows/full-regression.flow --env staging
```

---

## 7.8 Organisasi File — Rekomendasi

```
flows/
├── smoke.flow              ← cepat, jalan setiap PR
├── user-crud.flow          ← domain users
├── order-checkout.flow     ← domain orders
└── full-regression.flow    ← gabungan semua

requests/
├── users/
│   ├── create-user.flow
│   ├── get-user.flow
│   └── delete-user.flow
├── orders/
│   └── ...
└── auth/
    └── login.flow
```

---

## 7.9 Workflow: Dari Request ke Flow

```
1. Buat request individual     → test satu-satu dulu
2. Pastikan extract benar      → variable mengalir
3. Susun flow linear           → 3-5 step
4. Tambah teardown             → cleanup
5. Tambah @tags                → filter
6. Commit & CI                 → automation
```

---

## Ringkasan Bab 7

| Syntax | Fungsi |
|---|---|
| `flow Name { ... }` | Definisi scenario |
| `step "Label" { ... }` | Satu langkah dalam flow |
| `run RequestName` | Jalankan request |
| `run Request(arg) { override }` | Run + override inline |
| `teardown { ignore_fail; ... }` | Cleanup |
| `@tags(...)` | Label filter |
| `include other.flow` | Gabung flow |

---

## Latihan Bab 7

**1.** Buat `flows/post-read.flow` — list posts → get post pertama (extract id) → get by id.

**2.** Buat `flows/smoke.flow` — minimal 3 request, tag `smoke`.

**3.** Tambah teardown di flow CRUD (cleanup mock).

**4.** Run `apitest run flows/ --tags smoke --env dev`.

---

**Lanjut →** [Bab 8 — Control Flow](08-control-flow.md)
