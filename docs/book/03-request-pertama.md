# Bab 3 — Request Pertama

**Estimasi waktu:** 20 menit

---

## Apa yang Kamu Pelajari di Bab Ini

- Tulis request GET sederhana
- Tulis request POST dengan body JSON
- Jalankan request dari terminal
- Baca output pass/fail

---

## 3.1 Request GET — Paling Sederhana

Buat file `requests/posts/list-posts.flow`:

```flow
request ListPosts {
  GET "{{base_url}}/posts"

  expect status 200
}
```

**Penjelasan:**

| Baris | Arti |
|---|---|
| `request ListPosts` | Nama request (bebas, PascalCase direkomendasikan) |
| `GET "{{base_url}}/posts"` | HTTP GET ke URL |
| `{{base_url}}` | Variable dari `env/dev.flow` |
| `expect status 200` | Response harus status 200 |

Jalankan:

```bash
apitest run requests/posts/list-posts.flow --env dev
```

Output sukses:

```
✓ ListPosts
  GET https://jsonplaceholder.typicode.com/posts
  Status: 200 OK  (142ms)
  Assertions: 1 passed

Summary: 1 passed, 0 failed
```

---

## 3.2 Request GET dengan Assertion Body

Tambahkan validasi isi response:

```flow
request ListPosts {
  GET "{{base_url}}/posts"

  expect status 200
  expect json "$" is array          // root response adalah array
  expect json "$[0].id" exists      // item pertama punya field id
  expect time < 3s                  // selesai dalam 3 detik
}
```

💡 **Tip JSONPath singkat:**

| JSONPath | Arti |
|---|---|
| `$` | Root (seluruh body) |
| `$.data` | Field `data` di root |
| `$[0]` | Item pertama array |
| `$.items[2].name` | Nested field |

---

## 3.3 Request POST — Kirim Data

Buat `requests/posts/create-post.flow`:

```flow
request CreatePost {
  POST "{{base_url}}/posts"

  header Content-Type = "application/json"

  body json {
    title: "Belajar FlowSpec"
    body:  "Request POST pertama saya"
    userId: 1
  }

  expect status 201
  expect json "$.title" == "Belajar FlowSpec"
}
```

⚠️ **Perhatian:** JSONPlaceholder selalu return `201` meski data tidak benar-benar disimpan — itu normal untuk API demo.

---

## 3.4 Request dengan Path Parameter

Buat `requests/posts/get-post.flow`:

```flow
request GetPost(post_id) {
  GET "{{base_url}}/posts/{{post_id}}"

  expect status 200
  expect json "$.id" == "{{post_id}}"
}
```

`post_id` adalah **parameter** — nilainya diisi saat request di-run:

```bash
apitest run requests/posts/get-post.flow --env dev --var post_id=1
```

Atau dari dalam flow (Bab 7):

```flow
// Literal value — langsung masukkan angka/string
run GetPost("1")
run GetPost("42")

// Variable — isi dari extract atau let
let my_id = "5"
run GetPost(my_id)

// Dari step sebelumnya (extract)
run GetPost(post_id)
```

💡 **Tip:** Parameterized request sangat powerful untuk menghindari duplikasi file. Satu request bisa dipanggil berkali-kali dengan argument berbeda — misalnya dropdown yang sama tapi filter berbeda:

```flow
request GetUserRole(role_id) {
  GET "{{base_url}}/combo/user-role?q=&role={{role_id}}"
  expect status 200
}

// Dari flow:
step "PIC Product"   { run GetUserRole("8") }
step "PIC Packaging" { run GetUserRole("9") }
```

---

## 3.5 Request dengan Query Parameter

```flow
request ListPostsByUser {
  GET "{{base_url}}/posts?userId={{user_id}}"

  expect status 200
  expect json "$" is array
}
```

Alternatif syntax (lebih terbaca):

```flow
request ListPostsByUser {
  GET "{{base_url}}/posts"
  query userId = "{{user_id}}"

  expect status 200
}
```

---

## 3.6 Header

```flow
request SecureEndpoint {
  GET "{{base_url}}/api/protected"

  header Authorization = "Bearer {{access_token}}"
  header Accept        = "application/json"
  header X-Request-Id  = "{{$uuid}}"

  expect status 200
}
```

Header `X-Request-Id` pakai dynamic variable `{{$uuid}}` — UUID baru setiap run.

---

## 3.7 Tipe Body Lainnya

### Form URL-encoded

```flow
request Login {
  POST "{{base_url}}/auth/login"

  body form {
    username: "admin"
    password: "{{password}}"
  }

  expect status 200
}
```

### Raw text / XML

```flow
request SendXml {
  POST "{{base_url}}/xml-endpoint"

  body raw {
    content: "<user><name>John</name></user>"
    content_type: "application/xml"
  }

  expect status 200
}
```

### Upload file (multipart)

```flow
request UploadAvatar {
  POST "{{base_url}}/upload"

  body multipart {
    name: "John"
    file: "./fixtures/avatar.png"
  }

  expect status 200
}
```

Untuk pemula, **90% kasus cukup `body json { ... }`**.

---

## 3.8 Tag — Label untuk Filter

```flow
@tags(posts, smoke)

request ListPosts {
  GET "{{base_url}}/posts"
  expect status 200
}
```

Tag dipakai saat run selective:

```bash
apitest run requests/ --env dev --tags smoke
```

---

## 3.9 Debug: Lihat Request Sebelum Dikirim

```bash
# Preview URL & headers setelah variable di-resolve
apitest dsl show requests/posts/list-posts.flow --env dev

# Verbose — tampilkan request + response
apitest run requests/posts/list-posts.flow --env dev -v
```

---

## 3.10 Import dari cURL — Jalan Pintas

Punya cURL command dari dokumentasi API atau browser DevTools? Import langsung:

### Import satu command

```bash
apitest import curl \
  'curl -X GET https://jsonplaceholder.typicode.com/posts -H "Accept: application/json"' \
  --output requests/posts/list-posts.flow
```

Output:
```
✓ Imported to requests/posts/list-posts.flow
  GET https://jsonplaceholder.typicode.com/posts
```

### Import dari file (banyak command sekaligus)

Simpan beberapa cURL command ke file teks — satu command per blok, pisah baris kosong:

`curls/posts-api.txt`:

```bash
# List semua posts
curl -X GET https://jsonplaceholder.typicode.com/posts \
  -H "Accept: application/json"

# Create post baru
curl -X POST https://jsonplaceholder.typicode.com/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"Hello","body":"World","userId":1}'

# Get post by ID
curl https://jsonplaceholder.typicode.com/posts/1
```

Import semua sekaligus:

```bash
apitest import curl --file curls/posts-api.txt --output-dir requests/posts/
```

Output:

```
Found 3 curl command(s) in curls/posts-api.txt

✓ Imported to requests/posts/get-posts.flow
  GET https://jsonplaceholder.typicode.com/posts
✓ Imported to requests/posts/create-posts.flow
  POST https://jsonplaceholder.typicode.com/posts
✓ Imported to requests/posts/get1.flow
  GET https://jsonplaceholder.typicode.com/posts/1
```

💡 **Tip:** Setelah import, review file yang dihasilkan. Ganti hardcoded URL ke `{{base_url}}`, tambahkan assertion yang lebih spesifik, dan rename file/request sesuai kebutuhan.

⚠️ **Perhatian:** Import curl menghasilkan assertion default (`expect status 200` atau `201`). Selalu tambahkan assertion body yang sesuai bisnis kamu.

---

## Ringkasan Bab 3

| Syntax / Command | Fungsi |
|---|---|
| `GET/POST/PUT/PATCH/DELETE "url"` | HTTP method + URL |
| `header Key = "value"` | Request header |
| `body json { ... }` | Request body JSON |
| `query key = "value"` | Query string |
| `expect status 200` | Assert status code |
| `request Name(param)` | Request dengan parameter |
| `apitest import curl '<cmd>'` | Import dari cURL command |
| `apitest import curl --file <path>` | Import dari file berisi cURL |

---

## Latihan Bab 3

Kerjakan dengan JSONPlaceholder (`https://jsonplaceholder.typicode.com`).

**1.** Buat `ListUsers` — GET `/users`, expect status 200, expect array.

**2.** Buat `GetUser(user_id)` — GET `/users/{id}`, expect `$.email` exists.

**3.** Buat `CreatePost` — POST `/posts` dengan title & body, expect status 201.

**4.** Jalankan semua, pastikan pass.

**5.** Buat file `curls/latihan.txt` berisi 2 curl command (GET dan POST). Import dengan `apitest import curl --file`. Review file hasil import.

---

**Lanjut →** [Bab 4 — Environment & Variabel](04-environment-variabel.md)
