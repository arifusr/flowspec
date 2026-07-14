# Bab 8 — Control Flow

**Estimasi waktu:** 25 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Kondisi `when` / `unless`
- Loop `repeat` dan `for`
- `wait` dan `retry until`
- Kapan pakai masing-masing

---

## 8.1 Kondisi: `when` dan `unless`

### `when` — jalankan step hanya jika kondisi true

```flow
step "Get user detail" {
  when user_id                    // skip jika user_id kosong/tidak ada
  run GetUser(user_id)
}

step "Verify admin role" {
  when role == "admin"
  run AdminDashboard
}
```

### `unless` — kebalikan when

```flow
step "Create new user" {
  unless user_exists              // skip jika user sudah ada
  run CreateUser
}
```

💡 **Tip:** Selalu pakai `when user_id` setelah create — lindungi step berikutnya jika create gagal.

---

## 8.2 Loop: `repeat`

Jalankan block N kali:

```flow
flow BulkCreatePosts {
  repeat 5 {
    let post_title = "Post {{$uuid}}"
    run CreatePost
    expect status 201
  }
}
```

---

## 8.3 Loop: `for` — Iterasi Array

### Inline array

```flow
flow TestMultipleRoles {
  for role in ["admin", "user", "guest"] {
    let expected_status = role == "admin" ? 200 : 403
    run AccessAdminPanel {
      header X-Test-Role = role
    }
    expect status expected_status
  }
}
```

### Dari CSV (data-driven — detail di Bab 10)

```flow
for row in csv("data/users.csv") {
  let user_name  = row.name
  let user_email = row.email
  run CreateUser
}
```

---

## 8.4 Wait — Jeda Antar Step

API async butuh waktu proses:

```flow
step "Create export job" {
  run CreateExportJob
  let job_id = last.json("$.data.id")
}

step "Wait for processing" {
  wait 3s                         // pause 3 detik
}

step "Check job status" {
  run GetJobStatus(job_id)
}
```

Syntax waktu: `500ms`, `3s`, `1m`

---

## 8.5 Retry Until — Polling

Lebih robust dari `wait` fixed — ulangi sampai kondisi terpenuhi:

```flow
step "Wait until order confirmed" {
  retry 10 times every 2s until json "$.data.status" == "confirmed" {
    run GetOrder(order_id)
  }
}
```

| Part | Arti |
|---|---|
| `retry 10 times` | Maks 10 percobaan |
| `every 2s` | Jeda 2 detik antar percobaan |
| `until json "..." == "..."` | Stop jika kondisi terpenuhi |
| Block `{ run ... }` | Request yang diulang |

Jika 10x retry gagal → step FAIL dengan pesan:

```
Retry exhausted after 10 attempts (20s)
Last response: { "data": { "status": "processing" } }
Expected: status == "confirmed"
```

---

## 8.6 Kombinasi: Login + Retry + When

```flow
flow AsyncWorkflow {
  step "Login" {
    run Login
  }

  step "Submit job" {
    run SubmitJob
    let job_id = last.json("$.id")
  }

  step "Poll job completion" {
    when job_id
    retry 15 times every 3s until json "$.status" == "done" {
      run GetJobStatus(job_id)
    }
  }

  step "Download result" {
    when job_id
    run DownloadJobResult(job_id)
    expect status 200
  }

  teardown {
    ignore_fail
    when job_id
    run CancelJob(job_id)
  }
}
```

---

## 8.7 Parallel (Fase 2)

Jalankan beberapa request bersamaan:

```flow
step "Fetch all lists" {
  parallel {
    run ListUsers
    run ListProducts
    run ListOrders
  }
}
```

💡 Untuk pemula, fokus sequential dulu. Parallel untuk optimasi nanti.

---

## Ringkasan Bab 8

| Syntax | Fungsi |
|---|---|
| `when condition` | Skip step jika false |
| `unless condition` | Skip step jika true |
| `repeat N { ... }` | Ulangi N kali |
| `for x in [...] { ... }` | Iterasi array |
| `wait 3s` | Pause |
| `retry N times every Xs until ...` | Polling |

---

## Latihan Bab 8

**1.** Buat flow: create post → `when post_id` → get post.

**2.** Buat flow `repeat 3` create post dengan title unik (`{{$uuid}}`).

**3.** Buat flow dengan `wait 2s` antar 2 request — bandingkan total duration.

---

**Lanjut →** [Bab 9 — Reuse & Composition](09-reuse-composition.md)
