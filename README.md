<div align="center">

# InstantCrew

### Find the right crew for every shift.

Connect employers with people ready to work, from job posting and matching through booking and payment review.

![Go](https://img.shields.io/badge/Go-backend-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![HTML5](https://img.shields.io/badge/HTML5-frontend-E34F26?style=for-the-badge&logo=html5&logoColor=white)
![CSS3](https://img.shields.io/badge/CSS3-styling-1572B6?style=for-the-badge&logo=css3&logoColor=white)
![JavaScript](https://img.shields.io/badge/JavaScript-vanilla-F7DF1E?style=for-the-badge&logo=javascript&logoColor=111111)
![SQLite](https://img.shields.io/badge/SQLite-database-003B57?style=for-the-badge&logo=sqlite&logoColor=white)

</div>

InstantCrew helps employers find short-term and shift-based workers. Employers can post shifts and review applicants, while workers can set their profiles and availability, respond to opportunities, and track assignments.

## Features

### Employer portal

- Create and manage job and shift listings.
- Search for available workers and review matches.
- Confirm bookings and track job progress.
- View notifications and payment history.

### Applicant portal

- Set up a profile with work roles and availability.
- Review opportunities and accept or decline matches.
- Track confirmed assignments and update work status.
- View ratings and payment history.

### Admin tools

- Review payment submissions and uploaded proof.
- Manage users, jobs, and payment verification.

## Technology

The app uses a Go API, SQLite database, and a browser frontend built with HTML, CSS, and JavaScript. The frontend has no separate build step.

## API overview

Representative API routes include:

| Area | Endpoints |
|---|---|
| Authentication | `POST /api/auth/signup`, `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/me` |
| Job catalog | `GET /api/catalog` |
| Employer jobs | `POST /api/jobs`, `GET /api/jobs`, `GET /api/jobs/{id}`, `POST /api/jobs/{id}/search`, `POST /api/jobs/{id}/confirm` |
| Applicant | `/api/applicant/profile`, `/api/applicant/jobs`, `/api/applicant/assignments`, `/api/applicant/availability` |
| Notifications | `GET /api/notifications`, `POST /api/notifications/read` |
| Payments | `/api/employer/payments`, `/api/applicant/payments`, `/api/admin/payments` |

## Project structure

```text
cmd/server/       Go application entry point
internal/         Authentication, job workflows, and database code
css/              Frontend styles
js/               Browser scripts and API calls
images/           App images and artwork
index.html        Landing page
employer.html     Employer portal
applicant.html    Applicant portal
admin.html        Admin portal
login.html        Sign-in page
```
