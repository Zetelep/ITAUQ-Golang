# API Specification — Automated Usability Evaluation (ITAUQ-Based)

Base URL: `https://api.yourapp.com/v1`
Backend: Golang REST API · Database: Supabase (PostgreSQL) · Auth: JWT (Supabase Auth)

---

## 1. Conventions

### 1.1 Authentication

Login itself is **not** part of this API — the React frontend authenticates directly against Supabase Auth (`supabase.auth.signInWithPassword`, etc.) and gets a JWT back from Supabase, not from the Golang backend.

Two auth modes are used across this API:

| Mode | Used by | How |
|---|---|---|
| **Bearer JWT** | Super Admin, Administrator | React sends `Authorization: Bearer <access_token>` (the JWT issued by Supabase Auth) on every request. The Golang backend only **verifies** it — checks the signature against Supabase's JWT secret/JWKS, reads `sub` as the user id, and looks up `role`/`is_active` from `profiles` for authorization. The backend never generates or refreshes this token. |
| **Link token** | Respondent (anonymous) | Token embedded in the public evaluation URL, e.g. `/public/evaluation/{token}/...` — no login, no Supabase Auth involved at all |

Every authenticated endpoint below lists the roles allowed to call it. `Super Admin` implicitly has access to everything an `Administrator` has, unless the endpoint is explicitly ownership-scoped. Token refresh, logout, and session management are all handled client-side by the Supabase SDK — nothing for the backend to implement there.

### 1.2 Standard response envelope

```json
// success
{
  "success": true,
  "data": { },
  "meta": { }            // optional, e.g. pagination
}

// error
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human readable message",
    "details": { }        // optional, e.g. field-level errors
  }
}
```

### 1.3 Standard HTTP status codes

| Code | Meaning |
|---|---|
| 200 | OK |
| 201 | Created |
| 204 | No content (successful delete) |
| 400 | Validation error |
| 401 | Missing/invalid token |
| 403 | Authenticated but not allowed (wrong role / not the owner) |
| 404 | Resource not found |
| 409 | Conflict (e.g. duplicate pending application, evaluation link inactive) |
| 422 | Semantically invalid (e.g. score out of range) |
| 500 | Server error |

### 1.4 Pagination

List endpoints accept `?page=1&page_size=20` and return:

```json
"meta": { "page": 1, "page_size": 20, "total": 57, "total_pages": 3 }
```

### 1.5 Use case → endpoint map

| Use case (diagram) | Endpoint(s) |
|---|---|
| Submit account request (new) | `POST /applications` |
| Review submissions (new) | `GET /applications`, `PATCH /applications/{id}/approve`, `PATCH /applications/{id}/reject` |
| Edit own profile (new) | `GET /profiles/me`, `PATCH /profiles/me` |
| Manage Administrator | `GET/POST/PATCH/DELETE /administrators` |
| Manage Questionnaire and Task Scenario | `/questionnaires`, `/questionnaires/{id}/task-scenarios` |
| Generate Link for Respondent | `POST /questionnaires/{id}/evaluation-links` |
| Fill Questionnaire & Task Scenario | `/public/evaluation/{token}/...` |
| Generate Evaluation Report | `GET /questionnaires/{id}/report` |
| View All Evaluation Result | `GET /respondents` (Super Admin) |
| View Own Evaluation Result | `GET /respondents` (Administrator, auto-scoped) |

---

## 2. Administrator Applications

Public submission → Super Admin review → account granted on approval. Maps to your `administrator_applications` table.

### `POST /applications`
Prospective Administrator submits a request. No login required.

**Auth:** none (public)
**Body:**
```json
{
  "full_name": "Siti Aminah",
  "email": "siti@example.com",
  "phone": "0812xxxxxxx",
  "institution": "Universitas ABC",
  "occupation": "Dosen"
}
```
**Response `201`:**
```json
{ "success": true, "data": { "id": "uuid", "status": "pending", "created_at": "..." } }
```
**Errors:** `409` if an application from this email is already `pending`.

### `GET /applications`
List submissions, for review.

**Auth:** Super Admin only
**Query:** `?status=pending|approved|rejected&page=1&page_size=20`
**Response `200`:** array of applications, newest first.

### `GET /applications/{id}`
**Auth:** Super Admin only · Full detail of one submission.

### `PATCH /applications/{id}/approve`
Approves the application: creates the Supabase Auth user (email invite to set password — the applicant then sets their own password directly through Supabase, not through this API), creates the `profiles` row (`role = administrator`, `application_id = {id}`), and marks the application `approved`. Should run as one transaction/flow on the backend using Supabase's admin/service-role API.

**Auth:** Super Admin only
**Body:** *(none required, or optionally override)*
```json
{ "review_note": "Diverifikasi, sesuai dengan data kampus." }
```
**Response `200`:**
```json
{
  "success": true,
  "data": {
    "application": { "id": "uuid", "status": "approved", "reviewed_at": "..." },
    "administrator": { "id": "uuid", "email": "siti@example.com", "role": "administrator" }
  }
}
```
**Errors:** `409` if the application is not `pending`.

### `PATCH /applications/{id}/reject`
**Auth:** Super Admin only
**Body:**
```json
{ "review_note": "Data institusi tidak dapat diverifikasi." }
```
**Response `200`:** updated application with `status: "rejected"`.

---

## 3. Profile (self-service)

Lets the logged-in user (Super Admin or Administrator) fetch and edit their **own** `profiles` row — e.g. an Administrator updating their display name. Separate from §4 `/administrators`, which is Super Admin managing *other* people's accounts.

### `GET /profiles/me`
**Auth:** Bearer (Super Admin or Administrator)
**Response `200`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "full_name": "Siti Aminah",
    "email": "siti@example.com",
    "role": "administrator",
    "is_active": true,
    "created_at": "2026-06-01T08:00:00Z"
  }
}
```

### `PATCH /profiles/me`
Edits the caller's own profile. Only `full_name` is editable here — `role` and `is_active` are Super-Admin-controlled (via `/administrators`), and `email` is tied to the Supabase Auth identity, so email changes go through Supabase's own `updateUser` flow on the frontend, not this API.

**Auth:** Bearer (Super Admin or Administrator)
**Body:**
```json
{ "full_name": "Siti Aminah Putri" }
```
**Response `200`:** the updated profile.
**Errors:** `400` if `full_name` is blank; any other field in the body is ignored or rejected with `VALIDATION_ERROR`.

---

## 4. Administrators (account management)

Maps to "Manage Administrator". Operates on `profiles` where `role = administrator`.

### `GET /administrators`
**Auth:** Super Admin only · List all administrator accounts.
**Query:** `?is_active=true&page=1&page_size=20`

### `GET /administrators/{id}`
**Auth:** Super Admin only

### `POST /administrators`
Direct account creation, bypassing the application flow — optional convenience for Super Admin (e.g. onboarding staff directly).

**Auth:** Super Admin only
**Body:**
```json
{ "full_name": "Rudi Hartono", "email": "rudi@example.com" }
```
**Response `201`:** created administrator, invite email sent. `application_id` will be `null`.

### `PATCH /administrators/{id}`
Update name or activate/deactivate an account (recommended over hard delete, to preserve referential history of their questionnaires/respondents).

**Auth:** Super Admin only
**Body:** any of `{ "full_name": "...", "is_active": false }`

### `DELETE /administrators/{id}`
Hard delete (cascades to their questionnaires, task scenarios, evaluation links, respondents — see schema `on delete cascade`). Prefer `PATCH { is_active: false }` unless the account must be fully purged.

**Auth:** Super Admin only · **Response:** `204`

---

## 5. Questionnaires

Maps to "Manage Questionnaire and Task Scenario". An "evaluation project" for one app; the 30 ITAUQ questions themselves are **not** in the DB — they're served from the fixed `itauq-v1` JSON shipped with the backend (see §9).

### `GET /questionnaires`
**Auth:** Bearer (Administrator or Super Admin)
- Administrator: automatically scoped to `administrator_id = caller.id`
- Super Admin: sees all; optional `?administrator_id=uuid` filter

**Query:** `?status=draft|active|closed&page=1&page_size=20`

### `POST /questionnaires`
**Auth:** Administrator (creates as themselves)
**Body:**
```json
{
  "title": "Evaluasi Usability App Wisata Kalsel",
  "app_name": "WisataKu",
  "description": "Evaluasi tahap 1 untuk skripsi",
  "status": "draft"
}
```
`itauq_version` defaults to `"itauq-v1"` if omitted.
**Response `201`:** created questionnaire.

### `GET /questionnaires/{id}`
**Auth:** Owning Administrator or Super Admin · `403` if a different Administrator requests it.

### `PATCH /questionnaires/{id}`
**Auth:** Owning Administrator only
**Body:** any subset of `title`, `app_name`, `description`, `status`.

### `DELETE /questionnaires/{id}`
**Auth:** Owning Administrator only · cascades to its task scenarios, evaluation links, respondents. **Response:** `204`

---

## 6. Task Scenarios

Nested under a questionnaire.

### `GET /questionnaires/{id}/task-scenarios`
**Auth:** Owning Administrator or Super Admin

### `POST /questionnaires/{id}/task-scenarios`
**Auth:** Owning Administrator
**Body:**
```json
{
  "title": "Mencari destinasi wisata terdekat",
  "instruction": "Gunakan fitur pencarian untuk menemukan tempat wisata dalam radius 5km.",
  "task_order": 1
}
```

### `PATCH /task-scenarios/{id}`
**Auth:** Owning Administrator (ownership resolved via parent questionnaire)

### `DELETE /task-scenarios/{id}`
**Auth:** Owning Administrator · **Response:** `204`

---

## 7. Evaluation Links

Maps to "Generate Link for Respondent".

### `GET /questionnaires/{id}/evaluation-links`
**Auth:** Owning Administrator or Super Admin

### `POST /questionnaires/{id}/evaluation-links`
Generates a new public token/link for this questionnaire.

**Auth:** Owning Administrator
**Body:**
```json
{ "expires_at": "2026-12-31T23:59:59Z" }
```
(`expires_at` optional; omit for a link that doesn't expire.)
**Response `201`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "token": "8f3ka92j",
    "url": "https://itauq.site/e/8f3ka92j",
    "is_active": true,
    "expires_at": null
  }
}```

### `PATCH /evaluation-links/{id}`
Deactivate a link or change its expiry.

**Auth:** Owning Administrator
**Body:** `{ "is_active": false }`

### `DELETE /evaluation-links/{id}`
**Auth:** Owning Administrator · **Response:** `204`

---

## 8. Public Respondent Flow
 
Maps to "Fill Questionnaire & Task Scenario". No login — identified only by the link `token` and, after starting, a `respondent_id`. These endpoints write through the backend's service role, per the RLS design (respondents/answers have no public Supabase policies).
 
### `GET /public/evaluation/{token}`
Loads everything the respondent-facing app needs to render the flow.
 
**Auth:** none
**Response `200`:**
```json
{
  "success": true,
  "data": {
    "questionnaire": { "title": "Evaluasi Usability App Wisata Kalsel", "app_name": "WisataKu" },
    "task_scenarios": [
      { "id": "uuid", "title": "Mencari destinasi wisata terdekat", "instruction": "...", "task_order": 1 }
    ],
    "itauq": { "version": "itauq-v1", "scale_min": 1, "scale_max": 7, "questions": [ /* 30 items from the JSON file, {AppName} already substituted */ ] },
    "sus": { "version": "sus-v1", "scale_min": 1, "scale_max": 5, "questions": [ /* 10 fixed SUS items about the evaluation website itself */ ] }
  }
}
```
**Errors:** `404` unknown token, `409` link `is_active = false` or past `expires_at`.
 
### `POST /public/evaluation/{token}/respondents`
Submits respondent identity, starts a session.
 
**Auth:** none
**Body:**
```json
{ "name": "Ahmad", "email": "ahmad@example.com", "age": 22, "gender": "male", "occupation": "Mahasiswa" }
```
**Response `201`:**
```json
{ "success": true, "data": { "respondent_id": "uuid", "started_at": "..." } }
```
 
### `POST /public/evaluation/{token}/respondents/{respondent_id}/task-attempts`
Submits results for the task scenarios (batch, one call per task or all at once).
 
**Auth:** none (respondent_id acts as the session key)
**Body:**
```json
{
  "attempts": [
    { "task_scenario_id": "uuid", "is_success": true, "duration_seconds": 42, "notes": "" }
  ]
}
```
**Response `201`:** echoes stored attempts.
 
### `POST /public/evaluation/{token}/respondents/{respondent_id}/answers`
Submits ITAUQ Likert answers, typically all 30 at once.
 
**Auth:** none
**Body:**
```json
{
  "answers": [
    { "item_id": 1, "category": "Attractiveness", "score": 6 },
    { "item_id": 2, "category": "Attractiveness", "score": 7 }
  ]
}
```
`item_id`/`category`/`score` are validated server-side against the shipped `itauq-v1` JSON before insert (matches the DB check constraints `item_id 1–30`, `score 1–7`).
**Response `201`:** echoes stored answers. **Errors:** `422` if any `item_id`/`category` pair doesn't match the instrument, or `score` is out of range.
 
### `POST /public/evaluation/{token}/respondents/{respondent_id}/sus-answers`
Submits the System Usability Scale (SUS) answers — the respondent's rating of the **evaluation website itself** (not the app being evaluated). Filled once, all 10 items, right after the ITAUQ answers.
 
**Auth:** none
**Body:**
```json
{
  "answers": [
    { "item_id": 1, "score": 4 },
    { "item_id": 2, "score": 2 }
  ]
}
```
`item_id`/`score` are validated server-side against the fixed 10-item SUS instrument (matches the DB check constraints `item_id 1–10`, `score 1–5`).
**Response `201`:** echoes stored answers. **Errors:** `422` if `item_id` isn't 1–10 or `score` is outside 1–5.
 
### `POST /public/evaluation/{token}/respondents/{respondent_id}/submit`
Finalizes the session (sets `submitted_at`). Backend should verify all 30 ITAUQ answers, all 10 SUS answers, and all task attempts exist before accepting.
 
**Auth:** none
**Response `200`:** `{ "success": true, "data": { "submitted_at": "..." } }`
**Errors:** `400` if incomplete (missing ITAUQ answers, SUS answers, or task attempts).
 
---
 
## 9. Instruments (reference, not DB-backed)
 
Both instruments below are fixed and shipped as constants/JSON in your codebase — not stored or manageable in the database (see schema notes on `questionnaire_answers` and `sus_answers`).
 
### `GET /instruments/itauq`
Serves the fixed 30-question ITAUQ instrument straight from the backend's bundled JSON — useful for the Administrator-facing frontend to preview questions, and reused internally by `GET /public/evaluation/{token}`.
 
**Auth:** Bearer (Administrator or Super Admin)
**Response `200`:** the itauq-v1 JSON as uploaded (metadata + 30 questions).
 
> Note: fix item `id: 9` (currently blank `text`) in the source JSON before wiring this endpoint up.
 
### `GET /instruments/sus`
Serves the fixed 10-item SUS instrument (about the evaluation website itself).
 
**Auth:** Bearer (Administrator or Super Admin)
**Response `200`:**
```json
{
  "success": true,
  "data": {
    "version": "sus-idn",
    "scale_min": 1,
    "scale_max": 5,
    "questions": [
      { "id": 1, "text": "Saya pikir saya akan sering menggunakan sistem ini.", "polarity": "positive" },
      { "id": 2, "text": "Saya merasa sistem ini rumit untuk digunakan.", "polarity": "negative" }
    ]
  }
}
```
`polarity` tells the frontend/backend which items are odd/even for scoring — matches `v_respondent_sus_score`'s `item_id % 2` rule, so keep item order fixed at 1–10 exactly as the standard SUS defines it.
 
---
 
## 10. Evaluation Results & Reports
 
Maps to "View Own/All Evaluation Result" and "Generate Evaluation Report". Backed by the `v_evaluation_report`, `v_respondent_category_scores`, `v_respondent_overall_score`, `v_respondent_task_success`, `v_respondent_sus_score` views.
 
### `GET /respondents`
List respondents with their computed scores. Automatically scoped:
- **Administrator** → only respondents of their own questionnaires ("View Own Evaluation Result")
- **Super Admin** → all respondents across all administrators ("View All Evaluation Result")
**Auth:** Bearer (Administrator or Super Admin)
**Query:** `?questionnaire_id=uuid&administrator_id=uuid(super_admin only)&page=1&page_size=20`
**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "respondent_id": "uuid",
      "respondent_name": "Ahmad",
      "questionnaire_title": "Evaluasi Usability App Wisata Kalsel",
      "app_name": "WisataKu",
      "administrator_name": "Siti Aminah",
      "overall_usability_score": 82.14,
      "task_success_rate_pct": 100.0,
      "evaluation_website_sus_score": 77.5,
      "submitted_at": "2026-08-10T09:15:00Z"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 12, "total_pages": 1 }
}
```
 
### `GET /respondents/{id}`
Full detail: identity, per-category ITAUQ scores, per-task results, and the SUS score for the evaluation website.
 
**Auth:** Owning Administrator (via the respondent's questionnaire) or Super Admin
**Response `200`:**
```json
{
  "success": true,
  "data": {
    "respondent": { "id": "uuid", "name": "Ahmad", "age": 22, "gender": "male", "occupation": "Mahasiswa" },
    "overall_usability_score": 82.14,
    "category_scores": [
      { "category": "Attractiveness", "avg_raw_score": 6.33, "normalized_score": 88.89 },
      { "category": "Efficiency", "avg_raw_score": 5.67, "normalized_score": 77.78 }
    ],
    "task_results": [
      { "task_scenario_id": "uuid", "title": "Mencari destinasi wisata terdekat", "is_success": true, "duration_seconds": 42 }
    ],
    "evaluation_website_sus": {
      "answered_items": 10,
      "sus_score": 77.5
    }
  }
}
```
 
### `GET /questionnaires/{id}/report`
Aggregate report across **all** respondents of one questionnaire — the actual "Generate Evaluation Report" output.
 
**Auth:** Owning Administrator or Super Admin
**Response `200`:**
```json
{
  "success": true,
  "data": {
    "questionnaire": { "id": "uuid", "title": "Evaluasi Usability App Wisata Kalsel", "app_name": "WisataKu" },
    "respondent_count": 12,
    "overall_usability_score_avg": 79.4,
    "category_averages": [
      { "category": "Attractiveness", "normalized_score_avg": 85.1 },
      { "category": "Trust", "normalized_score_avg": 71.3 }
    ],
    "task_success_rate_avg": 91.7,
    "evaluation_website_sus_score_avg": 76.8
  }
}
```
 
### `GET /questionnaires/{id}/report/export`
Same data as above, rendered as a file.
 
**Auth:** Owning Administrator or Super Admin
**Query:** `?format=pdf|csv`
**Response `200`:** binary file (`Content-Type: application/pdf` or `text/csv`).
 
---
## 11. Role permission matrix

| Endpoint group | Super Admin | Administrator | Respondent (public) |
|---|:---:|:---:|:---:|
| `/applications` (submit) | – | – | ✅ (anonymous) |
| `/applications` (review) | ✅ | – | – |
| `/profiles/me` | ✅ own row | ✅ own row | – |
| `/administrators` | ✅ | – | – |
| `/questionnaires`, `/task-scenarios` | ✅ view all | ✅ own only | – |
| `/evaluation-links` | ✅ view all | ✅ own only | – |
| `/public/evaluation/*` | – | – | ✅ |
| `/instruments/itauq` | ✅ | ✅ | – |
| `/respondents`, `/respondents/{id}` | ✅ all | ✅ own only | – |
| `/questionnaires/{id}/report*` | ✅ all | ✅ own only | – |

---

## 12. Error codes reference

| `error.code` | HTTP | Meaning |
|---|---|---|
| `VALIDATION_ERROR` | 400 | Request body failed validation |
| `UNAUTHORIZED` | 401 | Missing or expired token |
| `FORBIDDEN` | 403 | Authenticated but wrong role or not the resource owner |
| `NOT_FOUND` | 404 | Resource doesn't exist |
| `DUPLICATE_APPLICATION` | 409 | A pending application already exists for this email |
| `LINK_INACTIVE` | 409 | Evaluation link deactivated or expired |
| `ITEM_OUT_OF_RANGE` | 422 | `item_id` not in 1–30 or `score` outside 1–7 |
| `INCOMPLETE_SUBMISSION` | 400 | Respondent tried to submit before answering all items/tasks |
| `INTERNAL_ERROR` | 500 | Unhandled server error |
