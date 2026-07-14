# Bab 11 — Debugging & CI/CD

**Estimasi waktu:** 30 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Debug test yang gagal (membaca error, verbose mode)
- Dry run & preview
- Integrasi CI/CD (GitHub Actions, GitLab CI)
- Report & artifact
- Tips workflow tim

---

## 11.1 Membaca Error Message

FlowSpec memberikan error yang **kontekstual** — bukan generic YAML error.

### Error: Request gagal

```
✗ CreateUser
  POST https://api.example.com/api/v1/users
  Status: 422 Unprocessable Entity  (85ms)

  Assertion failed [requests/users/create-user.flow:14]:
    expect status 201

    Expected: 201
    Actual:   422

  Response body:
    { "error": "email already exists" }
```

**Cara baca:**
1. Request mana yang gagal (`CreateUser`)
2. Status code actual (`422`)
3. Baris assertion yang gagal (line 14)
4. Body response — petunjuk kenapa gagal

### Error: Variable tidak ditemukan

```
Error [flows/user-crud.flow:18:5]
  Variable 'user_id' used before assignment
  Hint: 'user_id' is extracted in step "Create user" (line 10)
        — ensure that step runs successfully first
```

### Error: Syntax DSL salah

```
Error [requests/create-user.flow:8:3]
  Unexpected token 'jsn' — did you mean 'json'?
  
  8 |   body jsn {
         ^^^
```

---

## 11.2 Verbose Mode — Lihat Detail

```bash
# Level 1: request + response status
apitest run flows/smoke.flow -v

# Level 2: + headers + body
apitest run flows/smoke.flow -vv

# Level 3: + full response body + timing breakdown
apitest run flows/smoke.flow -vvv
```

Output `-vv`:

```
→ POST https://api.example.com/api/v1/users
  Headers:
    Authorization: Bearer ***REDACTED***
    Content-Type: application/json
  Body:
    { "name": "Test User", "email": "test-a1b2@example.com" }

← 201 Created (92ms)
  Headers:
    Content-Type: application/json
    X-Request-Id: req-xyz123
  Body:
    { "data": { "id": 42, "name": "Test User", "email": "test-a1b2@example.com" } }

  ✓ expect status 201
  ✓ expect json "$.data.email" == "test-a1b2@example.com"
  ✓ extract user_id = 42
```

---

## 11.3 Dry Run — Preview Tanpa Kirim Request

```bash
# Lihat request yang akan dikirim (tanpa execute)
apitest dsl show requests/users/create-user.flow --env dev

# Preview seluruh flow — tampilkan step sequence
apitest dsl show flows/user-crud.flow --env staging
```

Output `dsl show`:

```
Flow: UserCRUD (env: staging)
Variables:
  base_url = https://staging-api.example.com
  user_email = test-{{$uuid}}@example.com

Steps:
  1. Create user   → POST /api/v1/users
  2. Get user      → GET /api/v1/users/{{user_id}}  [when user_id]
  3. Delete user   → DELETE /api/v1/users/{{user_id}}
```

---

## 11.4 Lint — Tangkap Error Sebelum Run

```bash
# Lint semua file
apitest dsl lint .

# Lint folder tertentu
apitest dsl lint requests/
apitest dsl lint flows/

# Lint satu file
apitest dsl lint requests/users/create-user.flow
```

Output lint:

```
requests/users/create-user.flow
  ✓ Syntax valid
  ⚠ Line 5: variable 'user_name' defined but never used in this file
  
flows/user-crud.flow
  ✗ Line 12: 'GetUsers' not found — did you mean 'GetUser'?
  ✗ Line 3: missing import for 'CreateUser'

Summary: 1 error, 1 warning
```

💡 **Tip:** Jalankan `apitest dsl lint .` sebelum commit — tangkap typo lebih awal.

---

## 11.5 CI/CD — GitHub Actions

Buat file `.github/workflows/api-test.yml`:

```yaml
name: API Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  api-test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Install apitest
        run: |
          curl -fsSL https://example.com/install.sh | sh
          apitest --version

      - name: Lint FlowSpec files
        run: apitest dsl lint .

      - name: Run smoke tests
        env:
          API_TOKEN: ${{ secrets.STAGING_API_TOKEN }}
        run: apitest run flows/smoke.flow --env staging --report junit

      - name: Run full regression
        if: github.event_name == 'push'
        env:
          API_TOKEN: ${{ secrets.STAGING_API_TOKEN }}
        run: apitest run flows/ --env staging --tags regression --report junit

      - name: Upload test report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: api-test-report
          path: reports/

      - name: Publish JUnit results
        if: always()
        uses: mikepenz/action-junit-report@v4
        with:
          report_paths: reports/*.xml
```

**Penjelasan flow CI:**

1. Checkout code
2. Install `apitest`
3. Lint — gagal cepat jika syntax error
4. Smoke test — cepat, jalan setiap PR
5. Full regression — hanya saat push ke main
6. Upload report — tersedia sebagai artifact

---

## 11.6 CI/CD — GitLab CI

File `.gitlab-ci.yml`:

```yaml
stages:
  - lint
  - test

variables:
  API_TOKEN: $STAGING_API_TOKEN

lint:
  stage: lint
  script:
    - curl -fsSL https://example.com/install.sh | sh
    - apitest dsl lint .

smoke-test:
  stage: test
  script:
    - apitest run flows/smoke.flow --env staging --report junit
  artifacts:
    when: always
    reports:
      junit: reports/*.xml
    paths:
      - reports/

regression-test:
  stage: test
  only:
    - main
  script:
    - apitest run flows/ --env staging --tags regression --report junit
  artifacts:
    when: always
    reports:
      junit: reports/*.xml
    paths:
      - reports/
```

---

## 11.7 Report Format

```bash
# JUnit XML (untuk CI)
apitest run flows/ --report junit

# HTML report (human-readable)
apitest run flows/ --report html

# JSON (untuk custom dashboard)
apitest run flows/ --report json

# Semua format sekaligus
apitest run flows/ --report junit,html,json
```

Output di folder `reports/`:

```
reports/
├── report-2024-01-15T10-30-00.xml    ← JUnit
├── report-2024-01-15T10-30-00.html   ← HTML
└── report-2024-01-15T10-30-00.json   ← JSON
```

### Report HTML

Report HTML menampilkan:
- Summary: total pass/fail/skip
- Timeline: durasi per step
- Detail: request/response untuk yang gagal
- Environment info: env, variable (secret redacted)

---

## 11.8 Exit Code untuk CI

| Exit code | Arti | CI behavior |
|---|---|---|
| `0` | Semua test pass | ✅ Pipeline hijau |
| `1` | Ada test yang fail | ❌ Pipeline merah |
| `2` | Error syntax/config | ❌ Pipeline merah |
| `3` | Network/timeout error | ❌ Pipeline merah |

CI otomatis gagal jika exit code ≠ 0 — tidak perlu konfigurasi tambahan.

---

## 11.9 Secret di CI

⚠️ **Jangan hardcode token di file `.flow` atau CI config!**

### GitHub Actions

```yaml
env:
  API_TOKEN: ${{ secrets.API_TOKEN }}
  STAGING_API_TOKEN: ${{ secrets.STAGING_API_TOKEN }}
```

Set di: Repository → Settings → Secrets and variables → Actions

### GitLab CI

Set di: Settings → CI/CD → Variables (masked & protected)

### Di `env/*.flow`

```flow
env staging {
  base_url     = "https://staging-api.example.com"
  access_token = env("API_TOKEN")         // baca dari CI secret
}
```

---

## 11.10 Workflow Tim — Rekomendasi

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Developer  │     │    CI/CD    │     │  Main/Prod  │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                    │
  1. Edit .flow            │                    │
  2. apitest dsl lint      │                    │
  3. apitest run --env dev │                    │
  4. git push (PR)────────▶│                    │
       │              5. Lint                    │
       │              6. Smoke test (staging)    │
       │              7. PR status check         │
       │                   │                    │
  8. Merge ───────────────▶│                    │
       │              9. Full regression ───────▶│
       │              10. Report artifact        │
```

**Tips:**
- PR = smoke test saja (cepat, < 2 menit)
- Merge ke main = full regression
- Pakai `--tags smoke` untuk PR, `--tags regression` untuk main
- Review file `.flow` di PR seperti review code

---

## 11.11 Troubleshooting Checklist

Saat test gagal di CI tapi pass di lokal:

| Cek | Command |
|---|---|
| Environment benar? | `apitest config show --env staging` |
| Variable ter-set? | Cek CI secrets |
| Network accessible? | `curl -I $STAGING_URL` |
| Timeout? | Naikkan `settings { timeout = 60s }` |
| Data state? | Tambah teardown cleanup |
| Race condition? | Tambah `wait` atau `retry until` |

---

## Ringkasan Bab 11

| Perintah | Fungsi |
|---|---|
| `apitest run ... -v` | Verbose output |
| `apitest run ... -vv` | Full request/response |
| `apitest dsl show ...` | Dry run / preview |
| `apitest dsl lint .` | Validasi syntax |
| `apitest run ... --report junit` | Generate report |
| `--fail-fast` | Stop di error pertama |
| `--tags smoke` | Filter test by tag |

---

## Latihan Bab 11

**1.** Jalankan request dengan `-vv` — baca request headers dan response body yang muncul.

**2.** Sengaja tulis typo di request name — jalankan `apitest dsl lint`, baca error message.

**3.** Buat file GitHub Actions (`.github/workflows/api-test.yml`) — lint + smoke test.

**4.** Jalankan `apitest run flows/ --report html` — buka report di browser.

---

**Lanjut →** [Bab 12 — Project Lengkap](12-project-lengkap.md)
