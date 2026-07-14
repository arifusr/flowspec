# Bab 6 — Extract & Data Flow

**Estimasi waktu:** 25 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Ambil data dari response → simpan ke variable
- Gunakan data step 1 di step 2, 3, dst.
- Keyword `extract` dan `last`
- Pola umum: login → pakai token → CRUD

---

## 6.1 Masalah yang Diselesaikan Extract

Bayangkan scenario:

1. **Create user** → server return `user_id: 42`
2. **Get user** → butuh `user_id` dari step 1
3. **Delete user** → butuh `user_id` yang sama

Tanpa extract, kamu hardcode `42` — tidak realistis.

**Extract** = ambil value dari response, simpan ke variable, pakai di step berikutnya.

---

## 6.2 Syntax `extract`

```flow
request CreateUser {
  POST "{{base_url}}/api/v1/users"

  body json {
    name:  "{{user_name}}"
    email: "{{user_email}}"
  }

  expect status 201

  extract {
    user_id    from json "$.data.id"
    user_email from json "$.data.email"
    created_at from json "$.data.created_at"
  }
}
```

Setelah request sukses:
- `{{user_id}}` = nilai `$.data.id` dari response
- Variable tersedia di step flow berikutnya

---

## 6.3 Extract dari Header & Cookie

```flow
request Login {
  POST "{{base_url}}/auth/login"

  body json {
    email: "{{email}}"
    password: "{{password}}"
  }

  expect status 200

  extract {
    access_token from json "$.token"
    session_id   from header "Set-Cookie"
    request_id   from header "X-Request-Id"
  }
}
```

---

## 6.4 Data Flow Visual

```
Step 1: CreateUser
         │
         ▼
    Response: { "data": { "id": 42 } }
         │
         extract user_id = 42
         │
         ▼
Step 2: GetUser(user_id)     ← user_id = 42
         │
         ▼
Step 3: DeleteUser(user_id)  ← user_id = 42
```

---

## 6.5 Keyword `last` — Extract Inline di Flow

Alternatif extract di dalam step flow:

```flow
flow LoginAndBrowse {
  step "Login" {
    run Login
    let access_token = last.json("$.token")
    let user_id      = last.json("$.user.id")
  }

  step "Get profile" {
    run GetProfile(user_id)
    expect json "$.data.id" == user_id
  }
}
```

| Keyword | Arti |
|---|---|
| `last` | Response dari step/run terakhir |
| `last.json("$.path")` | Ambil field JSON |
| `last.header("Name")` | Ambil header |
| `last.status` | Status code (200, 201, dll.) |

---

## 6.6 Filter di Array — Cari Item Spesifik

Sering kali API mengembalikan array (misalnya dropdown, list). Kamu butuh **satu item spesifik** berdasarkan value field-nya.

Gunakan **filter expression** `[?(@.field=='value')]`:

```flow
step "Get company dropdown" {
  run GetCompanyList
  // Response: [{"id": 5, "name": "PT ABC"}, {"id": 8, "name": "PT XYZ"}]

  let company_id   = last.json("$[?(@.name=='PT ABC')].id")
  let company_name = last.json("$[?(@.name=='PT ABC')].name")
  log("Dipilih: {{company_name}} (id={{company_id}})")
}
```

**Operator filter yang didukung:**

| Operator | Contoh | Arti |
|---|---|---|
| `==` | `$[?(@.name=='Alice')]` | Sama dengan |
| `!=` | `$[?(@.status!='deleted')]` | Tidak sama |
| `>` | `$[?(@.price>100)]` | Lebih besar |
| `>=` | `$[?(@.stock>=10)]` | Lebih besar atau sama |
| `<` | `$[?(@.age<30)]` | Lebih kecil |
| `<=` | `$[?(@.score<=50)]` | Lebih kecil atau sama |

Filter mengembalikan item **pertama** yang cocok. Kamu bisa langsung akses field-nya dengan `.fieldname` setelah filter.

**Contoh nested path:**

```flow
// Response: { "data": { "items": [{"code": "IDR", "id": 1}, {"code": "USD", "id": 2}] } }
let idr_id = last.json("$.data.items[?(@.code=='IDR')].id")
```

---

## 6.7 `log()` — Debug Print Variable

Gunakan `log()` untuk mencetak variable atau data ke console:

```flow
step "Login" {
  run Login
  let token = last.json("$.token")
  log("Token: {{token}}")
  log("Status: {{last.status}}")
}
```

Output di terminal:

```
  📋 log: Token: eyJhbGciOi...
  📋 log: Status: 200
```

💡 **Tip:** `log()` sangat berguna untuk debug saat extract tidak menghasilkan value yang diharapkan — inspect actual response data.

⚠️ **Perhatian:** `log()` hanya tampil saat run biasa. Di `--quiet` mode (CI), log tetap tampil untuk membantu debug pipeline.

---

## 6.8 Contoh Lengkap: Login → CRUD

```flow
// requests/auth/login.flow
request Login {
  POST "{{base_url}}/auth/login"
  body json {
    email: "{{admin_email}}"
    password: env("ADMIN_PASSWORD")
  }
  expect status 200
  extract {
    access_token from json "$.token"
  }
}

// requests/users/create-user.flow
request CreateUser {
  POST "{{base_url}}/api/v1/users"
  header Authorization = "Bearer {{access_token}}"
  body json {
    name: "{{user_name}}"
    email: "{{user_email}}"
  }
  expect status 201
  extract {
    user_id from json "$.data.id"
  }
}

// requests/users/get-user.flow
request GetUser(user_id) {
  GET "{{base_url}}/api/v1/users/{{user_id}}"
  header Authorization = "Bearer {{access_token}}"
  expect status 200
}

// requests/users/delete-user.flow
request DeleteUser(user_id) {
  DELETE "{{base_url}}/api/v1/users/{{user_id}}"
  header Authorization = "Bearer {{access_token}}"
  expect status 204
}
```

---

## 6.7 Chain Extract — Response Bergantung Response

```flow
flow OrderFlow {
  step "Create order" {
    run CreateOrder
    let order_id = last.json("$.data.id")
  }

  step "Pay order" {
    run PayOrder(order_id)
    let payment_id = last.json("$.data.payment_id")
  }

  step "Verify payment" {
    run GetPayment(payment_id)
    expect json "$.data.status" == "paid"
  }
}
```

Setiap step bisa extract dan meneruskan variable ke step berikutnya.

---

## 6.8 Guard dengan `when` — Lindungi Step jika Extract Gagal

Jika step 1 gagal, `user_id` tidak ada. Lindungi step 2:

```flow
step "Get user" {
  when user_id              // skip step jika user_id kosong
  run GetUser(user_id)
}
```

Detail `when` di Bab 8.

---

## ⚠️ Kesalahan Umum Pemula

### 1. Extract path salah

```flow
// Response: { "data": { "id": 42 } }

extract { user_id from json "$.id" }        // ✗ SALAH — id ada di $.data.id
extract { user_id from json "$.data.id" }   // ✓ BENAR
```

**Solusi:** Jalankan request dengan `-vv`, lihat response body, tentukan JSONPath yang benar.

### 2. Pakai variable sebelum di-extract

```flow
step "Get user" { run GetUser(user_id) }    // ✗ user_id belum ada
step "Create"   { run CreateUser }          // ✓ create dulu, extract user_id
step "Get user" { run GetUser(user_id) }    // ✓ baru get
```

### 3. Lupa Authorization header setelah login

Setelah extract `access_token`, pastikan request berikutnya pakai:

```flow
header Authorization = "Bearer {{access_token}}"
```

Atau gunakan `auth` block (Bab 9).

---

## Ringkasan Bab 6

| Syntax | Fungsi |
|---|---|
| `extract { x from json "$.path" }` | Simpan field JSON ke variable |
| `extract { x from header "Name" }` | Simpan header ke variable |
| `let x = last.json("$.path")` | Extract inline setelah run |
| `let x = last.json("$[?(@.k=='v')].field")` | Filter array + extract |
| `let x = last.header("Name")` | Extract header inline |
| `log("message {{var}}")` | Debug print ke console |
| `run GetUser(user_id)` | Pakai variable sebagai parameter |

---

## Latihan Bab 6

**1.** Buat flow: `CreatePost` → extract `post_id` → `GetPost(post_id)` → assert title sama.

**2.** Buat flow login mock: step 1 set `let access_token = "fake-token-123"`, step 2 pakai token di header.

**3.** Sengaja salah JSONPath — lihat error "Variable 'post_id' not found".

---

**Lanjut →** [Bab 7 — Flow & Skenario](07-flow-skenario.md)
