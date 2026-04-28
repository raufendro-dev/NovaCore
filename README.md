# NovaCore

NovaCore adalah framework backend Golang untuk membangun REST API yang modular, production-ready, dan nyaman dipakai dari tahap prototyping sampai aplikasi serius. Framework ini sudah membawa JWT authentication, refresh token, RBAC middleware, adapter database, generator CRUD, migration, seeder, dokumentasi endpoint, dan Postman Collection otomatis.

NovaCore dibuat supaya developer junior bisa memahami struktur backend yang rapi, sementara developer senior tetap mendapat fondasi yang mudah dikembangkan.

## Daftar Isi

- [Fitur Utama](#fitur-utama)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Instalasi CLI](#instalasi-cli)
- [Create Project](#create-project)
- [Command Reference](#command-reference)
- [Konfigurasi](#konfigurasi)
- [Authentication](#authentication)
- [Generate CRUD](#generate-crud)
- [Update CRUD](#update-crud)
- [Relasi Model](#relasi-model)
- [Database](#database)
- [Migration dan Seeder](#migration-dan-seeder)
- [Postman](#postman)
- [Response Format](#response-format)
- [Struktur Project](#struktur-project)
- [Dokumentasi Lanjutan](#dokumentasi-lanjutan)
- [Testing](#testing)
- [Deployment](#deployment)
- [License](#license)

## Fitur Utama

- Struktur project backend Go yang bersih dan scalable
- REST API dengan response format konsisten
- JWT authentication, refresh token, logout, dan current user endpoint
- Password hashing dengan bcrypt
- Role-based access control
- Middleware logger, recovery, CORS, request ID, rate limiter, timeout, dan secure headers
- Database support: SQLite, MySQL, PostgreSQL, MongoDB
- Redis optional untuk cache/session
- CLI generator untuk module, model, controller, service, repository, endpoint, CRUD, migration, seeder, dan relasi
- CRUD generated protected by JWT secara default
- Opsi CRUD publik dengan flag `--public`
- Safe update CRUD dengan migration `ALTER TABLE`
- Auto-update Postman Collection dan Environment
- Dokumentasi lengkap di folder `docs/`
- Testing setup untuk service dan generated CRUD

## Tech Stack

NovaCore memakai library yang stabil dan umum dipakai di ekosistem Go:

| Kebutuhan | Library |
| --- | --- |
| HTTP router | Gin |
| SQL ORM | GORM |
| MongoDB | MongoDB official driver |
| Redis | Redis Go client |
| Config loader | Viper |
| CLI | Cobra |
| Logging | Zap |
| Password hashing | bcrypt |
| JWT | `golang-jwt/jwt` |
| Validation | `validator/v10` |

## Quick Start

Membuat project baru dari mana saja:

```bash
novacore create nama-project
cd nama-project
go mod tidy
novacore run
```

Jika sedang mengembangkan source framework NovaCore langsung:

```bash
git clone https://github.com/raufendro-dev/NovaCore
cd novacore
cp .env.example .env
go mod tidy
go install ./cmd/novacore
novacore setup
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

## Instalasi CLI

Install command `novacore`:

```bash
go install ./cmd/novacore
```

Set environment otomatis dari folder project saat ini:

```bash
novacore setup
```

`novacore setup` mengambil `pwd` sebagai `NOVACORE_HOME`, lalu menambahkan Go bin directory ke `PATH` untuk user saat ini.

OS yang didukung:

| OS | File/profile yang diperbarui |
| --- | --- |
| macOS | `~/.zshrc`, `~/.bashrc`, atau `~/.profile` sesuai shell |
| Linux | `~/.bashrc`, `~/.zshrc`, `~/.profile`, atau fish config |
| Windows | PowerShell user profile |

Setelah setup, restart terminal atau reload profile shell.

Lihat versi:

```bash
novacore version
```

Output versi:

```text
NovaCore CLI
Version     : 0.2.0
Framework   : Production-ready Go REST API framework
Author      : Rauf Endro Widagdo aka raufendro
Repository  : github.com/raufendro-dev/NovaCore
License     : MIT
```

Jika setelah `go install` muncul `zsh: command not found: novacore`, jalankan binary langsung dari Go bin lalu setup:

```bash
"$(go env GOPATH)/bin/novacore" setup
source ~/.zshrc
novacore version
```

Cek lokasi binary hasil install:

```bash
go env GOPATH
ls "$(go env GOPATH)/bin/novacore"
```

Alternatif tanpa mengubah `PATH`:

```bash
go build -o bin/novacore cmd/novacore/main.go
./bin/novacore run
```

Update project framework dari repository Git:

```bash
novacore update
```

`novacore update` dipakai untuk memperbarui source framework/CLI NovaCore. Command ini mencari folder root repository NovaCore, menjalankan `git pull`, menghapus binary CLI lama, lalu menjalankan ulang `go install ./cmd/novacore`.

`novacore update` tidak dirancang untuk menimpa project aplikasi user yang sudah banyak custom code.

NovaCore akan otomatis mencari clone project di folder umum seperti `~/Developer`, `~/Projects`, `~/Project`, `~/Code`, dan `$(go env GOPATH)/src`.

Jika sebelumnya kamu sudah install NovaCore versi lama yang belum punya auto-discovery, jalankan sekali dari folder repository:

```bash
go install ./cmd/novacore
novacore setup
```

Jika project berada di lokasi lain, set lokasi project dengan `NOVACORE_HOME`:

```bash
echo 'export NOVACORE_HOME="/path/to/novacore"' >> ~/.zshrc
source ~/.zshrc
novacore update
```

Upgrade project aplikasi user secara aman:

```bash
novacore upgrade
```

`novacore upgrade` dipakai di dalam project aplikasi yang dibuat dari NovaCore. Command ini melakukan patch bertahap dan aman ke file yang diperlukan, membuat backup terlebih dahulu, dan tidak melakukan overwrite brutal ke seluruh project.

Upgrade auth role:

```bash
novacore upgrade auth-role
```

Command ini menambahkan dukungan `role` pada auth user lama:

- menambah field `role` pada model user jika belum ada
- menambah parameter `role` pada register request jika belum ada
- menambah normalisasi role di service register
- membuat backup di `.novacore/backups/`
- membuat catatan SQL manual di `migrations/*_add_role_to_users.manual.sql` untuk production database yang tidak memakai AutoMigrate

Untuk hanya mengecek tanpa mengubah file:

```bash
novacore upgrade auth-role --check
```

Uninstall binary CLI NovaCore:

```bash
novacore uninstall
```

`novacore uninstall` menghapus binary `novacore` yang sedang dijalankan. Jika command dijalankan via `go run`, uninstall akan ditolak karena binary tersebut hanya file sementara Go.

## Create Project

Buat project backend baru dari template NovaCore:

```bash
novacore create nama-project
```

Contoh:

```bash
novacore create nama-project
cd nama-project
go mod tidy
novacore run
```

Command ini bisa dijalankan dari direktori mana pun. NovaCore akan mencari source template dari `NOVACORE_HOME` atau folder clone NovaCore yang terdeteksi otomatis.

Secara default module Go diambil dari nama folder project. Contoh `novacore create nama-project` menghasilkan:

```go
module nama-project
```

Jika ingin memakai module path sendiri:

```bash
novacore create nama-project --module github.com/raufendro/nama-project
```

Yang dilakukan oleh `create`:

- Membuat folder project baru
- Copy isi framework NovaCore
- Rewrite `go.mod` dan import internal ke module project baru
- Membuat `.env` dari `.env.example`
- Mengecualikan file lokal seperti `.git`, `.env`, `.cache`, `bin`, dan file database lokal

## Command Reference

| Command | Fungsi |
| --- | --- |
| `novacore create nama-project` | Membuat project backend baru dari template NovaCore |
| `novacore check` | Mengecek direktori project aktif, module, `.env`, entrypoint server, dan Postman environment |
| `novacore run` | Menjalankan HTTP server |
| `novacore setup` | Set `NOVACORE_HOME` dari `pwd` dan menambahkan Go bin ke `PATH` |
| `novacore version` | Menampilkan versi, author, repository, dan license |
| `novacore update` | Menjalankan `git pull`, uninstall binary lama, lalu install CLI terbaru |
| `novacore upgrade` | Patch aman untuk project aplikasi user |
| `novacore upgrade auth-role` | Menambahkan dukungan role ke auth user lama |
| `novacore uninstall` | Menghapus binary CLI NovaCore |
| `novacore make:module User` | Generate module |
| `novacore make:model Product` | Generate model |
| `novacore make:controller Product` | Generate controller/handler |
| `novacore make:service Product` | Generate service |
| `novacore make:repository Product` | Generate repository |
| `novacore make:endpoint Product` | Generate endpoint |
| `novacore make:crud Product` | Generate CRUD lengkap |
| `novacore update:crud Product` | Update CRUD yang sudah pernah digenerate |
| `novacore make:relation Product Category --type=belongs-to` | Generate relasi antar model |
| `novacore make:migration create_products_table` | Generate file migration SQL |
| `novacore migrate` | Menjalankan migration |
| `novacore make:seeder create_admin_user` | Generate file seeder |
| `novacore seed` | Menjalankan seeder |

Flag penting:

| Flag | Dipakai di | Fungsi |
| --- | --- | --- |
| `--public` | `make:crud`, `make:module`, `make:endpoint` | Route tidak memakai JWT middleware |
| `--safe` | `update:crud` | Membuat migration aman memakai `ALTER TABLE` |
| `--mode=safe` | `update:crud` | Sama seperti `--safe` |
| `--type=belongs-to` | `make:relation` | Menentukan tipe relasi |
| `--nested` | `make:relation` | Membuat nested endpoint |
| `--include` | `make:relation` | Mengaktifkan include query |

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

## Authentication

Auth endpoint sudah built-in, jadi tidak perlu digenerate.

Untuk database SQL (`sqlite`, `mysql`, `postgres`), tabel `users` auth otomatis dibuat saat server start:

```bash
novacore run
```

NovaCore menjalankan GORM `AutoMigrate` untuk model auth bawaan. Jadi setelah clone, `cp .env.example .env`, dan start server, tabel user auth sudah siap dipakai. Untuk production schema aplikasi, tetap disarankan memakai migration agar perubahan database tercatat.

| Method | Endpoint | Keterangan |
| --- | --- | --- |
| `POST` | `/api/v1/auth/register` | Membuat user baru |
| `POST` | `/api/v1/auth/login` | Login dan mendapatkan token |
| `POST` | `/api/v1/auth/refresh` | Membuat access token baru |
| `POST` | `/api/v1/auth/logout` | Logout client-side |
| `GET` | `/api/v1/auth/me` | Mengambil user saat ini |

Contoh register:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Demo User","email":"demo@example.com","password":"password123","role":"user"}'
```

Field `role` bersifat optional. Jika tidak dikirim, NovaCore otomatis memakai role `user`. Nilai role akan ikut masuk ke JWT claims dan bisa dipakai oleh middleware RBAC seperti `middleware.Role("admin")`.

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

Untuk CRUD publik:

```bash
novacore make:crud Article --public
```

Sebelum generate, CLI akan menanyakan metode endpoint yang ingin dibuat, lalu field yang ingin disimpan. Field default `id`, `created_at`, `updated_at`, dan `deleted_at` sudah otomatis tersedia dari `gorm.Model`, jadi tidak perlu dimasukkan.

Tekan `Ctrl+D` pada prompt `Field name` untuk menyelesaikan pengisian field.

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

Metode endpoint:

| Method | Endpoint |
| --- | --- |
| `GET` | `/api/v1/products` dan `/api/v1/products/:id` |
| `POST` | `/api/v1/products` |
| `PUT` | `/api/v1/products/:id` |
| `PATCH` | `/api/v1/products/:id` |
| `DELETE` | `/api/v1/products/:id` |

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

Setelah generate:

```bash
go test ./...
novacore migrate
novacore run
```

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

Mode reset akan meminta konfirmasi:

```text
Updating CRUD columns will create a reset migration.
Existing table data will be deleted and IDs will restart from 0 after the migration is run.
Continue? [y/N]:
```

Jawab `y` atau `Y` untuk lanjut. Jawab `n`, `N`, atau kosong untuk membatalkan.

Gunakan `--safe` agar migration memakai `ALTER TABLE` dan tidak menjalankan `DROP TABLE`. Jika field baru required, default value wajib diisi agar row lama tetap valid.

## Relasi Model

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

Atau aktifkan auto-create untuk development:

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

Jalankan migration:

```bash
novacore migrate
```

NovaCore mencatat migration yang sudah jalan di tabel `schema_migrations`, jadi file `.up.sql` yang sama tidak dijalankan berulang.

Buat seeder:

```bash
novacore make:seeder create_admin_user
```

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

## Dokumentasi Lanjutan

Dokumentasi detail tersedia di:

- [Introduction](docs/introduction.md)
- [Installation](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Database](docs/database.md)
- [Authentication](docs/authentication.md)
- [Generator](docs/generator.md)
- [Routing](docs/routing.md)
- [Middleware](docs/middleware.md)
- [Postman](docs/postman.md)
- [Deployment](docs/deployment.md)
- [Testing](docs/testing.md)
- [Best Practices](docs/best-practices.md)

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
