# Routing

Route utama berada di `/api/v1`.

Convention CRUD:

- `GET /api/v1/resources`
- `GET /api/v1/resources/:id`
- `POST /api/v1/resources`
- `PUT /api/v1/resources/:id`
- `PATCH /api/v1/resources/:id`
- `DELETE /api/v1/resources/:id`

Route generated diregistrasikan lewat `internal/routes/generated.go`.
