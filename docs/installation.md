# Installation

```bash
git clone <repository-url> novacore
cd novacore
cp .env.example .env
go mod tidy
go install ./cmd/novacore
novacore run
```

Jika `novacore` belum terbaca oleh shell:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
novacore version
```

`go install` menyimpan binary ke `$(go env GOPATH)/bin`. Command `novacore` hanya bisa dipanggil dari mana saja jika folder tersebut sudah masuk `PATH`.

Alternatif tanpa mengubah `PATH`:

```bash
go build -o bin/novacore cmd/novacore/main.go
./bin/novacore run
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
