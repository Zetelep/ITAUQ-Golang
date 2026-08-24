# ITAUQ API Documentation

> Automated Usability Evaluation System based on ITAUQ (Indonesian Tourism Application Usability Questionnaire)

**Base URL:** `http://localhost:8080` (development)  
**Backend:** Go (Gin Framework)  
**Database:** PostgreSQL (Supabase)  
**Authentication:** JWT (Supabase Auth) + Link Tokens (public respondents)

---

## Table of Contents

- [Conventions](#conventions)
- [Authentication](#authentication)
- [Endpoints](#endpoints)
  - [Applications](#1-applications)
  - [Questionnaires](#2-questionnaires)
  - [Task Scenarios](#3-task-scenarios)
  - [Evaluation Links](#4-evaluation-links)
  - [Public Respondent Flow](#5-public-respondent-flow)
  - [ITAUQ Instrument](#6-itauq-instrument)
- [Database Schema](#database-schema)
- [Error Codes](#error-codes)

---

## Conventions

### Standard Response Envelope

**Success:**
```json
{
  "success": true,
  "data": { },
  "meta": { }
}
```

**Error:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

### Pagination

List endpoints accept query parameters:
- `page` (default: 1)
- `page_size` (default: 20)

Response includes pagination metadata:
```json
{
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 57,
    "total_pages": 3
  }
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | OK |
| 201 | Created |
| 204 | No Content (successful delete) |
| 400 | Validation error |
| 401 | Missing/invalid token |
| 403 | Authenticated but not allowed |
| 404 | Resource not found |
| 409 | Conflict (e.g., duplicate pending application, evaluation link inactive) |
| 422 | Semantically invalid (e.g., score out of range) |
| 500 | Server error |

---

## Authentication

### Development Mode (No Database)

When `DATABASE_URL` is not set, the API uses header-based authentication:
- `X-User-ID`: User UUID
- `X-User-Role`: `administrator` or `super_admin`

### Production Mode (With Supabase)

Authentication uses JWT Bearer tokens issued by Supabase Auth:
```
Authorization: Bearer <access_token>
```

The backend verifies the JWT signature and looks up the user's role from the `profiles` table.

### Role-Based Access

| Role | Description |
|------|-------------|
| `super_admin` | Full access to all resources |
| `administrator` | Access to own resources only |
| Public (no auth) | Application submission only |

---

## Endpoints

### 1. Applications

Administrator account applications. Public submission, Super Admin review.

#### `POST /applications`

Submit a new administrator application (public, no auth required).

**Request Body:**
```json
{
  "full_name": "Siti Aminah",
  "email": "siti@example.com",
  "institution": "Universitas ABC",
  "occupation": "Dosen",
  "reason": "Ingin melakukan evaluasi usability aplikasi"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `full_name` | string | Yes | Applicant's full name |
| `email` | string | Yes | Valid email address |
| `institution` | string | No | Organization/university |
| `occupation` | string | No | Job title/role |
| `reason` | string | No | Reason for application |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "pending",
    "created_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Invalid input data
- `409 DUPLICATE_APPLICATION` - Pending application already exists for this email

---

#### `GET /applications`

List all applications with optional filtering.

**Auth:** Super Admin only

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `status` | string | Filter by: `pending`, `approved`, `rejected` |
| `page` | int | Page number (default: 1) |
| `page_size` | int | Items per page (default: 20) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "full_name": "Siti Aminah",
      "email": "siti@example.com",
      "institution": "Universitas ABC",
      "occupation": "Dosen",
      "status": "pending",
      "created_at": "2026-08-21T10:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

---

#### `GET /applications/:id`

Get a single application by ID.

**Auth:** Super Admin only

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "Siti Aminah",
    "email": "siti@example.com",
    "institution": "Universitas ABC",
    "occupation": "Dosen",
    "reason": "Ingin melakukan evaluasi usability",
    "status": "pending",
    "created_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `404 NOT_FOUND` - Application not found

---

#### `PATCH /applications/:id/approve`

Approve an application. Creates Supabase Auth user and profile.

**Auth:** Super Admin only

**Request Body (optional):**
```json
{
  "review_note": "Diverifikasi, sesuai dengan data kampus."
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "approved",
    "reviewed_at": "2026-08-21T11:00:00Z",
    "review_note": "Diverifikasi, sesuai dengan data kampus."
  }
}
```

**Errors:**
- `404 NOT_FOUND` - Application not found
- `409 APPLICATION_ALREADY_REVIEWED` - Application is not in `pending` status

---

#### `PATCH /applications/:id/reject`

Reject an application.

**Auth:** Super Admin only

**Request Body:**
```json
{
  "review_note": "Data institusi tidak dapat diverifikasi."
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "rejected",
    "reviewed_at": "2026-08-21T11:00:00Z",
    "review_note": "Data institusi tidak dapat diverifikasi."
  }
}
```

**Errors:**
- `404 NOT_FOUND` - Application not found
- `409 APPLICATION_ALREADY_REVIEWED` - Application is not in `pending` status

---

### 2. Questionnaires

Evaluation projects for applications. Each questionnaire contains ITAUQ questions and task scenarios.

#### `POST /questionnaires`

Create a new questionnaire.

**Auth:** Administrator (creates as themselves)

**Request Body:**
```json
{
  "title": "Evaluasi Usability App Wisata Kalsel",
  "app_name": "WisataKu",
  "description": "Evaluasi tahap 1 untuk skripsi",
  "status": "draft",
  "itauq_version": "itauq-v1"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Questionnaire title |
| `app_name` | string | Yes | Application being evaluated |
| `description` | string | No | Description of the evaluation |
| `status` | string | No | `draft`, `active`, or `closed` (default: `draft`) |
| `itauq_version` | string | No | ITAUQ version (default: `itauq-v1`) |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "administrator_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "title": "Evaluasi Usability App Wisata Kalsel",
    "app_name": "WisataKu",
    "description": "Evaluasi tahap 1 untuk skripsi",
    "itauq_version": "itauq-v1",
    "status": "draft",
    "created_at": "2026-08-21T10:30:00Z",
    "updated_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Invalid input data
- `401 UNAUTHORIZED` - User identity not found

---

#### `GET /questionnaires`

List questionnaires with filtering and pagination.

**Auth:** Administrator or Super Admin

- **Administrator:** Automatically scoped to own questionnaires
- **Super Admin:** Sees all; optional `administrator_id` filter

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `status` | string | Filter by: `draft`, `active`, `closed` |
| `administrator_id` | uuid | Filter by administrator (Super Admin only) |
| `page` | int | Page number (default: 1) |
| `page_size` | int | Items per page (default: 20) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "administrator_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
      "title": "Evaluasi Usability App Wisata Kalsel",
      "app_name": "WisataKu",
      "description": "Evaluasi tahap 1 untuk skripsi",
      "itauq_version": "itauq-v1",
      "status": "draft",
      "created_at": "2026-08-21T10:30:00Z",
      "updated_at": "2026-08-21T10:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

---

#### `GET /questionnaires/:id`

Get a single questionnaire by ID.

**Auth:** Owning Administrator or Super Admin

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "administrator_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "title": "Evaluasi Usability App Wisata Kalsel",
    "app_name": "WisataKu",
    "description": "Evaluasi tahap 1 untuk skripsi",
    "itauq_version": "itauq-v1",
    "status": "draft",
    "created_at": "2026-08-21T10:30:00Z",
    "updated_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `403 FORBIDDEN` - Not authorized to access this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

#### `PATCH /questionnaires/:id`

Update a questionnaire (partial update).

**Auth:** Owning Administrator only

**Request Body (any subset):**
```json
{
  "title": "Evaluasi Usability App Wisata Kalsel - Updated",
  "app_name": "WisataKu v2",
  "description": "Updated description",
  "status": "active"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | New title |
| `app_name` | string | New app name |
| `description` | string | New description |
| `status` | string | New status: `draft`, `active`, `closed` |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440000",
    "administrator_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "title": "Evaluasi Usability App Wisata Kalsel - Updated",
    "app_name": "WisataKu v2",
    "description": "Updated description",
    "itauq_version": "itauq-v1",
    "status": "active",
    "created_at": "2026-08-21T10:30:00Z",
    "updated_at": "2026-08-21T12:00:00Z"
  }
}
```

**Errors:**
- `401 UNAUTHORIZED` - User identity not found
- `403 FORBIDDEN` - Not authorized to update this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

#### `DELETE /questionnaires/:id`

Delete a questionnaire and all related data (cascades to task scenarios).

**Auth:** Owning Administrator only

**Response:** `204 No Content`

**Errors:**
- `401 UNAUTHORIZED` - User identity not found
- `403 FORBIDDEN` - Not authorized to delete this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

### 3. Task Scenarios

Tasks that respondents must complete during evaluation. Nested under questionnaires.

#### `POST /questionnaires/:id/task-scenarios`

Create a new task scenario under a questionnaire.

**Auth:** Administrator (must own the questionnaire)

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `id` | uuid | Questionnaire ID |

**Request Body:**
```json
{
  "title": "Mencari destinasi wisata terdekat",
  "instruction": "Gunakan fitur pencarian untuk menemukan tempat wisata dalam radius 5km.",
  "task_order": 1
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Task title |
| `instruction` | string | Yes | Instructions for the respondent |
| `task_order` | int | No | Display order (default: 0) |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
    "title": "Mencari destinasi wisata terdekat",
    "instruction": "Gunakan fitur pencarian untuk menemukan tempat wisata dalam radius 5km.",
    "task_order": 1,
    "created_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Invalid input data
- `403 FORBIDDEN` - Not authorized to access this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

#### `GET /questionnaires/:id/task-scenarios`

List all task scenarios for a questionnaire.

**Auth:** Owning Administrator or Super Admin

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `id` | uuid | Questionnaire ID |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
      "title": "Mencari destinasi wisata terdekat",
      "instruction": "Gunakan fitur pencarian untuk menemukan tempat wisata dalam radius 5km.",
      "task_order": 1,
      "created_at": "2026-08-21T10:30:00Z"
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
      "title": "Melihat detail tempat wisata",
      "instruction": "Klik salah satu hasil pencarian dan lihat informasi lengkapnya.",
      "task_order": 2,
      "created_at": "2026-08-21T10:35:00Z"
    }
  ]
}
```

**Errors:**
- `403 FORBIDDEN` - Not authorized to access this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

#### `GET /task-scenarios/:id`

Get a single task scenario by ID.

**Auth:** Owning Administrator or Super Admin

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
    "title": "Mencari destinasi wisata terdekat",
    "instruction": "Gunakan fitur pencarian untuk menemukan tempat wisata dalam radius 5km.",
    "task_order": 1,
    "created_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `404 NOT_FOUND` - Task scenario not found

---

#### `PATCH /task-scenarios/:id`

Update a task scenario (partial update).

**Auth:** Owning Administrator (ownership resolved via parent questionnaire)

**Request Body (any subset):**
```json
{
  "title": "Mencari destinasi wisata terdekat (updated)",
  "instruction": "Gunakan fitur pencarian untuk menemukan tempat wisata dalam radius 10km.",
  "task_order": 2
}
```

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | New title |
| `instruction` | string | New instruction |
| `task_order` | int | New display order |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440000",
    "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
    "title": "Mencari destinasi wisata terdekat (updated)",
    "instruction": "Gunakan fitur pencarian untuk menemukan tempat wisata dalam radius 10km.",
    "task_order": 2,
    "created_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Invalid input data
- `404 NOT_FOUND` - Task scenario not found

---

#### `DELETE /task-scenarios/:id`

Delete a task scenario.

**Auth:** Owning Administrator

**Response:** `204 No Content`

**Errors:**
- `404 NOT_FOUND` - Task scenario not found

---

### 4. Evaluation Links

Public links that let respondents access a questionnaire without logging in. Nested under questionnaires for list/create; top-level for update/delete by link ID.

#### `POST /questionnaires/:id/evaluation-links`

Generate a new public token/link for a questionnaire.

**Auth:** Administrator (must own the questionnaire — Super Admin cannot create on behalf of another administrator)

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `id` | uuid | Questionnaire ID |

**Request Body (optional):**
```json
{
  "expires_at": "2026-12-31T23:59:59Z"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `expires_at` | timestamp | No | When the link stops working. Omit for a link that never expires |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "990e8400-e29b-41d4-a716-446655440000",
    "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
    "token": "8f3ka92j",
    "url": "https://itauq.site/e/8f3ka92j",
    "created_by": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "is_active": true,
    "expires_at": null,
    "created_at": "2026-08-22T09:15:00Z"
  }
}
```

The `url` base comes from the `PUBLIC_EVALUATION_URL_BASE` environment variable (default: `https://itauq.site/e/`). Tokens are 8-character random strings from `a-z0-9`.

**Errors:**
- `400 VALIDATION_ERROR` - `expires_at` is in the past or malformed body
- `403 FORBIDDEN` - Not authorized to access this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

#### `GET /questionnaires/:id/evaluation-links`

List all evaluation links for a questionnaire, newest first.

**Auth:** Owning Administrator or Super Admin

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `id` | uuid | Questionnaire ID |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "990e8400-e29b-41d4-a716-446655440000",
      "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
      "token": "8f3ka92j",
      "url": "https://itauq.site/e/8f3ka92j",
      "created_by": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
      "is_active": true,
      "expires_at": null,
      "created_at": "2026-08-22T09:15:00Z"
    },
    {
      "id": "aa0e8400-e29b-41d4-a716-446655440000",
      "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
      "token": "k29djf81",
      "url": "https://itauq.site/e/k29djf81",
      "created_by": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
      "is_active": false,
      "expires_at": "2026-12-31T23:59:59Z",
      "created_at": "2026-08-21T14:00:00Z"
    }
  ]
}
```

**Errors:**
- `403 FORBIDDEN` - Not authorized to access this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

#### `PATCH /evaluation-links/:id`

Deactivate/reactivate a link or change its expiry.

**Auth:** Owning Administrator (ownership resolved via parent questionnaire)

**Request Body (any subset):**
```json
{
  "is_active": false,
  "expires_at": "2027-06-30T23:59:59Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `is_active` | bool | Set to `false` to deactivate the link |
| `expires_at` | timestamp | New expiry time (must be in the future) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "990e8400-e29b-41d4-a716-446655440000",
    "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
    "token": "8f3ka92j",
    "url": "https://itauq.site/e/8f3ka92j",
    "created_by": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "is_active": false,
    "expires_at": "2027-06-30T23:59:59Z",
    "created_at": "2026-08-22T09:15:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - `expires_at` is in the past or malformed body
- `403 FORBIDDEN` - Not the owning administrator
- `404 NOT_FOUND` - Evaluation link not found

---

#### `DELETE /evaluation-links/:id`

Delete an evaluation link.

**Auth:** Owning Administrator

**Response:** `204 No Content`

**Errors:**
- `403 FORBIDDEN` - Not the owning administrator
- `404 NOT_FOUND` - Evaluation link not found

---

### 5. Public Respondent Flow

Anonymous, token-gated flow used by respondents filling out an evaluation. No JWT, no login — each request is identified by the short `token` embedded in the evaluation URL, and after starting a session by the `respondent_id` returned in step 2. The backend uses the service-role DB connection (no public Supabase policies are defined for `respondents`/`answers`).

#### `GET /public/evaluation/:token`

Loads everything the respondent-facing app needs to render the flow.

**Auth:** none

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `token` | string | The evaluation link token |

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "questionnaire": {
      "title": "Evaluasi Usability App Wisata Kalsel",
      "app_name": "WisataKu"
    },
    "task_scenarios": [
      {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "title": "Mencari destinasi wisata terdekat",
        "instruction": "Gunakan fitur pencarian untuk menemukan tempat wisata dalam radius 5km.",
        "task_order": 1
      }
    ],
    "itauq": {
      "version": "itauq-v1",
      "scale_min": 1,
      "scale_max": 7,
      "questions": [
        {
          "id": 1,
          "category": "Attractiveness",
          "text": "Desain Aplikasi WisataKu tampak menarik secara visual.",
          "minLabel": "tidak menarik",
          "maxLabel": "sangat menarik"
        }
      ]
    }
  }
}
```

The 30 ITAUQ items come from the embedded `pkg/itauq/itauq.json` (compiled into the binary, no runtime file dependency). The `{AppName}` placeholder is substituted with `questionnaire.app_name` before the response is returned.

**Errors:**
- `404 NOT_FOUND` - Unknown token
- `409 LINK_INACTIVE` - Link is `is_active = false` or past its `expires_at`

---

#### `POST /public/evaluation/:token/respondents`

Starts a respondent session for this evaluation link.

**Auth:** none

**Request Body:**
```json
{
  "name": "Ahmad",
  "email": "ahmad@example.com",
  "age": 22,
  "gender": "male",
  "occupation": "Mahasiswa"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Respondent name |
| `email` | string | No | Email address |
| `age` | integer | No | Age (must be ≥ 0) |
| `gender` | string | No | `male`, `female`, or `other` |
| `occupation` | string | No | Job/role |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "respondent_id": "bb0e8400-e29b-41d4-a716-446655440000",
    "started_at": "2026-08-23T10:00:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - `name` empty, `age` negative, or `gender` not in the allowed set
- `404 NOT_FOUND` - Unknown token
- `409 LINK_INACTIVE` - Link is inactive or expired

---

#### `POST /public/evaluation/:token/respondents/:respondent_id/task-attempts`

Submits results for the task scenarios. A respondent may call this endpoint once with all attempts, or once per task. Existing attempts for the same `task_scenario_id` are not upserted; each call inserts new rows.

**Auth:** none (the `respondent_id` acts as the session key — it must belong to the evaluation link resolved from `:token`)

**Request Body:**
```json
{
  "attempts": [
    {
      "task_scenario_id": "770e8400-e29b-41d4-a716-446655440000",
      "is_success": true,
      "duration_seconds": 42,
      "notes": "Mudah dicari"
    }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `task_scenario_id` | uuid | Yes | Must belong to the questionnaire resolved from `:token` |
| `is_success` | bool | No | Whether the respondent completed the task |
| `duration_seconds` | int | No | Time taken (must be ≥ 0) |
| `notes` | string | No | Free-form notes |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "attempts": [
      {
        "id": "cc0e8400-e29b-41d4-a716-446655440000",
        "respondent_id": "bb0e8400-e29b-41d4-a716-446655440000",
        "task_scenario_id": "770e8400-e29b-41d4-a716-446655440000",
        "is_success": true,
        "duration_seconds": 42,
        "notes": "Mudah dicari",
        "created_at": "2026-08-23T10:05:00Z"
      }
    ]
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Empty `attempts`, or negative `duration_seconds`
- `404 NOT_FOUND` - Unknown token, or `respondent_id` does not belong to this evaluation link
- `409 LINK_INACTIVE` - Link is inactive or expired
- `422 INVALID_ATTEMPTS` - A `task_scenario_id` does not belong to this questionnaire

---

#### `POST /public/evaluation/:token/respondents/:respondent_id/answers`

Submits ITAUQ Likert answers. Typically all 30 are sent in a single call. Each `(item_id, category)` pair is validated server-side against the embedded `itauq-v1` instrument before insert; scores must be in `[1, 7]`. This mirrors the DB check constraints (`item_id 1–30`, `score 1–7`).

**Auth:** none

**Request Body:**
```json
{
  "answers": [
    { "item_id": 1, "category": "Attractiveness", "score": 6 },
    { "item_id": 2, "category": "Attractiveness", "score": 7 }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `item_id` | int | Yes | 1–30 (must be an instrument item) |
| `category` | string | Yes | Must match the category defined by the instrument for this `item_id` |
| `score` | int | Yes | 1–7 (Likert) |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "answers": [
      {
        "id": "dd0e8400-e29b-41d4-a716-446655440000",
        "respondent_id": "bb0e8400-e29b-41d4-a716-446655440000",
        "item_id": 1,
        "category": "Attractiveness",
        "score": 6,
        "created_at": "2026-08-23T10:10:00Z"
      }
    ]
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Empty `answers`
- `404 NOT_FOUND` - Unknown token, or `respondent_id` does not belong to this evaluation link
- `409 LINK_INACTIVE` - Link is inactive or expired
- `422 INVALID_ANSWERS` - Unknown `item_id`, mismatched `category` for that item, or `score` outside `[1, 7]`

---

#### `POST /public/evaluation/:token/respondents/:respondent_id/sus-answers`

Submits the System Usability Scale (SUS) answers — the respondent's rating of the **evaluation website itself** (not the app being evaluated). Filled once, all 10 items, right after the ITAUQ answers. Each `item_id`/`score` is validated server-side against the fixed `sus-v1` instrument (10 items, Likert 1–5); this mirrors the DB check constraints.

**Auth:** none

**Request Body:**
```json
{
  "answers": [
    { "item_id": 1, "score": 4 },
    { "item_id": 2, "score": 2 }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `item_id` | int | Yes | 1–10 (must be an instrument item) |
| `score` | int | Yes | 1–5 (Likert) |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "answers": [
      {
        "id": "ee0e8400-e29b-41d4-a716-446655440000",
        "respondent_id": "bb0e8400-e29b-41d4-a716-446655440000",
        "item_id": 1,
        "score": 4,
        "created_at": "2026-08-23T10:12:00Z"
      }
    ]
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Empty `answers`
- `404 NOT_FOUND` - Unknown token, or `respondent_id` does not belong to this evaluation link
- `409 LINK_INACTIVE` - Link is inactive or expired
- `422 INVALID_SUS_ANSWERS` - Unknown `item_id` or `score` outside `[1, 5]`

---

#### `POST /public/evaluation/:token/respondents/:respondent_id/submit`

Finalizes the session. The backend verifies that the respondent has submitted all 30 ITAUQ answers and one attempt per task scenario on the questionnaire before stamping `submitted_at`. Once accepted, the session cannot be reopened.

**Auth:** none

**Request Body:** none

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "submitted_at": "2026-08-23T10:15:00Z"
  }
}
```

**Errors:**
- `400 INCOMPLETE_SESSION` - Missing answers (ITAUQ or SUS) or task attempts (the message includes the expected vs. observed counts)
- `404 NOT_FOUND` - Unknown token, or `respondent_id` does not belong to this evaluation link
- `409 LINK_INACTIVE` - Link is inactive or expired

---

### 6. ITAUQ Instrument

The fixed 30-question instrument bundled with the backend (`pkg/itauq/itauq.json`, compiled into the binary). The same instrument is reused internally by `GET /public/evaluation/:token`, so changes here flow through to the respondent flow without redeploying any data.

#### `GET /instruments/itauq`

Returns the instrument definition so the Administrator-facing frontend can preview questions.

**Auth:** Administrator or Super Admin

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "version": "itauq-v1",
    "scale_min": 1,
    "scale_max": 7,
    "questions": [
      {
        "id": 1,
        "category": "Attractiveness",
        "text": "Desain Aplikasi {AppName} tampak menarik secara visual.",
        "minLabel": "tidak menarik",
        "maxLabel": "sangat menarik"
      }
    ]
  }
}
```

Notes:
- Question `text` contains the literal `{AppName}` placeholder — substitute with the relevant `questionnaire.app_name` at render time (the public endpoint does this server-side).
- Returns 30 questions covering the ITAUQ Likert items; no pagination.

---

#### `GET /instruments/sus`

Returns the fixed 10-item SUS instrument (System Usability Scale, applied to the evaluation website itself). Same source as the public endpoint — bundled as `pkg/sus/sus.json` and compiled into the binary.

**Auth:** Administrator or Super Admin

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "version": "sus-v1",
    "scale_min": 1,
    "scale_max": 5,
    "questions": [
      { "id": 1, "text": "Saya pikir saya akan sering menggunakan website ini.", "polarity": "positive" },
      { "id": 2, "text": "Saya merasa website ini rumit untuk digunakan, padahal seharusnya tidak perlu serumit itu.", "polarity": "negative" }
    ]
  }
}
```

Notes:
- `polarity` tells the scoring view which items contribute positively vs. negatively; item order is fixed 1–10 and matches the standard SUS.
- The respondent-facing flow exposes the same instrument under `data.sus` on `GET /public/evaluation/:token`.

---

## Database Schema

### Tables

#### `administrator_applications`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `full_name` | text | Applicant name |
| `email` | text | Applicant email |
| `institution` | text | Organization |
| `occupation` | text | Job title |
| `reason` | text | Application reason |
| `status` | enum | `pending`, `approved`, `rejected` |
| `reviewed_by` | uuid | FK to profiles |
| `reviewed_at` | timestamp | Review timestamp |
| `review_note` | text | Reviewer notes |
| `created_at` | timestamp | Creation time |

#### `profiles`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key (FK to auth.users) |
| `full_name` | text | User name |
| `occupation` | text | Job title |
| `institution` | text | Organization |
| `roles` | enum | `administrator`, `super_admin` |
| `application_id` | uuid | FK to administrator_applications |
| `created_at` | timestamp | Creation time |

#### `questionnaires`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `administrator_id` | uuid | FK to profiles |
| `title` | text | Questionnaire title |
| `app_name` | text | Application name |
| `description` | text | Description |
| `itauq_version` | text | ITAUQ version (default: `itauq-v1`) |
| `status` | enum | `draft`, `active`, `closed` |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

#### `task_scenarios`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `questionnaire_id` | uuid | FK to questionnaires |
| `title` | text | Task title |
| `instruction` | text | Task instructions |
| `task_order` | integer | Display order |
| `created_at` | timestamp | Creation time |

#### `evaluation_links`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `questionnaire_id` | uuid | FK to questionnaires |
| `token` | text | Unique link token |
| `created_by` | uuid | FK to profiles |
| `is_active` | boolean | Link active status |
| `expires_at` | timestamp | Expiration time |
| `created_at` | timestamp | Creation time |

#### `respondents`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `evaluation_link_id` | uuid | FK to evaluation_links |
| `name` | text | Respondent name |
| `email` | text | Respondent email |
| `age` | integer | Respondent age |
| `gender` | enum | `male`, `female` |
| `occupation` | text | Respondent occupation |
| `extra_data` | jsonb | Additional data |
| `started_at` | timestamp | Session start |
| `submitted_at` | timestamp | Submission time |
| `created_at` | timestamp | Creation time |

#### `task_scenario_attempts`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `respondent_id` | uuid | FK to respondents |
| `task_scenario_id` | uuid | FK to task_scenarios |
| `is_success` | boolean | Task completion status |
| `duration_seconds` | integer | Time taken |
| `notes` | text | Additional notes |
| `created_at` | timestamp | Creation time |

#### `questionnaire_answers`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `respondent_id` | uuid | FK to respondents |
| `item_id` | integer | Question number (1-30) |
| `category` | text | ITAUQ category |
| `score` | integer | Likert scale (1-7) |
| `created_at` | timestamp | Creation time |

#### `sus_answers`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `respondent_id` | uuid | FK to respondents |
| `item_id` | integer | SUS item number (1-10) |
| `score` | integer | Likert scale (1-5) |
| `created_at` | timestamp | Creation time |

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Request body failed validation |
| `INCOMPLETE_SESSION` | 400 | Respondent tried to submit without all answers/attempts |
| `UNAUTHORIZED` | 401 | Missing or expired token |
| `FORBIDDEN` | 403 | Authenticated but wrong role or not the resource owner |
| `NOT_FOUND` | 404 | Resource doesn't exist (also: unknown public evaluation token, unknown `respondent_id` for the resolved link) |
| `DUPLICATE_APPLICATION` | 409 | Pending application already exists for this email |
| `APPLICATION_ALREADY_REVIEWED` | 409 | Application is not in pending status |
| `LINK_INACTIVE` | 409 | Evaluation link is inactive or expired |
| `INVALID_ANSWERS` | 422 | One or more answers fail ITAUQ instrument validation (unknown `item_id`, mismatched `category`, or `score` outside `[1, 7]`) |
| `INVALID_SUS_ANSWERS` | 422 | One or more SUS answers fail instrument validation (unknown `item_id` or `score` outside `[1, 5]`) |
| `INVALID_ATTEMPTS` | 422 | One or more task attempts reference a `task_scenario_id` outside the questionnaire |
| `INTERNAL_ERROR` | 500 | Unhandled server error |

---

## Role Permission Matrix

| Endpoint | Super Admin | Administrator | Public |
|----------|:-----------:|:-------------:|:------:|
| `POST /applications` | - | - | ✅ |
| `GET /applications` | ✅ | - | - |
| `GET /applications/:id` | ✅ | - | - |
| `PATCH /applications/:id/approve` | ✅ | - | - |
| `PATCH /applications/:id/reject` | ✅ | - | - |
| `POST /questionnaires` | - | ✅ | - |
| `GET /questionnaires` | ✅ (all) | ✅ (own) | - |
| `GET /questionnaires/:id` | ✅ | ✅ (own) | - |
| `PATCH /questionnaires/:id` | - | ✅ (own) | - |
| `DELETE /questionnaires/:id` | - | ✅ (own) | - |
| `POST /questionnaires/:id/task-scenarios` | - | ✅ (own) | - |
| `GET /questionnaires/:id/task-scenarios` | ✅ | ✅ (own) | - |
| `GET /task-scenarios/:id` | ✅ | ✅ (own) | - |
| `PATCH /task-scenarios/:id` | - | ✅ (own) | - |
| `DELETE /task-scenarios/:id` | - | ✅ (own) | - |
| `POST /questionnaires/:id/evaluation-links` | - | ✅ (own) | - |
| `GET /questionnaires/:id/evaluation-links` | ✅ | ✅ (own) | - |
| `PATCH /evaluation-links/:id` | - | ✅ (own) | - |
| `DELETE /evaluation-links/:id` | - | ✅ (own) | - |
| `GET /public/evaluation/:token` | - | - | ✅ |
| `GET /instruments/itauq` | ✅ | ✅ | - |
| `GET /instruments/sus` | ✅ | ✅ | - |
| `POST /public/evaluation/:token/respondents` | - | - | ✅ |
| `POST /public/evaluation/:token/respondents/:respondent_id/task-attempts` | - | - | ✅ |
| `POST /public/evaluation/:token/respondents/:respondent_id/answers` | - | - | ✅ |
| `POST /public/evaluation/:token/respondents/:respondent_id/sus-answers` | - | - | ✅ |
| `POST /public/evaluation/:token/respondents/:respondent_id/submit` | - | - | ✅ |

---

## Development Setup

### Environment Variables

```bash
# Database (optional - falls back to in-memory if not set)
DATABASE_URL=postgresql://user:pass@host:5432/dbname

# Supabase (required for production auth)
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key

# Public evaluation link base URL (optional)
PUBLIC_EVALUATION_URL_BASE=https://itauq.site/e/

# Server
APP_PORT=8080
```

### Running the Server

```bash
# Development (in-memory storage)
go run cmd/api/main.go

# With database
DATABASE_URL="postgresql://..." go run cmd/api/main.go
```

### Testing with Postman

Import the provided Postman collections:
- `postman_questionnaires.json` - Questionnaire CRUD tests
- `postman_task_scenarios.json` - Task scenario CRUD tests
- `postman_evaluation_links.json` - Evaluation link CRUD tests
- `postman_public_flow.json` - Public respondent flow (anonymous, end-to-end evaluation session)

> The public-flow collection bootstraps its own questionnaire + task scenarios + evaluation link in the **Prerequisites** folder. Run it first to populate the variables (`questionnaire_id`, `task_scenario_id_1`, `task_scenario_id_2`, `evaluation_link_token`, `evaluation_link_id`, `respondent_id`), then the rest of the collection exercises the five public endpoints and their error cases.

Set collection variables:
- `base_url`: `http://localhost:8080`
- `admin_id`: Your administrator UUID
- `questionnaire_id`: Set automatically by test scripts
- `task_scenario_id`: Set automatically by test scripts
- `evaluation_link_id`: Set automatically by test scripts
- `respondent_id`: Set automatically by the public-flow collection after the `POST .../respondents` step

---

## Architecture

```
cmd/api/main.go                 # Entry point
internal/
  domain/                       # Business entities
    application/                # Application domain
    questionnaire/              # Questionnaire domain
    taskscenario/               # Task scenario domain
    evaluationlink/             # Evaluation link domain
  repository/                   # Data access layer
    application/                # In-memory + PostgreSQL
    questionnaire/              # In-memory + PostgreSQL
    taskscenario/               # In-memory + PostgreSQL
    evaluationlink/             # In-memory + PostgreSQL
  usecase/                      # Business logic
    application/                # Application use cases
    questionnaire/              # Questionnaire use cases
    taskscenario/               # Task scenario use cases
    evaluationlink/             # Evaluation link use cases
  delivery/http/                # HTTP layer
    handler/                    # Request handlers
    middleware/                  # Auth middleware
    route/                      # Route definitions
  infrastructure/               # External services
    config/                     # Configuration
    database/                   # Database connection
    supabase/                   # Supabase integration
pkg/                            # Shared utilities
  errors/                       # Error types
  response/                     # Response helpers
  utils/                        # Utility functions
  validator/                    # Input validation
```

---

## Planned Endpoints (Not Yet Implemented)

The following endpoints are specified in `API_SPECIFICATION.md` but not yet implemented:

- **Profiles:** `GET/PATCH /profiles/me`
- **Administrators:** `GET/POST/PATCH/DELETE /administrators`
- **ITAUQ Instrument:** ~~`GET /instruments/itauq`~~ (implemented, see [§6](#6-itauq-instrument))
- **Evaluation Results:** `GET /respondents`, `GET /questionnaires/{id}/report`
