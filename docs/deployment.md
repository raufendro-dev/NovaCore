# Deployment

Checklist deployment:

- Set `APP_ENV=production`
- Set `JWT_SECRET` yang kuat
- Pilih database production
- Jalankan migration
- Aktifkan CORS origin spesifik
- Jalankan aplikasi di balik reverse proxy atau container orchestrator

Build binary:

```bash
go build -o bin/novacore cmd/server/main.go
```
