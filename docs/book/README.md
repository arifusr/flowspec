# FlowSpec untuk Pemula — Buku Tutorial

Selamat datang! Buku ini mengajarkan **FlowSpec DSL** dari nol — cocok untuk kamu yang:

- Baru pertama kali testing API
- Familiar dengan Postman/Apidog tapi belum pernah pakai CLI
- Ingin menulis test yang rapi, terbaca, dan bisa di-commit ke Git

> **FlowSpec** adalah bahasa khusus (`.flow`) untuk mendefinisikan dan menjalankan HTTP API test dari terminal.

---

## Cara Membaca Buku Ini

Baca berurutan dari Bab 1–12 jika kamu benar-benar pemula. Jika sudah paham HTTP dasar, mulai dari Bab 3.

Setiap bab berakhir dengan **Ringkasan** dan **Latihan**. Jawaban latihan ada di [Bab 13](13-jawaban-latihan.md).

---

## Daftar Isi

| Bab | Judul | Apa yang akan kamu pelajari |
|:---:|---|---|
| [1](01-pengenalan.md) | Pengenalan | Apa itu API testing, FlowSpec, dan kapan menggunakannya |
| [2](02-setup-project.md) | Setup Project | Install `apitest`, init project, struktur folder |
| [3](03-request-pertama.md) | Request Pertama | Tulis & jalankan HTTP GET/POST pertama |
| [4](04-environment-variabel.md) | Environment & Variabel | Kelola dev/staging/prod, secret, dynamic vars |
| [5](05-assertion-expect.md) | Assertion dengan `expect` | Validasi status, body, header, response time |
| [6](06-extract-data-flow.md) | Extract & Data Flow | Kirim data antar step (user_id, token, dll.) |
| [7](07-flow-skenario.md) | Flow & Skenario | Gabungkan request jadi alur bisnis |
| [8](08-control-flow.md) | Control Flow | Kondisi, loop, retry, wait |
| [9](09-reuse-composition.md) | Reuse & Composition | Import, extends, fragment, override |
| [10](10-data-driven.md) | Data-Driven Testing | Test banyak data sekaligus dari CSV |
| [11](11-debugging-cicd.md) | Debugging & CI/CD | Debug error, integrasi pipeline |
| [12](12-project-lengkap.md) | Project Lengkap | Bangun test suite dari nol sampai CI |
| [13](13-jawaban-latihan.md) | Jawaban Latihan | Solusi semua latihan |

---

## Referensi Cepat

- [Spesifikasi DSL lengkap](../flowspec-dsl.md) — untuk lookup syntax detail
- [Contoh kode](../../examples/) — file `.flow` siap pakai
- [README proyek](../../README.md) — overview produk

---

## Konvensi di Buku Ini

| Simbol | Arti |
|---|---|
| `apitest run ...` | Perintah terminal — ketik persis seperti ini |
| Blok kode `.flow` | Kode FlowSpec — simpan ke file `.flow` |
| 💡 **Tip** | Saran praktis |
| ⚠️ **Perhatian** | Kesalahan umum pemula |
| ✅ **Checklist** | Verifikasi sebelum lanjut |

---

**Siap?** Mulai dari [Bab 1 — Pengenalan](01-pengenalan.md).
