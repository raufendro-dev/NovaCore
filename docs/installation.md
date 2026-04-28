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

Command ini mencari root repository NovaCore, menjalankan `git pull`, menghapus binary CLI lama, lalu menjalankan ulang `go install ./cmd/novacore`. Command ini memperbarui source framework/CLI, bukan menimpa project aplikasi user yang sudah banyak custom code.

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

Upgrade project aplikasi user:

```bash
novacore upgrade
novacore upgrade auth-role
```

`novacore upgrade` dijalankan di dalam project aplikasi yang dibuat dari NovaCore. Command ini melakukan patch aman dan membuat backup di `.novacore/backups/` sebelum mengubah file. Untuk `auth-role`, NovaCore juga membuat catatan SQL manual di `migrations/` bagi production database yang tidak memakai AutoMigrate. Untuk cek tanpa mengubah file:

```bash
novacore upgrade auth-role --check
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
