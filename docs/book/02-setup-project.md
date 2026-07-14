# Bab 2 — Setup Project

**Estimasi waktu:** 15 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Install tool `apitest`
- Buat project pertama dengan `apitest init`
- Pahami struktur folder
- Jalankan test kosong pertama kali

---

## 2.1 Install apitest

```bash
# macOS
brew install apitest

# Linux / WSL
curl -fsSL https://example.com/install.sh | sh

# Python (alternatif development)
pip install apitest-cli

# Verifikasi
apitest --version
# apitest v0.1.0
```

💡 **Tip:** Jika perintah `apitest` tidak ditemukan, restart terminal atau cek PATH.

---

## 2.2 Buat Project Baru

```bash
mkdir belajar-flowspec
cd belajar-flowspec
apitest init
```

Output:

```
✓ Created apitest.flow
✓ Created env/dev.flow
✓ Created env/staging.flow
✓ Created requests/
✓ Created flows/
✓ Created data/
✓ Created reports/  (gitignored)
✓ Created .gitignore
```

---

## 2.3 Struktur Folder — Penjelasan untuk Pemula

```
belajar-flowspec/
│
├── apitest.flow          ← Konfigurasi global project
│
├── env/                  ← Variable per environment
│   ├── dev.flow          ← Server lokal / development
│   └── staging.flow      ← Server staging
│
├── requests/             ← SATU file = SATU request API
│   └── (kosong dulu)
│
├── flows/                ← Scenario / alur bisnis
│   └── (kosong dulu)
│
├── data/                 ← File CSV untuk data-driven test
├── scripts/              ← JavaScript hooks (nanti)
├── specs/                ← OpenAPI spec (nanti)
└── reports/              ← Hasil test (auto-generated)
```

**Aturan emas untuk pemula:**

| Folder | Isi | Analogi |
|---|---|---|
| `env/` | URL & token | "Mau test ke server mana?" |
| `requests/` | Request individual (auto-loaded) | "Resep masakan per hidangan" |
| `flows/` | Scenario | "Menu lengkap: appetizer → main → dessert" |

💡 **Tip:** Semua file `.flow` di `requests/` dan `shared/` otomatis dimuat saat kamu run flow. Flow bisa langsung `run NamaRequest` tanpa perlu `import`.

---

## 2.4 File Konfigurasi Global

Buka `apitest.flow`:

```flow
project "Belajar FlowSpec" {
  version     = "1.0"
  default_env = dev

  env dev     from "env/dev.flow"
  env staging from "env/staging.flow"

  settings {
    timeout    = 30s
    report_dir = "reports/"
  }
}
```

Penjelasan baris per baris:

| Baris | Arti |
|---|---|
| `project "..."` | Nama project (tampil di report) |
| `default_env = dev` | Environment default saat run tanpa `--env` |
| `env dev from "..."` | Load variable dari file env |
| `timeout = 30s` | Maks waktu tunggu per request |
| `report_dir` | Folder output report |

---

## 2.5 Environment Pertama

Buka `env/dev.flow`:

```flow
env dev {
  base_url = "http://localhost:8080"
}
```

Ini artinya semua request ke dev akan pakai `http://localhost:8080` sebagai awalan URL.

⚠️ **Perhatian:** Ganti `base_url` sesuai API kamu. Untuk latihan, kita pakai **JSONPlaceholder** — API publik gratis:

```flow
env dev {
  base_url = "https://jsonplaceholder.typicode.com"
}
```

JSONPlaceholder tidak butuh token — cocok untuk belajar.

---

## 2.6 Perintah CLI yang Perlu Dihafal

```bash
# Jalankan file .flow
apitest run <path>

# Pilih environment
apitest run flows/smoke.flow --env staging

# Validasi syntax (lint)
apitest dsl lint .

# Preview request setelah variable di-resolve
apitest dsl show requests/list-users.flow --env dev

# Bantuan
apitest help run
apitest help dsl
```

---

## 2.7 Init Git (Opsional tapi Direkomendasikan)

```bash
git init
git add .
git commit -m "chore: init FlowSpec project"
```

File yang **jangan** di-commit:

```
reports/       # hasil test
.env           # secret
*.local.flow   # override lokal
```

`.gitignore` sudah dibuat otomatis oleh `apitest init`.

---

## Ringkasan Bab 2

- `apitest init` → scaffold project lengkap
- `env/` = konfigurasi server
- `requests/` = unit test per endpoint
- `flows/` = scenario multi-step
- Pakai JSONPlaceholder jika belum punya API sendiri

---

## Latihan Bab 2

**1.** Buat project `belajar-flowspec` dan jalankan `apitest dsl lint .`

**2.** Edit `env/dev.flow` agar `base_url` mengarah ke JSONPlaceholder

**3.** Jalankan `apitest config show --env dev` — pastikan `base_url` benar

---

**Lanjut →** [Bab 3 — Request Pertama](03-request-pertama.md)
