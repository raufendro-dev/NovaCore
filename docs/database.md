# Database

Pilih database melalui `DB_DRIVER`.

SQL yang didukung:

- `sqlite`
- `mysql`
- `postgres`

MongoDB didukung dengan:

```env
DB_DRIVER=mongodb
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=app_db
```

Repository generated memakai GORM untuk SQL. Untuk MongoDB, gunakan adapter repository sendiri pada module yang membutuhkan document storage.

## Tabel Auth User

Untuk database SQL (`sqlite`, `mysql`, `postgres`), tabel `users` untuk auth otomatis dibuat saat server start:

```bash
go run cmd/server/main.go
```

Ini dilakukan oleh GORM `AutoMigrate` pada model auth bawaan. Untuk production, schema module aplikasi sebaiknya tetap dikelola lewat migration.

## Migration

Migration adalah file SQL untuk mengubah struktur database secara terkontrol. Gunakan migration untuk membuat tabel, menambah kolom, membuat index, atau perubahan schema lain.

Migration:

```bash
go run cmd/framework/main.go make:migration create_products_table
go run cmd/framework/main.go migrate
```

File `.up.sql` dijalankan saat migration. NovaCore mencatat file yang sudah dijalankan di tabel `schema_migrations`.

## Seeder

Seeder adalah file SQL untuk mengisi data awal seperti admin pertama, role default, permission default, atau data referensi.

```bash
go run cmd/framework/main.go make:seeder create_admin_user
go run cmd/framework/main.go seed
```

NovaCore mencatat seeder yang sudah dijalankan di tabel `seed_history`.
