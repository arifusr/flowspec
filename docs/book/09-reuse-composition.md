# Bab 9 — Reuse & Composition

**Estimasi waktu:** 25 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Import request dari file lain
- Extends — inheritance request
- Fragment / mixin — potongan step reusable
- Auth block — satu kali tulis, pakai di mana-mana
- Override inline vs extends

---

## 9.1 Masalah: Duplikasi

Tanpa mekanisme reuse, kamu akan **copy-paste** header Authorization di setiap request:

```flow
// ✗ Duplikasi di mana-mana
request ListUsers {
  GET "{{base_url}}/api/v1/users"
  header Authorization = "Bearer {{access_token}}"
  header Accept = "application/json"
  expect status 200
}

request GetUser(user_id) {
  GET "{{base_url}}/api/v1/users/{{user_id}}"
  header Authorization = "Bearer {{access_token}}"   // copy-paste
  header Accept = "application/json"                 // copy-paste
  expect status 200
}
```

Kalau token format berubah → edit **semua** file. FlowSpec punya solusi: `auth`, `extends`, `fragment`, dan `import`.

---

## 9.2 Import — Gunakan Request dari File Lain

Agar flow bisa `run` request, file request harus di-import:

```flow
// flows/user-crud.flow

import requests/auth/login.flow
import requests/users/create-user.flow
import requests/users/get-user.flow
import requests/users/delete-user.flow

flow UserCRUD {
  step "Login"  { run Login }
  step "Create" { run CreateUser }
  step "Get"    { run GetUser(user_id) }
  step "Delete" { run DeleteUser(user_id) }
}
```

**Rules:**
- Path relatif dari root project
- Satu `import` per baris
- Boleh import folder: `import requests/users/` (semua file `.flow` di dalamnya)

---

## 9.3 Auth Block — Header Reusable

Buat file `shared/auth.flow`:

```flow
// shared/auth.flow

auth BearerAuth {
  header Authorization = "Bearer {{access_token}}"
}

auth ApiKeyAuth {
  header X-Api-Key = "{{api_key}}"
}

auth BasicAuth {
  header Authorization = "Basic {{$base64(username:password)}}"
}
```

Pakai di request:

```flow
import shared/auth.flow

request ListUsers {
  use auth BearerAuth

  GET "{{base_url}}/api/v1/users"
  expect status 200
}

request GetUser(user_id) {
  use auth BearerAuth

  GET "{{base_url}}/api/v1/users/{{user_id}}"
  expect status 200
}
```

Sekarang token format berubah → edit **satu file** saja (`shared/auth.flow`).

---

## 9.4 Extends — Inheritance Request

Buat request baru berdasarkan request yang sudah ada:

```flow
// requests/users/create-user.flow
request CreateUser {
  POST "{{base_url}}/api/v1/users"

  use auth BearerAuth

  body json {
    name:  "{{user_name}}"
    email: "{{user_email}}"
    role:  "user"
  }

  expect status 201
  extract { user_id from json "$.data.id" }
}
```

Buat variasi tanpa duplikasi:

```flow
// requests/users/create-admin.flow

request CreateAdmin extends CreateUser {
  body json {
    role: "admin"                 // override field role saja
  }

  expect json "$.data.role" == "admin"   // tambah assertion
}
```

**Cara kerja extends:**
- Method, URL, header → inherit dari parent
- `body json` → **merge** (field baru ditambah, field sama di-override)
- `expect` → **append** (assertion parent tetap jalan + assertion baru)
- `extract` → inherit, bisa override

---

## 9.5 Extends — Contoh Lebih Kompleks

```flow
// Base request
request CreatePost {
  POST "{{base_url}}/posts"
  use auth BearerAuth

  body json {
    title:  "{{post_title}}"
    body:   "{{post_body}}"
    userId: 1
  }

  expect status 201
  extract { post_id from json "$.id" }
}

// Variasi: post draft (status = draft)
request CreateDraftPost extends CreatePost {
  body json {
    status: "draft"
  }
  expect json "$.status" == "draft"
}

// Variasi: post published
request CreatePublishedPost extends CreatePost {
  body json {
    status: "published"
  }
  expect json "$.status" == "published"
}
```

---

## 9.6 Fragment — Potongan Step Reusable

Fragment = **sekelompok step** yang bisa dipakai di banyak flow.

```flow
// shared/common-steps.flow

fragment AuthenticatedSetup {
  step "Login" {
    run Login
    let access_token = last.json("$.token")
  }

  step "Verify token valid" {
    run GetProfile
    expect status 200
  }
}

fragment CleanupUsers {
  step "Delete test users" {
    ignore_fail
    run DeleteTestUsers
  }
}
```

Pakai di flow:

```flow
import shared/common-steps.flow
import shared/auth.flow

flow OrderCheckout {
  use fragment AuthenticatedSetup     // step 1 & 2 otomatis masuk

  step "Create order" {
    run CreateOrder
  }

  step "Pay" {
    run PayOrder(order_id)
  }

  teardown {
    use fragment CleanupUsers
  }
}
```

**Keuntungan fragment:**
- Login flow ditulis sekali, dipakai di 10+ flow
- Ubah login logic → semua flow ikut terupdate
- Step numbering otomatis disesuaikan

---

## 9.7 Override Inline vs Extends — Kapan Pakai Mana?

| Situasi | Gunakan |
|---|---|
| Variasi kecil di satu step | Override inline (`run X { ... }`) |
| Variasi yang dipakai berkali-kali | `extends` (buat request baru) |
| Header/auth sama di banyak request | `auth` block |
| Step sequence sama di banyak flow | `fragment` |

### Override inline (sekali pakai)

```flow
step "Create VIP user" {
  run CreateUser {
    body json { role: "vip" }
  }
}
```

### Extends (dipakai berkali-kali)

```flow
request CreateVIPUser extends CreateUser {
  body json { role: "vip" }
}

// Bisa di-run dari banyak flow:
step "..." { run CreateVIPUser }
```

---

## 9.8 Include Flow — Sub-Flow

Gabungkan flow kecil jadi flow besar:

```flow
// flows/regression.flow

flow FullRegression {
  include flows/smoke.flow
  include flows/user-crud.flow
  include flows/post-crud.flow
}
```

Berbeda dari `import` (load definisi request), `include` **menjalankan** flow lain sebagai bagian dari flow ini.

| Keyword | Fungsi |
|---|---|
| `import` | Load definisi request/auth/fragment |
| `include` | Jalankan flow lain sebagai sub-flow |

---

## 9.9 Organisasi File Reuse

```
my-api-tests/
├── shared/
│   ├── auth.flow              ← auth block
│   ├── common-steps.flow      ← fragment
│   └── assertions.flow        ← shared expect patterns
├── requests/
│   ├── users/
│   │   ├── create-user.flow   ← base request
│   │   ├── create-admin.flow  ← extends CreateUser
│   │   └── ...
│   └── ...
└── flows/
    ├── smoke.flow
    ├── user-crud.flow
    └── regression.flow        ← include semua flow
```

---

## 9.10 Best Practice

✅ **Lakukan:**
- Satu `auth` block per scheme (Bearer, API key, Basic)
- Fragment untuk setup/teardown yang berulang
- `extends` untuk variasi request yang sering dipakai
- `import` di awal file — jelas dependensinya

❌ **Hindari:**
- Extends terlalu dalam (maks 2 level: base → child)
- Fragment terlalu besar (maks 5 step)
- Override inline yang panjang (pertimbangkan `extends`)

---

## Ringkasan Bab 9

| Syntax | Fungsi |
|---|---|
| `import path/to/file.flow` | Load definisi dari file lain |
| `auth Name { header ... }` | Auth block reusable |
| `use auth Name` | Pasang auth ke request |
| `request B extends A { ... }` | Inherit + override request |
| `fragment Name { step ... }` | Step sequence reusable |
| `use fragment Name` | Pasang fragment ke flow |
| `include path/to/flow.flow` | Jalankan flow lain sebagai sub-flow |

---

## Latihan Bab 9

**1.** Buat `shared/auth.flow` dengan `BearerAuth`. Pakai di 3 request berbeda.

**2.** Buat `CreateAdminUser extends CreateUser` — override role jadi "admin".

**3.** Buat `fragment LoginSetup` — login + extract token. Pakai di 2 flow berbeda.

**4.** Buat `flows/regression.flow` yang `include` minimal 2 flow lain.

---

**Lanjut →** [Bab 10 — Data-Driven Testing](10-data-driven.md)
