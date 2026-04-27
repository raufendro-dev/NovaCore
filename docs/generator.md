# Generator

Command:

```bash
go run cmd/framework/main.go make:module User
go run cmd/framework/main.go make:model Product
go run cmd/framework/main.go make:controller Product
go run cmd/framework/main.go make:service Product
go run cmd/framework/main.go make:repository Product
go run cmd/framework/main.go make:endpoint Product
go run cmd/framework/main.go make:crud Product
go run cmd/framework/main.go make:migration create_products_table
go run cmd/framework/main.go make:seeder create_admin_user
```

`make:crud` membuat model, DTO, repository, service, handler, routes, migration, test, docs endpoint, dan Postman request.

CRUD generated otomatis protected dengan JWT middleware. Login terlebih dahulu lewat `/api/v1/auth/login`, lalu kirim header:

```http
Authorization: Bearer <access_token>
```

Untuk CRUD publik, tambahkan flag:

```bash
go run cmd/framework/main.go make:crud Article --public
```

Flag `--public` juga tersedia untuk `make:module` dan `make:endpoint`.
