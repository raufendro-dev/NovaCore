# Deployment

Checklist deployment:

- Set `APP_ENV=production`
- Set `JWT_SECRET` yang kuat
- Pilih database production
- Jalankan migration
- Jalankan seeder jika ada data awal yang diperlukan
- Aktifkan CORS origin spesifik
- Jalankan aplikasi di balik reverse proxy atau container orchestrator

Build binary:

```bash
go build -o bin/novacore cmd/server/main.go
```

Urutan deployment yang disarankan:

```bash
go run cmd/framework/main.go migrate
go run cmd/framework/main.go seed
./bin/novacore
```
