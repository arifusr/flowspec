# Bab 13 — Jawaban Latihan

Berikut jawaban dari semua latihan di Bab 1–12.

---

## Jawaban Bab 1

**1. Apa perbedaan utama `request` dan `flow`?**

`request` = satu panggilan HTTP (unit test satu endpoint).
`flow` = gabungan beberapa request jadi scenario bisnis (integration test).

**2. Mengapa file `.flow` lebih cocok untuk Git dibanding klik manual di GUI?**

- File teks bisa di-diff, review di PR, dan version control
- Bisa dijalankan otomatis di CI/CD tanpa intervensi manual
- Perubahan test terdokumentasi di commit history
- Tim bisa kolaborasi lewat branching dan merge

**3. Sebutkan 3 hal yang bisa kamu `expect` dari response API.**

- Status code (`expect status 200`)
- Body JSON (`expect json "$.data.id" exists`)
- Response time (`expect time < 500ms`)
- Header (`expect header Content-Type contains "json"`)

---

## Jawaban Bab 2

**1. Buat project dan lint**

```bash
mkdir belajar-flowspec
cd belajar-flowspec
apitest init
apitest dsl lint .
```

Output: `✓ All files valid`

**2. Edit env/dev.flow**

```flow
env dev {
  base_url = "https://jsonplaceholder.typicode.com"
}
```

**3. Verifikasi config**

```bash
apitest config show --env dev
```

Output:
```
Environment: dev
Variables:
  base_url = https://jsonplaceholder.typicode.com
```

---

## Jawaban Bab 3

**1. ListUsers**

```flow
// requests/users/list-users.flow

request ListUsers {
  GET "{{base_url}}/users"

  expect status 200
  expect json "$" is array
}
```

**2. GetUser(user_id)**

```flow
// requests/users/get-user.flow

request GetUser(user_id) {
  GET "{{base_url}}/users/{{user_id}}"

  expect status 200
  expect json "$.email" exists
}
```

**3. CreatePost**

```flow
// requests/posts/create-post.flow

request CreatePost {
  POST "{{base_url}}/posts"

  header Content-Type = "application/json"

  body json {
    title:  "Belajar FlowSpec"
    body:   "Ini post pertama saya"
    userId: 1
  }

  expect status 201
}
```

**4. Jalankan semua**

```bash
apitest run requests/users/list-users.flow --env dev
apitest run requests/users/get-user.flow --env dev --var user_id=1
apitest run requests/posts/create-post.flow --env dev
```

**5. Import dari file cURL**

Buat `curls/latihan.txt`:
```bash
# Get users
curl -X GET https://jsonplaceholder.typicode.com/users \
  -H "Accept: application/json"

# Create a comment
curl -X POST https://jsonplaceholder.typicode.com/comments \
  -H "Content-Type: application/json" \
  -d '{"postId":1,"name":"Test","email":"test@example.com","body":"Hello"}'
```

Import:
```bash
apitest import curl --file curls/latihan.txt --output-dir requests/imported/
```

Review file yang dihasilkan, tambahkan assertion, dan ganti hardcoded URL ke `{{base_url}}`.

---

## Jawaban Bab 4

**1. env/staging.flow**

```flow
env staging {
  base_url = "https://staging-api.example.com"
}
```

**2. Request dengan base_url**

```flow
request ListPosts {
  GET "{{base_url}}/posts"
  expect status 200
}
```

```bash
apitest run requests/posts/list-posts.flow --env dev
# → GET https://jsonplaceholder.typicode.com/posts

apitest run requests/posts/list-posts.flow --env staging
# → GET https://staging-api.example.com/posts
```

**3. Flow dengan UUID email**

```flow
flow UniqueEmails {
  let user_email = "test-{{$uuid}}@example.com"

  step "Create user" {
    run CreateUser
  }
}
```

Run 2 kali — email berbeda setiap kali:
```
Run 1: test-a1b2c3d4-...@example.com
Run 2: test-e5f6g7h8-...@example.com
```

**4. Secret redaction**

```bash
export API_TOKEN="sk-secret-12345"
apitest run flows/smoke.flow --env dev -vv
```

Output header: `Authorization: Bearer ***REDACTED***`

---

## Jawaban Bab 5

**1. ListPosts dengan banyak assertion**

```flow
request ListPosts {
  GET "{{base_url}}/posts"

  expect status 200
  expect json "$" is array
  expect json "$" length >= 10
  expect time < 3s
}
```

**2. GetPost(1) dengan assertions**

```flow
request GetPost(post_id) {
  GET "{{base_url}}/posts/{{post_id}}"

  expect status 200
  expect json "$.id" == 1
  expect json "$.title" exists
  expect json "$.userId" is number
}
```

**3. Assertion yang sengaja salah**

```flow
request WillFail {
  GET "{{base_url}}/posts/1"
  expect status 404      // sengaja salah — response sebenarnya 200
}
```

Output:
```
✗ WillFail
  Assertion failed [line 3]:
    expect status 404
    Expected: 404
    Actual:   200
```

**4. Regex assertion**

```flow
request GetPost(post_id) {
  GET "{{base_url}}/posts/{{post_id}}"

  expect status 200
  expect json "$.title" matches "^[a-z]"
}
```

---

## Jawaban Bab 6

**1. Flow: Create → Extract → Get → Assert**

```flow
import requests/posts/create-post.flow
import requests/posts/get-post.flow

flow PostCreateAndVerify {
  let post_title = "FlowSpec Test {{$uuid}}"

  step "Create post" {
    run CreatePost {
      body json {
        title:  post_title
        body:   "Test body"
        userId: 1
      }
    }
    let post_id = last.json("$.id")
  }

  step "Get and verify" {
    when post_id
    run GetPost(post_id)
    expect json "$.title" == post_title
  }
}
```

**2. Flow dengan fake token**

```flow
flow FakeAuthFlow {
  step "Set fake token" {
    let access_token = "fake-token-123"
  }

  step "Call protected endpoint" {
    run SecureEndpoint
    // Akan gagal 401 jika API benar-benar cek token
  }
}
```

**3. JSONPath salah**

Jika kamu extract `from json "$.id"` tapi response adalah `{ "data": { "id": 42 } }`:

```
Error: Extract failed
  Path "$.id" returned null
  Response: { "data": { "id": 42 } }
  Hint: Did you mean "$.data.id"?
```

---

## Jawaban Bab 7

**1. flows/post-read.flow**

```flow
import requests/posts/list-posts.flow
import requests/posts/get-post.flow

@tags(posts, read)

flow PostRead {
  description "List posts → get first post by id"

  step "List all posts" {
    run ListPosts
    let post_id = last.json("$[0].id")
  }

  step "Get post by id" {
    when post_id
    run GetPost(post_id)
    expect json "$.id" == post_id
  }
}
```

**2. flows/smoke.flow**

```flow
import requests/posts/list-posts.flow
import requests/users/list-users.flow
import requests/posts/get-post.flow

@tags(smoke)

flow SmokeTest {
  step "List posts"  { run ListPosts }
  step "List users"  { run ListUsers }
  step "Get post 1"  { run GetPost(1) }
}
```

**3. Flow CRUD dengan teardown**

```flow
flow PostCRUD {
  step "Create" {
    run CreatePost
    let post_id = last.json("$.id")
  }

  step "Read" {
    when post_id
    run GetPost(post_id)
  }

  teardown "Cleanup" {
    ignore_fail
    when post_id
    run DeletePost(post_id)
  }
}
```

**4. Run dengan tags**

```bash
apitest run flows/ --tags smoke --env dev
```

---

## Jawaban Bab 8

**1. Flow dengan `when`**

```flow
flow SafePostRead {
  step "Create post" {
    run CreatePost
    let post_id = last.json("$.id")
  }

  step "Get post" {
    when post_id
    run GetPost(post_id)
    expect status 200
  }
}
```

**2. Repeat 3x dengan UUID**

```flow
flow BulkCreatePosts {
  repeat 3 {
    let post_title = "Bulk Post {{$uuid}}"

    step "Create post" {
      run CreatePost {
        body json {
          title:  post_title
          body:   "Auto generated"
          userId: 1
        }
      }
      expect status 201
    }
  }
}
```

**3. Wait antar request**

```flow
flow WaitDemo {
  step "First request" {
    run ListPosts
  }

  step "Wait 2 seconds" {
    wait 2s
  }

  step "Second request" {
    run ListUsers
  }
}
```

Total duration ≈ request1_time + 2s + request2_time.

---

## Jawaban Bab 9

**1. Auth block di 3 request**

```flow
// shared/auth.flow
auth BearerAuth {
  header Authorization = "Bearer {{access_token}}"
}
```

```flow
// Request 1
request ListUsers {
  use auth BearerAuth
  GET "{{base_url}}/users"
  expect status 200
}

// Request 2
request GetUser(user_id) {
  use auth BearerAuth
  GET "{{base_url}}/users/{{user_id}}"
  expect status 200
}

// Request 3
request CreateUser {
  use auth BearerAuth
  POST "{{base_url}}/users"
  body json { name: "Test" }
  expect status 201
}
```

**2. CreateAdminUser extends CreateUser**

```flow
request CreateAdminUser extends CreateUser {
  body json {
    role: "admin"
  }
  expect json "$.data.role" == "admin"
}
```

**3. Fragment LoginSetup di 2 flow**

```flow
// shared/common-steps.flow
fragment LoginSetup {
  step "Login" {
    run Login
    let access_token = last.json("$.token")
  }
}

// flows/flow-a.flow
flow FlowA {
  use fragment LoginSetup
  step "Do something" { run ListUsers }
}

// flows/flow-b.flow
flow FlowB {
  use fragment LoginSetup
  step "Do other thing" { run ListPosts }
}
```

**4. Regression flow**

```flow
flow Regression {
  include flows/flow-a.flow
  include flows/flow-b.flow
}
```

---

## Jawaban Bab 10

**1. Inline table 3 user**

```flow
flow CreateMultipleUsers {
  for row in [
    { name: "Alice", email: "alice@test.com", role: "admin" },
    { name: "Bob",   email: "bob@test.com",   role: "user"  },
    { name: "Eve",   email: "eve@test.com",   role: "guest" }
  ] {
    step "Create {{row.name}} ({{row.role}})" {
      run CreateUser {
        body json {
          name:  row.name
          email: row.email
          role:  row.role
        }
      }
      expect status 201
      expect json "$.data.role" == row.role
    }
  }
}
```

**2. CSV data-driven posts**

File `data/posts.csv`:
```csv
title,body,userId
First Post,Hello World,1
Second Post,FlowSpec is great,2
Third Post,Data driven testing,1
```

```flow
flow CreatePostsFromCSV {
  for row in csv("data/posts.csv") {
    step "Create: {{row.title}}" {
      run CreatePost {
        body json {
          title:  row.title
          body:   row.body
          userId: row.userId
        }
      }
      expect status 201
    }
  }
}
```

**3. Negative email test**

```flow
flow InvalidEmailTest {
  for email in ["", "not-email", "@no-local.com", "no-domain@"] {
    step "Reject: {{email}}" {
      run CreateUser {
        body json { email: email, name: "Test" }
      }
      expect status 422
    }
  }
}
```

**4. Run dengan --fail-fast**

```bash
apitest run flows/create-posts-batch.flow --env dev --fail-fast
```

Behavior: stop di row pertama yang gagal, tidak lanjut ke row berikutnya.

---

## Jawaban Bab 11

**1. Verbose output**

```bash
apitest run requests/posts/list-posts.flow --env dev -vv
```

Akan muncul: full headers, body request, response headers, response body, assertion results.

**2. Typo dan lint**

```flow
// Sengaja typo
request ListPost {
  GET "{{base_url}}/posts"
  expect staus 200        // typo: staus → status
}
```

```bash
apitest dsl lint requests/posts/list-posts.flow
```

Output:
```
Error [requests/posts/list-posts.flow:3:3]
  Unknown keyword 'staus' — did you mean 'status'?
```

**3. GitHub Actions file**

```yaml
# .github/workflows/api-test.yml
name: API Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: curl -fsSL https://example.com/install.sh | sh
      - run: apitest dsl lint .
      - run: apitest run flows/smoke.flow --env dev --report junit
        env:
          API_TOKEN: ${{ secrets.API_TOKEN }}
```

**4. HTML report**

```bash
apitest run flows/ --env dev --report html
# Buka: reports/report-<timestamp>.html
```

---

## Jawaban Bab 12

Bab 12 adalah hands-on project. Tidak ada satu jawaban "benar" — yang penting:

**Checklist minimum:**

- [ ] `apitest.flow` config global ada
- [ ] `env/dev.flow` dengan base_url
- [ ] Minimal 3 request file di `requests/`
- [ ] 1 flow smoke dengan tag `smoke`
- [ ] 1 flow CRUD dengan teardown
- [ ] `.gitignore` include `reports/`
- [ ] CI file ada (GitHub Actions / GitLab CI)
- [ ] Semua test pass: `apitest run flows/ --env dev`

**Contoh struktur minimal:**

```
my-project/
├── apitest.flow
├── env/
│   └── dev.flow
├── requests/
│   ├── list-posts.flow
│   ├── get-post.flow
│   └── create-post.flow
├── flows/
│   ├── smoke.flow
│   └── post-crud.flow
├── reports/
├── .gitignore
└── .github/
    └── workflows/
        └── api-test.yml
```

---

## Penutup

Selamat! Kamu sudah menyelesaikan buku **FlowSpec untuk Pemula**.

**Apa selanjutnya?**

- Baca [Spesifikasi DSL lengkap](../flowspec-dsl.md) untuk syntax detail
- Explore [contoh kode](../../examples/) untuk pattern real-world
- Mulai tulis test untuk API project kamu sendiri
- Integrasikan ke CI/CD tim kamu

💡 **Tips terakhir:** Mulai kecil (smoke test 3-5 endpoint), lalu perluas seiring waktu. Test yang jalan di CI lebih berharga dari test suite besar yang tidak pernah dijalankan.

Happy testing! 🚀
