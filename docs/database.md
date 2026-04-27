# Database

Pilih database melalui `DB_DRIVER`.

SQL yang didukung:

- `sqlite`
- `mysql`
- `postgres`

Jika memakai MySQL/PostgreSQL, database harus sudah ada sebelum aplikasi connect, kecuali `DB_AUTO_CREATE=true` diaktifkan.

Contoh error:

```text
Error 1049 (42000): Unknown database 'app_db'
```

Artinya server MySQL hidup, user/password benar, tetapi database `app_db` belum dibuat.

Solusi manual:

```sql
CREATE DATABASE app_db;
```

Solusi development:

```env
DB_AUTO_CREATE=true
```

Untuk production, gunakan `DB_AUTO_CREATE=false` dan buat database melalui provisioning/deployment pipeline.

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
novacore run
```

Ini dilakukan oleh GORM `AutoMigrate` pada model auth bawaan. Untuk production, schema module aplikasi sebaiknya tetap dikelola lewat migration.

## Migration

Migration adalah file SQL untuk mengubah struktur database secara terkontrol. Gunakan migration untuk membuat tabel, menambah kolom, membuat index, atau perubahan schema lain.

Migration:

```bash
novacore make:migration create_products_table
novacore migrate
```

File `.up.sql` dijalankan saat migration. NovaCore mencatat file yang sudah dijalankan di tabel `schema_migrations`.

Untuk CRUD generated, kolom migration mengikuti field yang kamu isi saat prompt `make:crud`.

Jika menggunakan `update:crud`, NovaCore membuat migration reset table yang menjalankan `DROP TABLE` lalu membuat table ulang. Data lama akan hilang dan ID akan mulai dari awal setelah migration dijalankan. Gunakan hanya ketika perubahan schema memang boleh menghapus data.

Gunakan `update:crud --safe` untuk migration berbasis `ALTER TABLE`. Safe mode menjaga data lama tetap ada dan menolak field baru yang `required` tanpa default value.

## Seeder

Seeder adalah file SQL untuk mengisi data awal seperti admin pertama, role default, permission default, atau data referensi.

```bash
novacore make:seeder create_admin_user
novacore seed
```

NovaCore mencatat seeder yang sudah dijalankan di tabel `seed_history`.
