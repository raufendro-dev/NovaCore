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

Migration:

```bash
go run cmd/framework/main.go make:migration create_products_table
go run cmd/framework/main.go migrate
```
