# Installation

```bash
git clone <repository-url> novacore
cd novacore
cp .env.example .env
go mod tidy
go run cmd/server/main.go
```

Untuk membuat CRUD:

```bash
go run cmd/framework/main.go make:crud Product
go run cmd/server/main.go
```
