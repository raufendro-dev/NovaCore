# Configuration

NovaCore membaca konfigurasi dari `.env` dan environment variable.

Variabel utama:

- `APP_NAME`, `APP_ENV`, `APP_PORT`, `APP_DEBUG`
- `DB_DRIVER`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- `DB_AUTO_CREATE`
- `MONGO_URI`, `MONGO_DATABASE`
- `JWT_SECRET`, `JWT_ACCESS_EXPIRY`, `JWT_REFRESH_EXPIRY`
- `CORS_ALLOWED_ORIGINS`
- `RATE_LIMIT_REQUESTS`, `RATE_LIMIT_WINDOW`

Default lokal menggunakan SQLite di `database/app.db` agar server langsung bisa dijalankan.

`DB_AUTO_CREATE=true` bisa dipakai saat development untuk membuat database MySQL/PostgreSQL otomatis jika belum ada. Untuk production, biarkan `false`.

Saat memakai SQL database, tabel auth `users` dibuat otomatis ketika server start. Pastikan `.env` sudah menunjuk ke database yang benar sebelum menjalankan:

```bash
novacore run
```
