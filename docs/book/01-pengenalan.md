# Bab 1 — Pengenalan

**Estimasi waktu:** 10 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Apa itu API testing dan mengapa penting
- Perbedaan Postman/Apidog vs FlowSpec CLI
- Konsep dasar: `request`, `flow`, `env`
- Preview syntax FlowSpec

---

## 1.1 Apa Itu API Testing?

API (Application Programming Interface) adalah "pintu" aplikasi berkomunikasi lewat HTTP. Contoh:

```
GET  /api/v1/users     → ambil daftar user
POST /api/v1/users     → buat user baru
DELETE /api/v1/users/5 → hapus user id 5
```

**API testing** = kirim request HTTP, lalu **periksa apakah responsenya benar**.

Contoh pemeriksaan:

- Status code `200 OK` (bukan `500 Error`)
- Body JSON berisi field `email`
- Response selesai dalam < 2 detik

Tanpa testing otomatis, setiap kali developer ubah kode, QA harus klik manual di Postman — lambat dan mudah terlewat.

---

## 1.2 Mengapa FlowSpec?

Kamu mungkin sudah kenal **Postman** atau **Apidog** — klik tombol Send, lihat response di panel.

FlowSpec menawarkan pendekatan berbeda:

| Postman/Apidog | FlowSpec |
|---|---|
| GUI (klik-klik) | CLI + file teks (`.flow`) |
| Collection di cloud/app | File di Git repo |
| Sulit di-automate di CI | Native untuk CI/CD |
| Format JSON proprietary | DSL yang terbaca manusia |

**Analogi sederhana:**

- Postman = kalkulator dengan tombol
- FlowSpec = spreadsheet formula — tulis sekali, jalankan berkali-kali, version control friendly

---

## 1.3 Tiga Konsep Inti

Sebelum menulis kode, hafalkan tiga building block FlowSpec:

```
┌──────────┐    ┌──────────┐    ┌──────────┐
│   env    │ →  │ request  │ →  │   flow   │
│ dev/prod │    │ 1 API    │    │ scenario │
└──────────┘    └──────────┘    └──────────┘
```

### `env` — Environment

Tempat kamu simpan **URL server** dan **secret** (token, password).

```flow
env dev {
  base_url = "http://localhost:8080"
}
```

### `request` — Request (unit)

Satu panggilan HTTP + aturan validasi.

```flow
request ListUsers {
  GET "{{base_url}}/api/v1/users"
  expect status 200
}
```

### `flow` — Flow (scenario)

Urutan beberapa request jadi alur bisnis.

```flow
flow UserCRUD {
  step "Create" { run CreateUser }
  step "Delete" { run DeleteUser(user_id) }
}
```

---

## 1.4 Preview: File `.flow` Terlihat Seperti Apa?

Ini contoh lengkap (jangan khawatir jika belum paham semua — kita bahas di bab berikutnya):

```flow
// env/dev.flow
env dev {
  base_url = "http://localhost:8080"
}

// requests/users/list-users.flow
request ListUsers {
  GET "{{base_url}}/api/v1/users"
  expect status 200
  expect json "$.data" is array
}

// flows/smoke.flow
flow SmokeTest {
  step "List all users" { run ListUsers }
}
```

Jalankan:

```bash
apitest run flows/smoke.flow --env dev
```

Output:

```
✓ Step 1: List all users    200  87ms
Summary: 1 passed, 0 failed
```

---

## 1.5 Siapa Cocok Pakai FlowSpec?

✅ **Cocok jika kamu:**
- Developer yang ingin test API sebelum push code
- QA yang ingin regression test otomatis
- DevOps yang pasang gate test di CI/CD
- Tim yang ingin test di-review lewat Pull Request

❌ **Kurang cocok jika kamu:**
- Hanya butuh coba 1 endpoint sekali (pakai `curl` saja)
- Butuh GUI visual flow builder (FlowSpec = tulis teks)

---

## 1.6 Prasyarat

Kamu **tidak perlu** jadi expert programming. Cukup paham:

- [ ] Apa itu HTTP (GET, POST, status code 200/404/500)
- [ ] Apa itu JSON (`{"name": "John"}`)
- [ ] Cara buka terminal / command line

💡 **Tip:** Jika belum paham JSONPath (`$.data.id`), kita jelaskan di Bab 5.

---

## Ringkasan Bab 1

| Istilah | Arti singkat |
|---|---|
| API testing | Kirim HTTP request, validasi response |
| FlowSpec | Bahasa `.flow` khusus API testing |
| `env` | Konfigurasi server (dev/staging/prod) |
| `request` | Satu panggilan API + assertion |
| `flow` | Scenario multi-step |

---

## Latihan Bab 1

Jawaban di [Bab 13](13-jawaban-latihan.md).

**1.** Apa perbedaan utama `request` dan `flow`?

**2.** Mengapa file `.flow` lebih cocok untuk Git dibanding klik manual di GUI?

**3.** Sebutkan 3 hal yang bisa kamu `expect` dari response API.

---

**Lanjut →** [Bab 2 — Setup Project](02-setup-project.md)
