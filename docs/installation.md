# Installation

```bash
git clone <repository-url> novacore
cd novacore
cp .env.example .env
go mod tidy
go install ./cmd/novacore
novacore run
```

Lihat versi:

```bash
novacore version
```

Update framework dari repository Git:

```bash
novacore update
```

Command ini menjalankan `git pull` di direktori project yang sedang aktif.

Uninstall CLI:

```bash
novacore uninstall
```

Uninstall menghapus binary `novacore` yang sedang dijalankan. Jalankan dari binary hasil `go install` atau `go build`, bukan dari `go run`.

Untuk membuat CRUD:

```bash
novacore make:crud Product
novacore run
```
