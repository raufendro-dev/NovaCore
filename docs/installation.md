# Installation

```bash
git clone <repository-url> novacore
cd novacore
cp .env.example .env
go mod tidy
go install ./cmd/novacore
novacore setup
novacore run
```

Buat project baru dari direktori mana pun:

```bash
novacore create nama-project
cd nama-project
go mod tidy
novacore run
```

Gunakan module path custom jika project akan dipublish ke repository sendiri:

```bash
novacore create nama-project --module github.com/raufendro/nama-project
```

Jika `novacore` belum terbaca oleh shell:

```bash
"$(go env GOPATH)/bin/novacore" setup
source ~/.zshrc
novacore version
```

`go install` menyimpan binary ke `$(go env GOPATH)/bin`. `novacore setup` mengambil `pwd` sebagai `NOVACORE_HOME`, lalu menambahkan Go bin directory ke `PATH` untuk user saat ini.

OS yang didukung:

| OS | File/profile yang diperbarui |
| --- | --- |
| macOS | `~/.zshrc`, `~/.bashrc`, atau `~/.profile` sesuai shell |
| Linux | `~/.bashrc`, `~/.zshrc`, `~/.profile`, atau fish config |
| Windows | PowerShell user profile |

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

Command ini mencari root project NovaCore, menjalankan `git pull`, menghapus binary CLI lama, lalu menjalankan ulang `go install ./cmd/novacore`. Command ini bisa dijalankan dari folder repository atau dari direktori lain seperti `~`.

NovaCore akan otomatis mencari clone project di folder umum seperti `~/Developer`, `~/Projects`, `~/Project`, `~/Code`, dan `$(go env GOPATH)/src`.

Jika sebelumnya kamu sudah install NovaCore versi lama yang belum punya auto-discovery, jalankan sekali dari folder repository:

```bash
go install ./cmd/novacore
novacore setup
```

Jika project berada di lokasi lain, set `NOVACORE_HOME`:

```bash
echo 'export NOVACORE_HOME="/path/to/novacore"' >> ~/.zshrc
source ~/.zshrc
novacore update
```

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
