# Auth Endpoints

Auth endpoint sudah built-in, jadi tidak perlu menjalankan generator khusus.

Base path: `/api/v1/auth`

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`

Gunakan `access_token` dari login atau register untuk endpoint yang membutuhkan auth:

```http
Authorization: Bearer <access_token>
```
