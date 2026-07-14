# Bab 10 — Data-Driven Testing

**Estimasi waktu:** 25 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Apa itu data-driven testing dan kapan memakainya
- Inline table — data langsung di file `.flow`
- External CSV — data dari file terpisah
- Validasi per baris data
- Tips organisasi data test

---

## 10.1 Masalah: Test Satu Data Saja Tidak Cukup

Kamu punya endpoint "Create User" yang menerima role: `admin`, `user`, `guest`.

Tanpa data-driven, kamu tulis 3 request terpisah:

```flow
// ✗ Repetitif
request CreateAdmin { ... body json { role: "admin" } ... }
request CreateUser  { ... body json { role: "user"  } ... }
request CreateGuest { ... body json { role: "guest" } ... }
```

Dengan data-driven — satu request, banyak data:

```flow
// ✓ Satu loop, 3 data
for row in [
  { role: "admin", expected: 201 },
  { role: "user",  expected: 201 },
  { role: "guest", expected: 201 }
] {
  run CreateUser { body json { role: row.role } }
  expect status row.expected
}
```

---

## 10.2 Inline Table — Data di Dalam File Flow

### Array of objects

```flow
flow TestUserRoles {
  description "Verify semua role bisa di-create"

  for row in [
    { name: "Alice",   email: "alice@example.com",   role: "admin" },
    { name: "Bob",     email: "bob@example.com",     role: "user"  },
    { name: "Charlie", email: "charlie@example.com", role: "guest" }
  ] {
    let user_name  = row.name
    let user_email = row.email

    step "Create {{row.role}} user: {{row.name}}" {
      run CreateUser {
        body json {
          name:  user_name
          email: user_email
          role:  row.role
        }
      }
      expect status 201
      expect json "$.data.role" == row.role
    }
  }
}
```

Output:

```
Scenario: TestUserRoles
──────────────────────────────────────────
  ✓ Create admin user: Alice     201  92ms
  ✓ Create user user: Bob        201  88ms
  ✓ Create guest user: Charlie   201  95ms

Summary: 3 passed, 0 failed
```

### Array sederhana

```flow
flow TestInvalidEmails {
  for email in ["", "not-an-email", "@missing.com", "no-domain@"] {
    step "Reject invalid: {{email}}" {
      run CreateUser {
        body json { email: email }
      }
      expect status 422
      expect json "$.error" matches "email"
    }
  }
}
```

---

## 10.3 External CSV — Data dari File

Untuk data banyak (puluhan/ratusan row), simpan di file CSV:

### File `data/users.csv`

```csv
name,email,role,expected_status
Alice,alice@example.com,admin,201
Bob,bob@example.com,user,201
Charlie,charlie@example.com,guest,201
InvalidUser,,user,422
NoRole,norole@example.com,,422
```

### Flow yang membaca CSV

```flow
flow CreateUsersBatch {
  description "Test create user dari CSV data"

  for row in csv("data/users.csv") {
    let user_name  = row.name
    let user_email = row.email

    step "Create: {{row.name}} ({{row.role}})" {
      run CreateUser {
        body json {
          name:  user_name
          email: user_email
          role:  row.role
        }
      }
      expect status row.expected_status
    }
  }
}
```

**Aturan CSV:**
- Baris pertama = header (nama kolom)
- Akses via `row.nama_kolom`
- Semua value CSV adalah string — FlowSpec auto-convert angka untuk `expect status`
- Path relatif dari root project

---

## 10.4 JSON File sebagai Data Source

Selain CSV, bisa pakai JSON:

### File `data/test-cases.json`

```json
[
  { "name": "Alice", "email": "alice@example.com", "role": "admin", "expected_status": 201 },
  { "name": "Bob", "email": "bob@example.com", "role": "user", "expected_status": 201 },
  { "name": "", "email": "invalid", "role": "user", "expected_status": 422 }
]
```

### Flow

```flow
flow CreateUsersFromJSON {
  for row in data("data/test-cases.json") {
    step "Create: {{row.name}}" {
      run CreateUser {
        body json {
          name:  row.name
          email: row.email
          role:  row.role
        }
      }
      expect status row.expected_status
    }
  }
}
```

---

## 10.5 Kombinasi: Data-Driven + Extract + Assert

Test yang lebih realistis — create lalu verify:

```flow
flow VerifyUserCreation {
  for row in csv("data/users.csv") {
    step "Create {{row.name}}" {
      run CreateUser {
        body json {
          name:  row.name
          email: row.email
          role:  row.role
        }
      }
      expect status 201
      let user_id = last.json("$.data.id")
    }

    step "Verify {{row.name}}" {
      when user_id
      run GetUser(user_id)
      expect json "$.data.name" == row.name
      expect json "$.data.email" == row.email
      expect json "$.data.role" == row.role
    }
  }
}
```

---

## 10.6 Negative Testing — Validasi Error

Data-driven sangat cocok untuk test **boundary & error case**:

```flow
flow TestPasswordValidation {
  description "Password harus 8+ char, ada huruf besar & angka"

  for case in [
    { password: "short",      expect_error: "minimum 8 characters" },
    { password: "nouppercase1", expect_error: "uppercase letter" },
    { password: "NONUMBER",   expect_error: "at least one number" },
    { password: "Valid1234",  expect_error: null }
  ] {
    step "Test: {{case.password}}" {
      run Register {
        body json { password: case.password }
      }

      when case.expect_error {
        expect status 422
        expect json "$.error" matches case.expect_error
      }

      unless case.expect_error {
        expect status 201
      }
    }
  }
}
```

---

## 10.7 Tips Organisasi Data

### Struktur folder

```
data/
├── users.csv            ← user test data
├── products.csv         ← product test data
├── invalid-inputs.csv   ← negative test cases
└── fixtures/
    ├── avatar.png       ← upload test
    └── large-file.bin   ← size limit test
```

### Kapan inline vs external?

| Jumlah data | Gunakan |
|---|---|
| 2–5 row | Inline table di file `.flow` |
| 5–20 row | CSV/JSON terpisah |
| 20+ row | CSV terpisah + komentar/dokumentasi |

### Naming convention

```
data/
├── users-valid.csv          ← happy path
├── users-invalid.csv        ← negative cases
├── users-boundary.csv       ← edge cases
└── users-performance.csv    ← load test data
```

---

## 10.8 Data-Driven + Tags

Filter test berdasarkan jenis data:

```flow
@tags(users, negative)

flow InvalidUserInputs {
  for row in csv("data/users-invalid.csv") {
    step "Reject: {{row.description}}" {
      run CreateUser {
        body json { email: row.email, name: row.name }
      }
      expect status 422
    }
  }
}
```

```bash
# Jalankan hanya negative test
apitest run flows/ --tags negative

# Jalankan semua kecuali negative
apitest run flows/ --exclude-tags negative
```

---

## 10.9 Debugging Data-Driven Test

Saat satu row gagal di tengah 100 row:

```bash
# Verbose — lihat detail setiap iterasi
apitest run flows/create-users-batch.flow -v

# Fail fast — stop di error pertama
apitest run flows/create-users-batch.flow --fail-fast
```

Output error menunjukkan row mana yang gagal:

```
✗ Create: InvalidUser (row 4 of data/users.csv)
  POST https://api.example.com/api/v1/users
  Status: 500 Internal Server Error

  Assertion failed:
    expect status 422
    Expected: 422
    Actual:   500

  Data row:
    name: "InvalidUser"
    email: ""
    role: "user"
```

---

## Ringkasan Bab 10

| Syntax | Fungsi |
|---|---|
| `for row in [ {...}, {...} ]` | Inline data array |
| `for row in csv("path.csv")` | Data dari CSV |
| `for row in data("path.json")` | Data dari JSON |
| `row.field_name` | Akses kolom per baris |
| `step "... {{row.x}}"` | Label dinamis per iterasi |

---

## Latihan Bab 10

**1.** Buat inline table 3 user — create dan verify role masing-masing.

**2.** Buat file `data/posts.csv` (title, body, userId) — buat flow yang create post per row.

**3.** Buat negative test: 4 invalid email format, expect status 422.

**4.** Jalankan data-driven flow dengan `--fail-fast`, lihat behavior saat satu row gagal.

---

**Lanjut →** [Bab 11 — Debugging & CI/CD](11-debugging-cicd.md)
