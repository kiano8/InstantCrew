# InstantCrew

InstantCrew is a web app for connecting employers who need short-term or shift-based workers with applicants who are available to work. It includes employer and applicant portals, job matching and booking workflows, payment proof review, and administrative tools. The frontend is plain HTML, CSS, and JavaScript; the main application server and API are written in Go and use SQLite.

## Contents

- [Features](#features)
- [Technology](#technology)
- [Requirements](#requirements)
- [Run locally](#run-locally)
- [Configuration](#configuration)
- [User roles and workflows](#user-roles-and-workflows)
- [Project layout](#project-layout)
- [API overview](#api-overview)
- [Database and uploaded files](#database-and-uploaded-files)
- [Google sign-in](#google-sign-in)
- [Optional GCash verifier](#optional-gcash-verifier)
- [Tests](#tests)
- [Deployment notes](#deployment-notes)

## Features

- Employer and applicant account registration and login, with role-based access.
- Optional Google OAuth sign-in.
- Employer job and shift creation, editing, matching, booking confirmation, cancellation, and completion.
- Applicant profiles, role and availability settings, match responses, assignment tracking, and ratings.
- Search-fee and hiring-pass payment records, payment history, proof uploads, and admin verification.
- Persistent notifications and session idle timeout handling.
- Admin and accounting views for operational and payment review.
- Job role catalog synchronized into the database at server startup.

## Technology

- Go HTTP server (Go 1.26 or newer, as specified by `go.mod`)
- SQLite via `modernc.org/sqlite`; migrations are embedded in the application
- HTML, CSS, and browser JavaScript; no frontend build step is required
- Optional standalone Go service for GCash SMS payment verification

## Requirements

- Go 1.26 or newer
- A modern browser
- Git (to clone the repository)

The main app uses Go modules and downloads dependencies on its first build/run. You do not need Node.js to run the Go application. `serve.js` is a minimal static-file server for frontend-only preview; it does not provide the API, authentication, or database-backed features.

## Run locally

From the repository root:

```sh
go run ./cmd/server
```

Open [http://localhost:3000](http://localhost:3000). On first startup the app creates `data/instantcrew.db`, applies database migrations, syncs the role catalog, and creates the `uploads/` directory.

In development (`ENV=dev` by default), the server seeds an admin account when `ADMIN_PASSWORD` is unset:

```text
Email:    admin@instantcrew.com
Password: admin123
```

Set `ADMIN_PASSWORD` to choose a different password. Do not use the development default outside a local environment.

To build a server binary:

```sh
go build -o bin/instantcrew ./cmd/server
./bin/instantcrew
```

The server expects to run with the repository's public HTML, `css/`, `js/`, and `images/` assets available in its working directory. Configure `DB_PATH` and `GOOGLE_OAUTH_CREDENTIALS` as needed when launching from another directory.

## Configuration

The main server reads these environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3000` | HTTP listen port. |
| `ENV` | `dev` | Runtime environment label. `dev` enables the fallback development admin credentials. |
| `DB_PATH` | `data/instantcrew.db` | Main SQLite database path. Parent directory is created if needed. |
| `SESSION_IDLE_MINUTES` | `15` | Idle timeout before a session expires. |
| `BETA_MAX_USERS` | `300` | Maximum user count enforced by signup. |
| `COOKIE_SECURE` | `false` | Set to `true` when HTTPS is served directly or by a trusted TLS proxy. |
| `ADMIN_EMAIL` | `admin@instantcrew.com` | Email for the seeded admin account. |
| `ADMIN_PASSWORD` | Dev fallback only | Password used to create/update the admin account at startup. Set explicitly outside development. |
| `ACCOUNTING_EMAIL` | `accounting@instantcrew.com` | Email for the optional accounting account. |
| `ACCOUNTING_PASSWORD` | Unset | If set, creates/updates the accounting account at startup. |
| `GOOGLE_CLIENT_ID` | Unset | Google OAuth client ID. Set together with `GOOGLE_CLIENT_SECRET`. |
| `GOOGLE_CLIENT_SECRET` | Unset | Google OAuth client secret. |
| `GOOGLE_REDIRECT_URL` | Inferred from credentials/config where possible | OAuth callback URL registered with Google, typically `https://your-host/api/auth/google/callback`. |
| `GOOGLE_OAUTH_CREDENTIALS` | Searches for a client secret JSON file | Path to Google OAuth client credentials JSON when using file-based configuration. |

When Google OAuth settings are missing or invalid, Google sign-in is disabled and the server logs the reason; email/password signup and login remain available. For production, use HTTPS, set `COOKIE_SECURE=true`, provide strong admin credentials, and keep OAuth credentials and database files out of version control.

Example local configuration:

```sh
ENV=dev PORT=3000 DB_PATH=data/instantcrew.db go run ./cmd/server
```

## User roles and workflows

### Employer

Employers create an account, post a shift with a role, schedule, location, and crew requirements, then start a crew search. A search fee is recorded with the selected payment details or uploaded proof; the employer can review matches, confirm an applicant, and manage the booking through completion or cancellation. Employers can review job and payment history and receive notifications.

### Applicant

Applicants create an account, fill out their profile, select work roles, and set availability. They can review and accept or reject opportunities, track confirmed assignments, update their status while travelling, and end an assignment. Applicant ratings and hiring-pass/payment history are available in the applicant portal.

### Admin and accounting

Administrators review submitted payment records and uploaded proof, verify payments, and access administrative operations. An optional accounting account can be created by setting `ACCOUNTING_PASSWORD`; it uses the accounting view at `/accounting` (the same HTML shell as the admin page, with access governed by the account role).

## Project layout

```text
.
├── cmd/server/              Main Go HTTP server entry point
├── internal/auth/           Signup, login, sessions, Google OAuth, middleware
├── internal/jobs/           Job, matching, applicant, notification, and payment handlers
├── internal/store/          SQLite persistence and embedded schema migrations
├── css/                     Stylesheets
├── js/                      Frontend behavior and API calls
├── images/                  App artwork and payment assets
├── uploads/                 Runtime payment-proof uploads
├── gcash-verifier/          Optional standalone SMS payment verification service
├── index.html               Landing page
├── employer.html            Employer portal
├── applicant.html           Applicant portal
├── admin.html               Admin/accounting portal shell
└── login.html               Login page
```

The Go server only exposes an allowlist of HTML pages and the `css/`, `js/`, `images/`, and `uploads/` directories. Go source and the SQLite database are not served as static files.

## API overview

All main-app API endpoints are served by the same origin as the website. Authenticated routes use the session cookie; the browser sends it with same-origin requests.

| Area | Representative endpoints |
|---|---|
| Authentication | `POST /api/auth/signup`, `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/me`, `GET /api/auth/google` |
| Catalog | `GET /api/catalog` |
| Employer jobs | `POST /api/jobs`, `GET /api/jobs`, `GET /api/jobs/{id}`, `PUT /api/jobs/{id}`, `POST /api/jobs/{id}/search`, `POST /api/jobs/{id}/confirm` |
| Applicant | `/api/applicant/profile`, `/api/applicant/jobs`, `/api/applicant/assignments`, `/api/applicant/availability` |
| Notifications | `GET /api/notifications`, `POST /api/notifications/read`, `DELETE /api/notifications` |
| Payments | `GET /api/employer/payments`, `GET /api/applicant/payments`, `GET /api/admin/payments`, `POST /api/admin/payments/verify` |
| Uploads | `POST /api/upload-proof` |

For the full request and response behavior, see route registration and handler documentation in `internal/jobs/handler.go` and `internal/auth/handler.go`.

## Database and uploaded files

The main database defaults to `data/instantcrew.db`. Schema migrations are stored in `internal/store/migrations/` and are embedded and applied by the store when it opens the database. Back up the SQLite database and the `uploads/` directory together; uploaded proof images are referenced by records in the database.

The `uploads/` directory contains runtime user-submitted files and should be treated as private application data. Avoid committing real user uploads or database contents.

## Google sign-in

Configure a Google OAuth web client and register the callback URL for your deployment:

```text
https://your-host.example/api/auth/google/callback
```

Provide `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and (when needed) `GOOGLE_REDIRECT_URL`, or point `GOOGLE_OAUTH_CREDENTIALS` at the downloaded client secret JSON. Google sign-in also supports the app's role selection flow. If configuration fails, inspect the server startup log; the rest of the app can still run with password authentication.

## Optional GCash verifier

`gcash-verifier/` is a separate Go service for matching incoming GCash SMS notifications to pending exact-amount payments. It has its own SQLite database and test page, and is not automatically connected to the main app's payment records or employer/applicant UI. The main app currently records payment details and proof for administrative verification; do not assume running the verifier will settle a main-app payment.

See [`gcash-verifier/README.md`](gcash-verifier/README.md) for provider setup, SMS forwarding options, environment variables, callback signing, and the verifier API. To run its development test page from the project root:

```sh
GCASH_NUMBER=09XXXXXXXXX go run ./gcash-verifier/cmd/verifier
```

Then visit [http://localhost:4000](http://localhost:4000). The verifier defaults to development mode and logs warnings when webhook signing and sender allowlisting are not configured. Configure secrets and access controls before exposing it publicly.

## Tests

Run the main application test suite:

```sh
go test ./...
```

Run the optional verifier tests:

```sh
go test ./gcash-verifier/...
```

## Deployment notes

- Run the Go server behind HTTPS and set `COOKIE_SECURE=true`.
- Set `ENV` to the deployment environment and explicitly configure strong `ADMIN_PASSWORD` and any accounting credentials.
- Persist `DB_PATH` and `uploads/` across restarts; include both in backups.
- Keep OAuth secrets, signing keys, user data, and payment evidence out of source control and logs.
- Restrict access to administrative routes through the app's roles and protect the host and its backups.
- The standalone verifier is a separate service. If using it with real SMS forwarding, set a strong `SIGNING_KEY`, configure a sender allowlist, and expose only the HTTPS webhook URL required by the SMS provider.

