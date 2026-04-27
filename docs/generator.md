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

Saat menjalankan `make:crud`, CLI akan menanyakan metode endpoint yang ingin dibuat, lalu field yang ingin disimpan. Field default `id`, `created_at`, `updated_at`, dan `deleted_at` sudah disediakan otomatis oleh `gorm.Model`. Tekan `Ctrl+D` pada prompt `Field name` untuk selesai.

Contoh:

```text
Methods [GET, POST, PUT, PATCH, DELETE]: POST, GET, DELETE
Field name: name
Type [string]: string
Required? [y/N]: y
Field name: price
Type [string]: float
Required? [y/N]: n
Field name: <Ctrl+D>
```

Tipe yang didukung: `string`, `text`, `int`, `uint`, `float`, `bool`, `time`.

Metode yang didukung: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`. `GET` menghasilkan route list dan detail.

Jika `Field name` kosong lalu ditekan `Enter`, CLI akan meminta input ulang dan menampilkan informasi bahwa `Ctrl+D` digunakan untuk menyelesaikan input.

CRUD generated otomatis protected dengan JWT middleware. Login terlebih dahulu lewat `/api/v1/auth/login`, lalu kirim header:

```http
Authorization: Bearer <access_token>
```

Untuk CRUD publik, tambahkan flag:

```bash
go run cmd/framework/main.go make:crud Article --public
```

Flag `--public` juga tersedia untuk `make:module` dan `make:endpoint`.
