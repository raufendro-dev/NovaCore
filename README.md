# NovaCore

NovaCore adalah framework backend Golang modular untuk membangun REST API production-ready dengan cepat. Framework ini sudah membawa JWT authentication, refresh token, RBAC middleware, adapter database, generator CRUD, migration runner, dokumentasi, dan Postman Collection otomatis.

NovaCore dirancang supaya mudah dipahami developer junior, tetapi tetap rapi untuk kebutuhan tim backend yang lebih senior.

## Fitur Utama

- Struktur project backend Go yang bersih dan scalable
- REST API dengan response format konsisten
- JWT authentication dan refresh token
- Password hashing dengan bcrypt
- Role-based access control
- Middleware logger, recovery, CORS, request ID, rate limiter, timeout, secure headers
- Database support: SQLite, MySQL, PostgreSQL, MongoDB
- Redis optional untuk cache/session
- CLI generator untuk module, model, service, repository, controller, endpoint, CRUD, dan migration
- CRUD generated protected by JWT secara default
- Opsi CRUD public dengan flag `--public`
- Auto-update Postman Collection dan Environment
- Dokumentasi lengkap di folder `docs/`
- Testing setup untuk service dan generated CRUD

## Tech Stack

NovaCore memakai stack yang stabil dan umum dipakai di ekosistem Go:

- Gin untuk HTTP router
- GORM untuk SQL database
- MongoDB official driver untuk document database
- Redis Go client untuk Redis
- Viper untuk config loader
- Cobra untuk CLI generator
- Zap untuk structured logging
- bcrypt untuk password hashing
- `golang-jwt/jwt` untuk JWT
- `validator/v10` untuk request validation

## Quick Start

```bash
cp .env.example .env
go mod tidy
go install ./cmd/novacore
novacore run
```

Server berjalan di:

```text
http://localhost:8080
```

Health check:

```http
GET /api/v1/health
```

Jika `$GOPATH/bin` belum ada di `PATH`, kamu juga bisa build lokal:

```bash
go build -o bin/novacore cmd/novacore/main.go
./bin/novacore run
```

## Konfigurasi

Semua konfigurasi utama ada di `.env`.

Contoh default lokal memakai SQLite:

```env
APP_NAME=NovaCore
APP_ENV=development
APP_PORT=8080
APP_DEBUG=true

DB_DRIVER=sqlite
DB_SQLITE_PATH=database/app.db
DB_AUTO_CREATE=false

JWT_SECRET=change_this_secret
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
```

Jangan gunakan `JWT_SECRET` default untuk production.

## Auth

Auth endpoint sudah built-in, jadi tidak perlu digenerate.

Untuk database SQL (`sqlite`, `mysql`, `postgres`), tabel `users` auth otomatis dibuat saat server start:

```bash
novacore run
```

NovaCore menjalankan GORM `AutoMigrate` untuk model auth bawaan. Jadi setelah clone, `cp .env.example .env`, dan start server, tabel user auth sudah siap dipakai. Untuk production schema aplikasi, tetap disarankan memakai migration agar perubahan database tercatat.

| Method | Endpoint | Keterangan |
| --- | --- | --- |
| POST | `/api/v1/auth/register` | Membuat user baru |
| POST | `/api/v1/auth/login` | Login dan mendapatkan token |
| POST | `/api/v1/auth/refresh` | Membuat access token baru |
| POST | `/api/v1/auth/logout` | Logout client-side |
| GET | `/api/v1/auth/me` | Mengambil user saat ini |

Contoh register:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Demo User","email":"demo@example.com","password":"password123"}'
```

Contoh login:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@example.com","password":"password123"}'
```

Gunakan `access_token` untuk endpoint protected:

```http
Authorization: Bearer <access_token>
```

## Generate CRUD

CRUD default protected dengan JWT:

```bash
novacore make:crud Product
```

Sebelum generate, CLI akan menanyakan metode endpoint yang ingin dibuat, lalu field yang ingin disimpan. Field default `id`, `created_at`, `updated_at`, dan `deleted_at` sudah otomatis tersedia dari `gorm.Model`, jadi tidak perlu dimasukkan. Tekan `Ctrl+D` pada prompt `Field name` untuk menyelesaikan pengisian field.

Contoh input:

```text
Methods [GET, POST, PUT, PATCH, DELETE]: POST, GET, DELETE
Field name: name
Type [string]: string
Required? [y/N]: y
Default value [none]:
Field name: price
Type [string]: float
Required? [y/N]: n
Default value [none]: 0
Field name: <Ctrl+D>
```

Tipe field yang didukung:

- `string`
- `text`
- `int`
- `uint`
- `float`
- `bool`
- `time`

Metode yang didukung:

- `GET`: membuat `GET /resources` dan `GET /resources/:id`
- `POST`: membuat `POST /resources`
- `PUT`: membuat `PUT /resources/:id`
- `PATCH`: membuat `PATCH /resources/:id`
- `DELETE`: membuat `DELETE /resources/:id`

Endpoint yang dibuat:

| Method | Endpoint |
| --- | --- |
| GET | `/api/v1/products` |
| GET | `/api/v1/products/:id` |
| POST | `/api/v1/products` |
| PUT | `/api/v1/products/:id` |
| PATCH | `/api/v1/products/:id` |
| DELETE | `/api/v1/products/:id` |

Generator membuat:

- Model
- DTO request validation
- Repository
- Service
- Handler/controller
- Routes
- Migration SQL
- Basic service test
- Dokumentasi endpoint
- Postman requests

Field yang kamu masukkan akan ikut dibuat di model, DTO validation, service assignment, migration SQL, docs endpoint, dan Postman body example.

Metode yang kamu pilih akan menentukan route, docs endpoint, dan Postman request yang dibuat.

Jika `Field name` kosong lalu ditekan `Enter`, CLI akan meminta input ulang. Gunakan `Ctrl+D` untuk selesai.

Setelah generate:

```bash
go test ./...
novacore run
```

## CRUD Public

Untuk resource yang memang boleh diakses publik, gunakan `--public`.

```bash
novacore make:crud Article --public
```

Route yang dibuat tidak memakai JWT middleware.

## Update CRUD

Jika setelah membuat CRUD ada field atau tipe data yang terlewat, gunakan:

```bash
novacore update:crud Product
```

Safe mode:

```bash
novacore update:crud Product --safe
```

atau:

```bash
novacore update:crud Product --mode=safe
```

Alias lain:

```bash
novacore make update-crud Product
```

Command ini akan menanyakan konfirmasi terlebih dahulu:

```text
Updating CRUD columns will create a reset migration.
Existing table data will be deleted and IDs will restart from 0 after the migration is run.
Continue? [y/N]:
```

Jawab `y` atau `Y` untuk lanjut. Jawab `n`, `N`, atau kosong untuk membatalkan.

Setelah dikonfirmasi, CLI akan menanyakan ulang metode endpoint dan field. NovaCore lalu memperbarui model, DTO validation, repository, service, handler, routes API, test, migration reset table, dokumentasi endpoint, dan Postman request.

Migration update yang dibuat bersifat destructive karena menjalankan `DROP TABLE` lalu membuat ulang table. Jalankan migration ini hanya jika kamu siap kehilangan data tabel tersebut.

Gunakan `--safe` agar migration memakai `ALTER TABLE` untuk field baru dan tidak menjalankan `DROP TABLE`. Jika field baru required, default value wajib diisi agar row lama tetap valid.

## Generate Relation

Relasi antar CRUD/model bisa dibuat dengan:

```bash
novacore make:relation Product Category --type=belongs-to --include
novacore make:relation Category Product --type=has-many --nested
novacore make:relation User Role --type=many-to-many
novacore make:relation User Profile --type=has-one
```

Mode interaktif:

```bash
novacore make:relation
```

CLI akan menanyakan source model, target model, tipe relasi, foreign key, nested endpoint, include query, Postman, dan docs.

Tipe relasi:

- `belongs-to`
- `has-one`
- `has-many`
- `many-to-many`

Include query:

```http
GET /api/v1/products?include=category
GET /api/v1/orders?include=user,items
```

NovaCore memvalidasi include agar hanya relasi yang terdaftar yang diterima.

## Command Generator

```bash
novacore make:module User
novacore make:model Product
novacore make:controller Product
novacore make:service Product
novacore make:repository Product
novacore make:endpoint Product
novacore make:crud Product
novacore update:crud Product
novacore make:relation Product Category --type=belongs-to
novacore make:migration create_products_table
```

Flag `--public` tersedia untuk:

- `make:crud`
- `make:module`
- `make:endpoint`

## Database

Pilih database lewat `DB_DRIVER`.

SQLite:

```env
DB_DRIVER=sqlite
DB_SQLITE_PATH=database/app.db
```

MySQL:

```env
DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_NAME=app_db
DB_USER=root
DB_PASSWORD=password
DB_AUTO_CREATE=true
```

PostgreSQL:

```env
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=app_db
DB_USER=postgres
DB_PASSWORD=password
DB_SSL_MODE=disable
DB_AUTO_CREATE=true
```

MongoDB:

```env
DB_DRIVER=mongodb
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=app_db
```

Redis optional:

```env
REDIS_ENABLED=true
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

Jika muncul error seperti `Unknown database 'app_db'`, artinya MySQL/PostgreSQL server bisa diakses tetapi database `app_db` belum dibuat. Pilih salah satu:

```sql
CREATE DATABASE app_db;
```

atau aktifkan auto-create untuk development:

```env
DB_AUTO_CREATE=true
```

Untuk production, lebih aman buat database lewat provisioning/deployment pipeline dan biarkan `DB_AUTO_CREATE=false`.

## Migration dan Seeder

Migration dipakai untuk mengubah struktur database secara terkontrol, misalnya membuat tabel, menambah kolom, membuat index, atau mengubah tipe data. File migration disimpan di `migrations/`.

Seeder dipakai untuk mengisi data awal, misalnya admin pertama, role default, permission default, atau data referensi. File seeder disimpan di `seeders/`.

Buat migration:

```bash
novacore make:migration create_products_table
```

Command ini membuat file:

```text
migrations/<timestamp>_create_products_table.up.sql
migrations/<timestamp>_create_products_table.down.sql
```

Isi file `.up.sql` dengan SQL untuk menerapkan perubahan. Isi file `.down.sql` dengan SQL rollback sebagai dokumentasi rollback.

Jalankan migration:

```bash
novacore migrate
```

NovaCore mencatat migration yang sudah jalan di tabel `schema_migrations`, jadi file `.up.sql` yang sama tidak dijalankan berulang.

Buat seeder:

```bash
novacore make:seeder create_admin_user
```

Isi file `seeders/<timestamp>_create_admin_user.sql` dengan SQL data awal. Seeder sebaiknya idempotent, misalnya memakai `INSERT ... ON CONFLICT`, `INSERT IGNORE`, atau pola SQL sejenis sesuai database.

Jalankan seeder:

```bash
novacore seed
```

NovaCore mencatat seeder yang sudah jalan di tabel `seed_history`, jadi file seeder yang sama tidak dijalankan berulang.

## Postman

File Postman tersedia di:

```text
postman/collection.json
postman/environment.json
```

Import kedua file ke Postman. Isi variable `token` dengan `access_token` dari login.

Saat CRUD digenerate, collection akan diperbarui otomatis dengan folder module baru.

## Response Format

Success:

```json
{
  "success": true,
  "message": "Data retrieved successfully",
  "data": {},
  "meta": {}
}
```

Error:

```json
{
  "success": false,
  "message": "Validation failed",
  "errors": {}
}
```

## Struktur Project

```text
cmd/              entrypoint server dan CLI
internal/         kode aplikasi privat
pkg/              package reusable
config/           file konfigurasi tambahan
database/         file database lokal
migrations/       SQL migration
routes/           ruang route publik tambahan
middleware/       ruang middleware tambahan
modules/          ruang module publik tambahan
docs/             dokumentasi lengkap
scripts/          script operasional
postman/          collection dan environment
tests/            integration/e2e tests
seeders/          data seeder
```

## Dokumentasi

Dokumentasi detail tersedia di:

- `docs/introduction.md`
- `docs/installation.md`
- `docs/configuration.md`
- `docs/database.md`
- `docs/authentication.md`
- `docs/generator.md`
- `docs/routing.md`
- `docs/middleware.md`
- `docs/postman.md`
- `docs/deployment.md`
- `docs/testing.md`
- `docs/best-practices.md`

## Testing

```bash
go test ./...
```

Sebelum publish atau deploy:

```bash
gofmt -w .
go mod tidy
go test ./...
```

## Deployment

Build binary:

```bash
go build -o bin/novacore cmd/novacore/main.go
```

Production checklist:

- Set `APP_ENV=production`
- Gunakan `JWT_SECRET` yang kuat
- Gunakan database production
- Jalankan migration
- Batasi CORS origin
- Jalankan aplikasi di balik reverse proxy atau container orchestrator


## License

NovaCore is open-sourced software licensed under the MIT license.
