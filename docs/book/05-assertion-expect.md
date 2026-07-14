# Bab 5 — Assertion dengan `expect`

**Estimasi waktu:** 25 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Semua jenis `expect` di FlowSpec
- JSONPath untuk assert body JSON
- Assert header dan response time
- Pesan error saat assertion gagal

---

## 5.1 Apa Itu Assertion?

**Assertion** = pernyataan "response harus seperti ini". Jika tidak sesuai → test **FAIL**.

```flow
expect status 200           // HTTP status harus 200
expect json "$.id" exists  // field id harus ada
expect time < 500ms        // response harus cepat
```

FlowSpec dirancang agar `expect` terbaca seperti **bahasa natural**.

---

## 5.2 Assert Status Code

```flow
# Exact
expect status 200
expect status 201
expect status 204

# Salah satu dari list
expect status in [200, 201, 204]

# Range (semua 2xx)
expect status 2xx

# Negasi
expect status != 500
```

💡 **Tip status code umum:**

| Code | Arti |
|---|---|
| 200 | OK — sukses |
| 201 | Created — resource baru dibuat |
| 204 | No Content — sukses, body kosong |
| 400 | Bad Request — input salah |
| 401 | Unauthorized — butuh login/token |
| 403 | Forbidden — tidak punya akses |
| 404 | Not Found — endpoint/resource tidak ada |
| 422 | Unprocessable — validasi gagal |
| 500 | Internal Server Error — bug di server |

---

## 5.3 Assert JSON Body

### Cek field exists

```flow
expect json "$.data.id" exists
expect json "$.data.email" exists
```

### Cek nilai exact

```flow
expect json "$.data.name" == "Alice"
expect json "$.data.role" == "{{expected_role}}"
```

### Cek tipe data

```flow
expect json "$.data" is array
expect json "$.data.id" is number
expect json "$.data.active" is boolean
expect json "$.data.name" is string
expect json "$.data.meta" is object
```

### Cek panjang array/string

```flow
expect json "$.data" length 10       // array persis 10 item
expect json "$.data" length >= 1     // minimal 1 item
expect json "$.data.roles" length 3
```

### Cek pattern regex

```flow
expect json "$.data.email" matches "^[a-z]+@example\\.com$"
expect json "$.message" matches "^User created"
```

### Cek numerik

```flow
expect json "$.meta.total" >= 10
expect json "$.meta.total" > 0
expect json "$.price" <= 999.99
```

### Negasi

```flow
expect json "$.error" not exists
expect json "$.data.deleted" != true
```

---

## 5.4 Assert Header

```flow
expect header Content-Type contains "application/json"
expect header X-Request-Id exists
expect header Cache-Control == "no-cache"
expect header Set-Cookie matches "SESSION="
```

---

## 5.5 Assert Performance

```flow
expect time < 500ms      // kurang dari 500 milidetik
expect time <= 2s        // maks 2 detik
expect size > 100 bytes  // body minimal 100 byte
expect size < 1mb        // body maks 1 megabyte
```

💡 **Tip:** Assert response time penting untuk catch regression performa.

---

## 5.6 Contoh Request Lengkap dengan Assertions

```flow
@tags(users, smoke)

request GetUser(user_id) {
  GET "{{base_url}}/api/v1/users/{{user_id}}"

  header Authorization = "Bearer {{access_token}}"
  header Accept        = "application/json"

  expect status 200
  expect header Content-Type contains "json"
  expect time < 1s

  expect json "$.data.id" == "{{user_id}}"
  expect json "$.data.email" exists
  expect json "$.data.email" matches "@"
  expect json "$.data.roles" is array
  expect json "$.data.roles" length >= 1
}
```

---

## 5.7 Membaca Pesan Error Assertion

Saat assertion gagal, FlowSpec menampilkan **expected vs actual**:

```
✗ GetUser
  GET https://api.example.com/api/v1/users/42
  Status: 200 OK  (89ms)

  Assertion failed [line 12]:
    expect json "$.data.email" == "alice@example.com"

    Expected: "alice@example.com"
    Actual:   "bob@example.com"
    Path:     $.data.email
```

Informasi yang diberikan:
- File & baris assertion yang gagal
- Nilai yang diharapkan vs yang diterima
- JSONPath ke field bermasalah

---

## 5.8 Assert di Step vs di Request

Assertion bisa di **request** (jalan setiap kali request dipanggil):

```flow
request CreateUser {
  POST "..."
  expect status 201    // selalu dicek
}
```

Atau tambahan di **step flow** (hanya untuk context scenario tertentu):

```flow
flow AdminUserTest {
  step "Create admin" {
    run CreateUser {
      body json { role: "admin" }
      expect json "$.data.role" == "admin"   // override/tambahan
    }
  }
}
```

---

## 5.9 Best Practice untuk Pemula

✅ **Lakukan:**
- Assert status code **selalu**
- Assert field kritis bisnis (`id`, `email`, `total`)
- Assert response time untuk endpoint penting
- Beri assertion yang spesifik (bukan cuma `status 200`)

❌ **Hindari:**
- Assert terlalu banyak field yang tidak relevan
- Hardcode ID yang berubah-ubah (pakai `extract` + variable)
- Assert status 200 saja tanpa cek body

---

## Ringkasan Bab 5

| Assert | Contoh |
|---|---|
| Status | `expect status 200` |
| JSON exists | `expect json "$.id" exists` |
| JSON equals | `expect json "$.name" == "Alice"` |
| JSON type | `expect json "$.data" is array` |
| JSON length | `expect json "$.items" length >= 1` |
| Header | `expect header Content-Type contains "json"` |
| Time | `expect time < 500ms` |

---

## Latihan Bab 5

Pakai JSONPlaceholder.

**1.** `ListPosts` — assert status 200, root is array, length >= 10, time < 3s.

**2.** `GetPost(1)` — assert `$.id == 1`, `$.title` exists, `$.userId` is number.

**3.** Sengaja buat assertion salah (`expect status 404`) — baca pesan error, pahami formatnya.

**4.** Tambah assert regex: `$.title` matches `"^[A-Z]"`.

---

**Lanjut →** [Bab 6 — Extract & Data Flow](06-extract-data-flow.md)
