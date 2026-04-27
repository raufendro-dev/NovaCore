# Installation

```bash
git clone <repository-url> novacore
cd novacore
cp .env.example .env
go mod tidy
go install ./cmd/novacore
novacore run
```

Untuk membuat CRUD:

```bash
novacore make:crud Product
novacore run
```
