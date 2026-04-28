# Authentication

Auth bawaan menyediakan register, login, refresh token, logout, current user, validasi token, bcrypt, dan role claim.

User auth memiliki field `role`. Saat register, `role` boleh dikirim sebagai parameter optional:

```json
{
  "name": "Demo User",
  "email": "demo@example.com",
  "password": "password123",
  "role": "user"
}
```

Jika `role` tidak dikirim, NovaCore otomatis memakai `user`. Role disimpan di tabel `users`, ikut masuk ke JWT claims, dan digunakan oleh middleware RBAC.

Login menghasilkan:

- `access_token`
- `refresh_token`

Gunakan access token sebagai header:

```http
Authorization: Bearer <token>
```

Endpoint protected bisa memakai middleware `middleware.Auth(jwt)` dan RBAC dengan `middleware.Role("admin")`.

CRUD yang dibuat oleh generator NovaCore memakai JWT auth secara default:

```go
group := router.Group("/products", middleware.Auth(jwt))
```

Jika endpoint perlu role tertentu, tambahkan role middleware:

```go
group := router.Group("/products", middleware.Auth(jwt), middleware.Role("admin"))
```

Untuk generated CRUD yang memang harus publik, gunakan:

```bash
novacore make:crud Article --public
```
