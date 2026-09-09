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
  - [Profile & First-login Password Change](#2-profile--first-login-password-change)
  - [Administrators (Account Management)](#3-administrators-account-management)
  - [Questionnaires](#4-questionnaires)
  - [Task Scenarios](#5-task-scenarios)
  - [Eligibility Criteria (Terms & Conditions Checklist)](#5b-eligibility-criteria-terms--conditions-checklist)
  - [Evaluation Links](#6-evaluation-links)
  - [Public Respondent Flow](#7-public-respondent-flow)
  - [ITAUQ Instrument](#8-itauq-instrument)
  - [Evaluation Results & Reports](#9-evaluation-results--reports)
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
    "application": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "full_name": "Siti Aminah",
      "email": "siti@example.com",
      "institution": "Universitas ABC",
      "occupation": "Dosen",
      "reason": "Ingin melakukan evaluasi usability aplikasi",
      "status": "approved",
      "review_note": "Diverifikasi, sesuai dengan data kampus.",
      "reviewed_by": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
      "reviewed_at": "2026-08-21T11:00:00Z",
      "created_at": "2026-08-21T10:30:00Z"
    },
    "administrator": {
      "id": "bbbbbbbb-cccc-dddd-eeee-ffffffffffff",
      "email": "siti@example.com",
      "role": "administrator"
    }
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
    "full_name": "Siti Aminah",
    "email": "siti@example.com",
    "institution": "Universitas ABC",
    "occupation": "Dosen",
    "reason": "Ingin melakukan evaluasi usability aplikasi",
    "status": "rejected",
    "review_note": "Data institusi tidak dapat diverifikasi.",
    "reviewed_by": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "reviewed_at": "2026-08-21T11:00:00Z",
    "created_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `404 NOT_FOUND` - Application not found
- `409 APPLICATION_ALREADY_REVIEWED` - Application is not in `pending` status

---

### 2. Profile & First-login Password Change

Self-service profile read/update and the forced-password-change flow used the first time a newly provisioned administrator logs in.

#### `GET /profiles/me`

Returns the caller's own profile row. The frontend should call this immediately after sign-in; if `must_change_password` is `true`, the user must be sent to the change-password screen and prevented from navigating until `POST /auth/change-password` succeeds.

**Auth:** Administrator or Super Admin (Bearer JWT)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "Siti Aminah",
    "occupation": "Dosen",
    "institution": "Universitas ABC",
    "role": "administrator",
    "must_change_password": true,
    "created_at": "2026-08-21T10:30:00Z"
  }
}
```

**Errors:**
- `401 UNAUTHORIZED` - Missing/invalid JWT
- `404 NOT_FOUND` - Profile row missing for the JWT subject

---

#### `PATCH /profiles/me`

Edit the caller's own profile. Only `full_name` is editable here; `role` is Super-Admin-controlled via `/administrators` and email is tied to the Supabase Auth identity.

**Auth:** Administrator or Super Admin (Bearer JWT)

**Request Body:**
```json
{ "full_name": "Siti Aminah Putri" }
```

**Response (200 OK):** the updated profile (same shape as `GET /profiles/me`).

**Errors:**
- `400 VALIDATION_ERROR` - `full_name` is blank or any other field was supplied

---

#### `POST /auth/change-password`

Rotates the caller's password via Supabase's admin API (service role) and clears `must_change_password` so the frontend can release navigation. `current_password` is informational — the backend does not verify it because identity is established by the JWT.

**Auth:** Administrator or Super Admin (Bearer JWT)

**Request Body:**
```json
{
  "current_password": "12345678",
  "new_password": "newSecret!9"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `current_password` | string | No | Previous password (informational, not validated). |
| `new_password` | string | Yes | New password, minimum 8 characters. |

**Response (200 OK):**
```json
{ "success": true, "data": { "message": "password updated" } }
```

**Errors:**
- `400 VALIDATION_ERROR` - `new_password` shorter than 8 characters
- `400 PASSWORD_UPDATE_FAILED` - Supabase admin update rejected (e.g. weak password)

---

### 3. Administrators (Account Management)

Super Admin manages administrator accounts (`profiles` rows where `role = administrator`). The `/profiles/me` endpoints above are for an administrator editing their *own* profile; this section is for Super Admin acting on other people's accounts.

#### `GET /administrators`

List all administrator accounts.

**Auth:** Super Admin only

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `is_active` | bool | Filter by active state (`true` / `false`) |
| `page` | int | Page number (default: 1) |
| `page_size` | int | Items per page (default: 20) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
      "full_name": "Siti Aminah",
      "email": "siti@example.com",
      "institution": "Universitas ABC",
      "occupation": "Dosen",
      "role": "administrator",
      "is_active": true,
      "must_change_password": false,
      "application_id": "550e8400-e29b-41d4-a716-446655440000",
      "created_at": "2026-08-01T08:00:00Z"
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

---

#### `GET /administrators/:id`

Get a single administrator account by id.

**Auth:** Super Admin only

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "full_name": "Siti Aminah",
    "email": "siti@example.com",
    "role": "administrator",
    "is_active": true,
    "must_change_password": false,
    "created_at": "2026-08-01T08:00:00Z"
  }
}
```

**Errors:**
- `404 NOT_FOUND` - Administrator not found

---

#### `POST /administrators`

Direct account creation, bypassing the application flow. Useful for Super Admin onboarding staff directly. Sends an invite email via Supabase Auth (the new administrator then sets their password through Supabase's own flow).

**Auth:** Super Admin only

**Request Body:**
```json
{
  "full_name": "Rudi Hartono",
  "email": "rudi@example.com"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `full_name` | string | Yes | Full name |
| `email` | string | Yes | Valid email address |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "bbbbbbbb-cccc-dddd-eeee-ffffffffffff",
    "full_name": "Rudi Hartono",
    "email": "rudi@example.com",
    "role": "administrator",
    "is_active": true,
    "must_change_password": true,
    "application_id": null,
    "created_at": "2026-08-22T09:00:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Missing/invalid `full_name` or `email`
- `502 INVITE_FAILED` - Supabase admin invite failed (e.g. duplicate email)

---

#### `PATCH /administrators/:id`

Update an administrator's `full_name` or activate/deactivate the account. Activation is preferred over hard delete to preserve referential history of their questionnaires/respondents.

**Auth:** Super Admin only

**Request Body (any subset):**
```json
{
  "full_name": "Siti Aminah Putri",
  "is_active": false
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "full_name": "Siti Aminah Putri",
    "email": "siti@example.com",
    "is_active": false,
    "created_at": "2026-08-01T08:00:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Blank `full_name`
- `404 NOT_FOUND` - Administrator not found

---

#### `DELETE /administrators/:id`

Hard delete (cascades to that administrator's questionnaires, task scenarios, evaluation links, and respondents — see schema `ON DELETE CASCADE`). Prefer `PATCH { "is_active": false }` unless the account must be fully purged.

**Auth:** Super Admin only · **Response:** `204 No Content`

**Errors:**
- `404 NOT_FOUND` - Administrator not found

---

### 4. Questionnaires

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

### 5. Task Scenarios

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

### 5b. Eligibility Criteria (Terms & Conditions Checklist)

Administrator-defined statements (e.g. domicile, citizenship, age group) that a respondent must tick before being allowed to fill in their identity on `POST /public/evaluation/:token/respondents`. Unlike the fixed ITAUQ / SUS instruments, criteria are stored in the database and managed per questionnaire. See `questionnaire_eligibility_criteria` and `respondent_eligibility_confirmations` in the schema.

If a questionnaire has **zero** criteria, the public-flow gate is skipped entirely (backward-compatible with legacy links). If it has at least one, every active criterion must appear in the respondent's `checked_criteria_ids` — partial coverage is rejected with `422 ELIGIBILITY_NOT_CONFIRMED`.

Nested under questionnaires for list/create; top-level for get/update/delete by criterion ID.

---

#### `POST /questionnaires/:id/eligibility-criteria`

Create a new eligibility statement under a questionnaire.

**Auth:** Administrator (must own the questionnaire)

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `id` | uuid | Questionnaire ID |

**Request Body:**
```json
{
  "statement": "Saya berdomisili di Kota Banjarbaru",
  "criteria_order": 1
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `statement` | string | Yes | The statement the respondent must confirm |
| `criteria_order` | int | No | Display order (default: 0) |

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "ee0e8400-e29b-41d4-a716-446655440000",
    "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
    "statement": "Saya berdomisili di Kota Banjarbaru",
    "criteria_order": 1,
    "created_at": "2026-08-25T08:00:00Z"
  }
}
```

**Errors:**
- `400 VALIDATION_ERROR` - Blank `statement`
- `403 FORBIDDEN` - Not authorized to access this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

#### `GET /questionnaires/:id/eligibility-criteria`

List all eligibility criteria for a questionnaire, ordered by `criteria_order` then creation time.

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
      "id": "ee0e8400-e29b-41d4-a716-446655440000",
      "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
      "statement": "Saya berdomisili di Kota Banjarbaru",
      "criteria_order": 1,
      "created_at": "2026-08-25T08:00:00Z"
    },
    {
      "id": "ff0e8400-e29b-41d4-a716-446655440000",
      "questionnaire_id": "660e8400-e29b-41d4-a716-446655440000",
      "statement": "Saya berusia 18-25 tahun",
      "criteria_order": 2,
      "created_at": "2026-08-25T08:01:00Z"
    }
  ]
}
```

**Errors:**
- `403 FORBIDDEN` - Not authorized to access this questionnaire
- `404 NOT_FOUND` - Questionnaire not found

---

#### `GET /eligibility-criteria/:id`

Get a single eligibility criterion by ID.

**Auth:** Owning Administrator (via parent questionnaire) or Super Admin

**Response (200 OK):** the criterion object (same shape as the create response).

**Errors:**
- `404 NOT_FOUND` - Criterion not found

---

#### `PATCH /eligibility-criteria/:id`

Update an eligibility criterion (partial update). Ownership is resolved via the criterion's parent questionnaire.

**Auth:** Owning Administrator

**Request Body (any subset):**
```json
{
  "statement": "Saya berdomisili di Kota Banjarbaru atau Martapura",
  "criteria_order": 1
}
```

| Field | Type | Description |
|-------|------|-------------|
| `statement` | string | New statement |
| `criteria_order` | int | New display order |

**Response (200 OK):** the updated criterion (same shape as the create response).

**Errors:**
- `400 VALIDATION_ERROR` - Blank `statement` after trim
- `403 FORBIDDEN` - Not the owning administrator
- `404 NOT_FOUND` - Criterion not found

---

#### `DELETE /eligibility-criteria/:id`

Delete an eligibility criterion. Confirmation rows referencing the criterion are removed by `ON DELETE CASCADE` from the criterion.

**Auth:** Owning Administrator

**Response:** `204 No Content`

**Errors:**
- `403 FORBIDDEN` - Not the owning administrator
- `404 NOT_FOUND` - Criterion not found

---

### 6. Evaluation Links

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

### 7. Public Respondent Flow

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
    "eligibility_criteria": [
      {
        "id": "ee0e8400-e29b-41d4-a716-446655440000",
        "statement": "Saya berdomisili di Kota Banjarbaru",
        "criteria_order": 1
      },
      {
        "id": "ff0e8400-e29b-41d4-a716-446655440000",
        "statement": "Saya berusia 18-25 tahun",
        "criteria_order": 2
      }
    ],
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
    },
    "sus": {
      "version": "sus-v1",
      "scale_min": 1,
      "scale_max": 5,
      "questions": [
        { "id": 1, "text": "Saya pikir saya akan sering menggunakan website ini.", "polarity": "positive" },
        { "id": 2, "text": "Saya merasa website ini rumit untuk digunakan, padahal seharusnya tidak perlu serumit itu.", "polarity": "negative" }
      ]
    }
  }
}
```

The 30 ITAUQ items come from the embedded `pkg/itauq/itauq.json` (compiled into the binary, no runtime file dependency). The `{AppName}` placeholder is substituted with `questionnaire.app_name` before the response is returned.

`eligibility_criteria` is empty (`[]`) when the questionnaire has no active terms; the respondent UI should hide the checklist screen entirely in that case.

**Errors:**
- `404 NOT_FOUND` - Unknown token
- `409 LINK_INACTIVE` - Link is `is_active = false` or past its `expires_at`

---

#### `POST /public/evaluation/:token/respondents`

Starts a respondent session for this evaluation link. If the questionnaire has any active eligibility criteria, the respondent must submit `checked_criteria_ids` covering every one of them — partial coverage is rejected before the respondent row is even created.

**Auth:** none

**Request Body:**
```json
{
  "name": "Ahmad",
  "email": "ahmad@example.com",
  "age": 22,
  "gender": "male",
  "occupation": "Mahasiswa",
  "checked_criteria_ids": [
    "ee0e8400-e29b-41d4-a716-446655440000",
    "ff0e8400-e29b-41d4-a716-446655440000"
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Respondent name |
| `email` | string | No | Email address |
| `age` | integer | No | Age (must be ≥ 0) |
| `gender` | string | No | `male`, `female`, `other`, or `prefer_not_to_say` |
| `occupation` | string | No | Job/role |
| `checked_criteria_ids` | uuid[] | Conditional | The IDs of the criteria the respondent has confirmed. Required only when the questionnaire has at least one active criterion; ignored otherwise. Each ID must be a criterion of the questionnaire, and the union must cover every active criterion (extra stale IDs are tolerated). |

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
- `422 ELIGIBILITY_NOT_CONFIRMED` - `checked_criteria_ids` does not cover every active criterion for the questionnaire. Nothing is written (no respondent row, no confirmations).

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

### 8. ITAUQ Instrument

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

### 9. Evaluation Results & Reports

Read-only views over submitted evaluation sessions. Scores are computed server-side from the raw `questionnaire_answers`, `sus_answers`, and `task_scenario_attempts` rows — there is no separate "computed" table to keep in sync.

**ITAUQ scoring.** Each of the 30 ITAUQ items belongs to one of 10 categories (3 items per category). For one respondent:

1. Per-category raw score = mean of the 3 item scores in that category (the `Skor Variabel` formula).
2. Per-category normalized score = `(raw - scale_min) / (scale_max - scale_min) * 100` (so a 1–7 Likert maps to 0–100).
3. Overall usability score = the mean of the per-category normalized scores, reported on the 0–100 scale.

**SUS scoring.** The standard System Usability Scale formula: for each of the 10 items, positive-polarity items contribute `(score - 1)` and negative-polarity items contribute `(5 - score)`; the final SUS score is the sum of contributions multiplied by 2.5, on the 0–100 scale.

#### `GET /respondents`

List respondents with their computed scores. Scope is enforced server-side from the caller's role:

- **Administrator** → only respondents of their own questionnaires ("View Own Evaluation Result"). Any `administrator_id` query param is ignored.
- **Super Admin** → all respondents across all administrators ("View All Evaluation Result"); `administrator_id` and `questionnaire_id` filters apply.

**Auth:** Administrator or Super Admin (Bearer JWT)

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `questionnaire_id` | uuid | Optional. Restrict to one questionnaire. |
| `administrator_id` | uuid | Optional, Super Admin only. Restrict to one administrator. |
| `page` | int | Page number (default: 1) |
| `page_size` | int | Items per page (default: 20, max: 100) |

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "respondent_id": "bb0e8400-e29b-41d4-a716-446655440000",
      "respondent_name": "Ahmad",
      "questionnaire_title": "Evaluasi Usability App Wisata Kalsel",
      "app_name": "WisataKu",
      "administrator_name": "Siti Aminah",
      "overall_usability_score": 82.14,
      "task_success_rate_pct": 100.0,
      "evaluation_website_sus_score": 77.5,
      "submitted_at": "2026-08-23T10:15:00Z"
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

Score fields are nullable: an Administrator viewing a respondent who has not yet submitted will see `null` for `overall_usability_score`, `task_success_rate_pct`, and `evaluation_website_sus_score`. Super Admin sees the same behavior.

**Errors:**
- `401 UNAUTHORIZED` - Missing/invalid JWT
- `403 FORBIDDEN` - JWT does not resolve to an administrator or super_admin role

---

#### `GET /respondents/:id`

Full per-respondent detail: identity, per-category ITAUQ scores, per-task results, and the SUS score for the evaluation website.

**Auth:** Owning Administrator (via the respondent's questionnaire) or Super Admin

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "respondent": {
      "id": "bb0e8400-e29b-41d4-a716-446655440000",
      "name": "Ahmad",
      "age": 22,
      "gender": "male",
      "occupation": "Mahasiswa"
    },
    "overall_usability_score": 82.14,
    "category_scores": [
      { "category": "Attractiveness", "avg_raw_score": 6.33, "normalized_score": 88.89 },
      { "category": "Efficiency", "avg_raw_score": 5.67, "normalized_score": 77.78 }
    ],
    "task_results": [
      {
        "task_scenario_id": "770e8400-e29b-41d4-a716-446655440000",
        "title": "Mencari destinasi wisata terdekat",
        "is_success": true,
        "duration_seconds": 42
      }
    ],
    "evaluation_website_sus": {
      "answered_items": 10,
      "sus_score": 77.5
    }
  }
}
```

`is_success` is `null` if the respondent skipped that scenario; `sus_score` is `null` if fewer than 10 SUS items were answered.

**Errors:**
- `401 UNAUTHORIZED` - Missing/invalid JWT
- `403 FORBIDDEN` - Caller is an administrator who does not own this respondent's questionnaire
- `404 NOT_FOUND` - Unknown respondent id

---

#### `GET /questionnaires/:id/report`

Aggregate report across **all** respondents of one questionnaire — the "Generate Evaluation Report" output.

**Auth:** Owning Administrator or Super Admin

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "questionnaire": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "title": "Evaluasi Usability App Wisata Kalser",
      "app_name": "WisataKu"
    },
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

`respondent_count` counts every respondent for the questionnaire (started and submitted). Average fields are nullable when no respondent has data for that metric yet. Each `category_averages[].normalized_score_avg` is the mean of the per-respondent normalized scores for that category.

**Errors:**
- `401 UNAUTHORIZED` - Missing/invalid JWT
- `403 FORBIDDEN` - Caller is an administrator who does not own this questionnaire
- `404 NOT_FOUND` - Unknown questionnaire id

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
| `must_change_password` | boolean | Forces first-login password rotation |
| `is_active` | boolean | Set to `false` to deactivate an administrator account |
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

#### `questionnaire_eligibility_criteria`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `questionnaire_id` | uuid | FK to questionnaires (CASCADE on delete) |
| `statement` | text | The eligibility statement the respondent must confirm |
| `criteria_order` | integer | Display order |
| `created_at` | timestamp | Creation time |

#### `respondent_eligibility_confirmations`
| Column | Type | Description |
|--------|------|-------------|
| `id` | uuid | Primary key |
| `respondent_id` | uuid | FK to respondents |
| `criteria_id` | uuid | FK to questionnaire_eligibility_criteria |
| `is_checked` | boolean | Whether the respondent ticked this criterion (default: true) |
| `created_at` | timestamp | Tick timestamp |

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
| `gender` | enum | `male`, `female`, `other`, `prefer_not_to_say` |
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
| `ELIGIBILITY_NOT_CONFIRMED` | 422 | `checked_criteria_ids` does not cover every active eligibility criterion for the questionnaire |
| `INVITE_FAILED` | 502 | Supabase admin invite to create an Auth user failed |
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
| `POST /questionnaires/:id/eligibility-criteria` | - | ✅ (own) | - |
| `GET /questionnaires/:id/eligibility-criteria` | ✅ | ✅ (own) | - |
| `GET /eligibility-criteria/:id` | ✅ | ✅ (own) | - |
| `PATCH /eligibility-criteria/:id` | - | ✅ (own) | - |
| `DELETE /eligibility-criteria/:id` | - | ✅ (own) | - |
| `POST /questionnaires/:id/evaluation-links` | - | ✅ (own) | - |
| `GET /questionnaires/:id/evaluation-links` | ✅ | ✅ (own) | - |
| `PATCH /evaluation-links/:id` | - | ✅ (own) | - |
| `DELETE /evaluation-links/:id` | - | ✅ (own) | - |
| `GET /public/evaluation/:token` | - | - | ✅ |
| `GET /instruments/itauq` | ✅ | ✅ | - |
| `GET /instruments/sus` | ✅ | ✅ | - |
| `GET /respondents` | ✅ (all) | ✅ (own) | - |
| `GET /respondents/:id` | ✅ | ✅ (own) | - |
| `GET /questionnaires/:id/report` | ✅ | ✅ (own) | - |
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
- `postman_eligibility_criteria.json` - Eligibility criterion CRUD tests (Terms & Conditions checklist)
- `postman_evaluation_links.json` - Evaluation link CRUD tests
- `postman_public_flow.json` - Public respondent flow (anonymous, end-to-end evaluation session)
- `postman_evaluation_results.json` - Evaluation Results & Reports (authenticated list / detail / report)

> The public-flow collection bootstraps its own questionnaire + task scenarios + eligibility criteria + evaluation link in the **Prerequisites** folder. Run it first to populate the variables (`questionnaire_id`, `task_scenario_id_1`, `task_scenario_id_2`, `eligibility_criterion_id_1`, `eligibility_criterion_id_2`, `evaluation_link_token`, `evaluation_link_id`, `respondent_id`), then the rest of the collection exercises the public endpoints and their error cases. The `POST .../respondents` step in the public flow now requires `checked_criteria_ids` covering the two prereq criteria.

The `postman_evaluation_results.json` collection reuses the same variables from `postman_public_flow.json` (it expects a `questionnaire_id` and `respondent_id` to already exist), then exercises `GET /respondents`, `GET /respondents/:id`, and `GET /questionnaires/:id/report`.

Set collection variables:
- `base_url`: `http://localhost:8080`
- `admin_id`: Your administrator UUID
- `questionnaire_id`: Set automatically by test scripts
- `task_scenario_id`: Set automatically by test scripts
- `eligibility_criterion_id`: Set automatically by test scripts
- `evaluation_link_id`: Set automatically by test scripts
- `respondent_id`: Set automatically by the public-flow collection after the `POST .../respondents` step

---

## Architecture

```
cmd/api/main.go                 # Entry point
internal/
  domain/                       # Business entities
    application/                # Application domain
    administrator/              # Administrator domain
    questionnaire/              # Questionnaire domain
    taskscenario/               # Task scenario domain
    eligibility/                # Eligibility criterion domain
    evaluationlink/             # Evaluation link domain
    evaluation/                 # Evaluation results domain
    profile/                    # Profile domain
    respondent/                 # Respondent domain
  repository/                   # Data access layer
    application/                # In-memory + PostgreSQL
    administrator/              # In-memory + PostgreSQL
    questionnaire/              # In-memory + PostgreSQL
    taskscenario/               # In-memory + PostgreSQL
    eligibility/                # In-memory + PostgreSQL
    evaluationlink/             # In-memory + PostgreSQL
    evaluation/                 # In-memory + PostgreSQL
    profile/                    # In-memory + PostgreSQL
    respondent/                 # In-memory + PostgreSQL
  usecase/                      # Business logic
    application/                # Application use cases
    administrator/              # Administrator use cases
    questionnaire/              # Questionnaire use cases
    taskscenario/               # Task scenario use cases
    eligibility/                # Eligibility criterion use cases
    evaluationlink/             # Evaluation link use cases
    evaluation/                 # Evaluation results use cases
    profile/                    # Profile use cases
    publicflow/                 # Public respondent flow use cases
  delivery/http/                # HTTP layer
    handler/                    # Request handlers
    route/                      # Route definitions
  infrastructure/               # External services
    config/                     # Configuration
    database/                   # Database connection
    supabase/                   # Supabase integration
pkg/                            # Shared utilities
  itauq/                        # ITAUQ instrument loader & scoring
  sus/                          # SUS instrument loader & scoring
```

---

## Planned Endpoints (Not Yet Implemented)

All endpoints in `API_SPECIFICATION.md` are now implemented.