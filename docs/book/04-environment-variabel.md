# Bab 4 — Environment & Variabel

**Estimasi waktu:** 20 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Beda environment dev, staging, prod
- Syntax `{{variable}}` dan `let`
- Dynamic variables (`{{$uuid}}`, dll.)
- Cara aman menyimpan secret (token, password)

---

## 4.1 Mengapa Perlu Environment?

API yang sama biasanya jalan di **beberapa server**:

| Environment | URL contoh | Kapan dipakai |
|---|---|---|
| `dev` | `http://localhost:8080` | Development lokal |
| `staging` | `https://staging-api.example.com` | Pre-production |
| `prod` | `https://api.example.com` | Production (hati-hati!) |

Tanpa environment, kamu harus **edit URL manual** setiap ganti server. Dengan FlowSpec:

```bash
apitest run flows/smoke.flow --env dev
apitest run flows/smoke.flow --env staging
```

File request **sama persis** — hanya environment yang berganti.

---

## 4.2 Definisi Environment

`env/dev.flow`:

```flow
env dev {
  base_url     = "http://localhost:8080"
  access_token = env("API_TOKEN")
  admin_email  = "admin@dev.local"
}
```

`env/staging.flow`:

```flow
env staging {
  base_url     = "https://staging-api.example.com"
  access_token = env("STAGING_API_TOKEN")
  admin_email  = "admin@staging.example.com"
}
```

`env/prod.flow`:

```flow
env prod {
  base_url     = "https://api.example.com"
  access_token = env("PROD_API_TOKEN")
}
```

---

## 4.3 Interpolasi Variable

Di file request, tulis `{{nama_variable}}`:

```flow
request ListUsers {
  GET "{{base_url}}/api/v1/users"
  header Authorization = "Bearer {{access_token}}"
  expect status 200
}
```

Saat run dengan `--env staging`, `{{base_url}}` otomatis jadi URL staging.

---

## 4.4 Prioritas Variable (Yang Mana Menang?)

Jika nama variable sama di beberapa tempat, yang **paling spesifik** menang:

```
1. CLI flag (--var key=value)          ← paling tinggi
2. let di dalam flow
3. extract dari response step sebelumnya
4. env file (dev/staging/prod)
5. apitest.flow global settings        ← paling rendah
```

Contoh override dari CLI:

```bash
apitest run requests/list-users.flow --env dev --var base_url=http://localhost:3000
```

---

## 4.5 Dynamic Variables (Built-in)

FlowSpec punya variable spesial yang di-generate otomatis:

| Variable | Hasil contoh | Kegunaan |
|---|---|---|
| `{{$uuid}}` | `a1b2c3d4-...` | Email unik per test run |
| `{{$timestamp}}` | `1720876543` | ID unik berbasis waktu |
| `{{$randomEmail}}` | `test-x7k2@example.com` | Email random |
| `{{$randomInt}}` | `847291` | Angka random |

Contoh pemakaian:

```flow
flow CreateUniqueUser {
  let user_email = "qa-{{$uuid}}@example.com"
  let created_at = "{{$timestamp}}"

  step "Create user" {
    run CreateUser
  }
}
```

---

## 4.6 Secret — Jangan Tulis di File!

⚠️ **Perhatian:** Jangan pernah commit token/password ke Git.

**Cara benar** — baca dari environment variable OS:

```flow
env dev {
  access_token = env("API_TOKEN")    // baca dari $API_TOKEN di terminal
}
```

Set di terminal sebelum run:

```bash
export API_TOKEN="sk-dev-xxxxx"
apitest run flows/smoke.flow --env dev
```

Atau pakai file `.env` (gitignored):

```bash
# .env
API_TOKEN=sk-dev-xxxxx
STAGING_API_TOKEN=sk-staging-xxxxx
```

FlowSpec otomatis load `.env` jika ada.

Secret **otomatis di-redact** di log:

```
Authorization: Bearer ***REDACTED***
```

---

## 4.7 Variable dengan `let` di Flow

`let` mendefinisikan variable lokal di dalam flow:

```flow
flow CreateAndVerifyUser {
  let user_name  = "Alice"
  let user_email = "alice-{{$uuid}}@example.com"

  step "Create" {
    run CreateUser
  }

  step "Verify" {
    run GetUser(user_id)
    expect json "$.data.name" == user_name
  }
}
```

Variable `user_name` dan `user_email` otomatis tersedia di semua step dalam flow yang sama.

---

## 4.8 Cek Variable yang Aktif

```bash
# Tampilkan semua variable setelah di-resolve
apitest config show --env dev

# Preview request dengan variable ter-resolve
apitest dsl show requests/users/create-user.flow --env dev
```

Output `config show`:

```
Environment: dev
Variables:
  base_url     = http://localhost:8080
  access_token = ***REDACTED***
  admin_email  = admin@dev.local
```

---

## Ringkasan Bab 4

| Konsep | Syntax |
|---|---|
| Definisi env | `env dev { base_url = "..." }` |
| Pakai variable | `{{base_url}}` |
| Baca secret dari OS | `env("API_TOKEN")` |
| Dynamic var | `{{$uuid}}`, `{{$timestamp}}` |
| Variable lokal flow | `let user_email = "..."` |
| Override CLI | `--var key=value` |

---

## Latihan Bab 4

**1.** Buat `env/staging.flow` dengan `base_url` berbeda dari dev.

**2.** Buat request yang pakai `{{base_url}}` — run dengan `--env dev` dan `--env staging`, bandingkan URL di output.

**3.** Buat flow dengan `let user_email = "test-{{$uuid}}@example.com"` — run 2x, pastikan email berbeda.

**4.** Set `API_TOKEN` via `export` — pastikan token tidak muncul di output `-v`.

---

**Lanjut →** [Bab 5 — Assertion dengan expect](05-assertion-expect.md)
