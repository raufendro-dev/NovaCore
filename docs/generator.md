# Generator

Command:

```bash
novacore create api-umkm
novacore make:module User
novacore make:model Product
novacore make:controller Product
novacore make:service Product
novacore make:repository Product
novacore make:endpoint Product
novacore make:crud Product
novacore make:migration create_products_table
novacore make:seeder create_admin_user
novacore setup
novacore version
novacore update
novacore uninstall
```

`create` membuat project baru dari template NovaCore di direktori aktif. Default module Go diambil dari nama folder project, dan bisa diubah dengan `--module`.

`make:crud` membuat model, DTO, repository, service, handler, routes, migration, test, docs endpoint, dan Postman request.

`setup` mengambil `pwd` sebagai `NOVACORE_HOME` dan menambahkan Go bin ke `PATH` user. `version` menampilkan versi CLI dan informasi author. `update` mencari root project NovaCore, menjalankan `git pull`, menghapus binary CLI lama, lalu menjalankan ulang `go install ./cmd/novacore`. Pencarian root otomatis membaca folder umum seperti `~/Developer`, `~/Projects`, dan `$(go env GOPATH)/src`; untuk lokasi lain gunakan `NOVACORE_HOME`. `uninstall` menghapus binary `novacore` yang sedang dijalankan, dan akan menolak jika dipanggil lewat `go run`.

Saat menjalankan `make:crud`, CLI akan menanyakan metode endpoint yang ingin dibuat, lalu field yang ingin disimpan. Field default `id`, `created_at`, `updated_at`, dan `deleted_at` sudah disediakan otomatis oleh `gorm.Model`. Tekan `Ctrl+D` pada prompt `Field name` untuk selesai.

Contoh:

```text
Methods [GET, POST, PUT, PATCH, DELETE]: POST, GET, DELETE
Field name: name
Type [string]: string
Required? [y/N]: y
Default value [none]:
Field name: price
Type [string]: float
Required? [y/N]: n
Default value [none]: 0
Field name: <Ctrl+D>
```

Tipe yang didukung: `string`, `text`, `int`, `uint`, `float`, `bool`, `time`.

Metode yang didukung: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`. `GET` menghasilkan route list dan detail.

Jika `Field name` kosong lalu ditekan `Enter`, CLI akan meminta input ulang dan menampilkan informasi bahwa `Ctrl+D` digunakan untuk menyelesaikan input.

CRUD generated otomatis protected dengan JWT middleware. Login terlebih dahulu lewat `/api/v1/auth/login`, lalu kirim header:

```http
Authorization: Bearer <access_token>
```

Untuk CRUD publik, tambahkan flag:

```bash
novacore make:crud Article --public
```

Flag `--public` juga tersedia untuk `make:module` dan `make:endpoint`.

## Update CRUD

Jika ada field atau tipe data yang terlewat setelah CRUD dibuat, gunakan:

```bash
novacore update:crud Product
```

Safe mode:

```bash
novacore update:crud Product --safe
novacore update:crud Product --mode=safe
```

Atau:

```bash
novacore make update-crud Product
```

CLI akan meminta konfirmasi karena migration yang dibuat akan menghapus dan membuat ulang table:

```text
Updating CRUD columns will create a reset migration.
Existing table data will be deleted and IDs will restart from 0 after the migration is run.
Continue? [y/N]:
```

Jawab `y` atau `Y` untuk lanjut. Jawab `n`, `N`, atau kosong untuk batal.

Setelah itu CLI akan menanyakan ulang metode endpoint dan field. Generator memperbarui model, DTO, service, repository, handler, routes, test, migration, docs endpoint, dan Postman request.

Safe mode menghasilkan migration `ALTER TABLE`, bukan `DROP TABLE`. Field baru yang required wajib memiliki default value.

## Relation Generator

```bash
novacore make:relation Product Category --type=belongs-to --include
novacore make:relation Category Product --type=has-many --nested
novacore make:relation User Role --type=many-to-many
novacore make:relation User Profile --type=has-one
```

Mode interaktif:

```bash
novacore make:relation
```

Include query didukung untuk relasi yang didaftarkan:

```http
GET /api/v1/products?include=category
GET /api/v1/orders?include=user,items
```
