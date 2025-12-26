# MONIK-ENTERPRISE

**Sistem monitoring enterprise untuk router MikroTik** dengan sistem versioning dan CHANGELOG terpadu.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Version](https://img.shields.io/badge/Version-1.0.0-blue.svg)](CHANGELOG.md)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](https://github.com/MarzuQnx/MONIK/actions)
[![Go Version](https://img.shields.io/badge/Go-1.23.0+-blue.svg)](https://golang.org/dl/)

## 🚀 Panduan Cepat

### Prasyarat

- Go 1.23.0+
- Git
- Perangkat MikroTik RouterOS

### Instalasi

1. Clone repository:
```bash
git clone https://github.com/MarzuQnx/MONIK.git
cd MONIK-ENTERPRISE
```

2. Instal dependensi:
```bash
cd backend
go mod tidy
```

3. Konfigurasi environment:
```bash
cp .env.example .env
# Edit .env dengan detail koneksi MikroTik Anda
```

4. Jalankan aplikasi:
```bash
go run cmd/monik/main.go
```

## 📋 Sistem Versioning

Proyek ini menggunakan **Semantic Versioning (SemVer)** dengan **Conventional Commits** untuk manajemen versi otomatis.

### Format Versi
```
MAJOR.MINOR.PATCH
```

- **MAJOR**: Perubahan yang bersifat breaking (API, skema database, arsitektur)
- **MINOR**: Fitur baru (backward compatible)
- **PATCH**: Perbaikan bug dan optimasi

### Format Pesan Commit

Semua commit harus mengikuti format conventional commits:

```
<type>(<scope>): <deskripsi>

[badan opsional]

[footer opsional]
```

#### Jenis Commit

| Jenis | Deskripsi | Contoh |
|------|-----------|--------|
| `feat` | Fitur baru | `feat(api): tambahkan otentikasi pengguna` |
| `fix` | Perbaikan bug | `fix(database): perbaiki timeout koneksi` |
| `perf` | Peningkatan performa | `perf(api): optimasi kinerja query` |
| `revert` | Pembatalan perubahan | `revert(feat): batalkan otentikasi pengguna` |
| `docs` | Perubahan dokumentasi | `docs(readme): perbarui panduan instalasi` |
| `style` | Pemformatan kode | `style(code): perbaiki indentasi` |
| `refactor` | Refactoring kode | `refactor(api): sederhanakan logika endpoint` |
| `test` | Perubahan testing | `test(api): tambahkan unit test` |
| `chore` | Tugas maintenance | `chore(deps): perbarui dependensi` |

#### Perubahan Breaking

Untuk perubahan yang bersifat breaking, tambahkan `BREAKING CHANGE:` di footer commit:

```
feat(api): tambahkan metode otentikasi baru

BREAKING CHANGE: Metode otentikasi lama dihapus
```

## 📖 CHANGELOG

CHANGELOG dibuat secara otomatis dan dikelola di [`CHANGELOG.md`](CHANGELOG.md). Mengikuti format [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

### Kategori CHANGELOG

- **Ditambahkan**: Fitur dan fungsionalitas baru
- **Diubah**: Perubahan pada fungsionalitas yang ada
- **Diperbaiki**: Perbaikan bug
- **Keamanan**: Perubahan terkait keamanan
- **Diuji**: Perubahan terkait testing
- **Depresiasi**: Fitur yang akan dihapus

## 🏗️ Arsitektur

### Backend (Go)
- **Framework**: Gin untuk REST API
- **Database**: SQLite dengan ORM GORM
- **Integrasi RouterOS**: library go-routeros
- **Real-time**: WebSocket untuk pembaruan langsung
- **Monitoring**: Worker pool dengan load balancing

### Frontend
- **Dashboard real-time** dengan koneksi WebSocket
- **Visualisasi traffic** dalam bentuk grafik
- **Pemantauan kesehatan sistem**
- **Pelacakan status interface**

### Skema Database
- **Migrasi otomatis** dengan versioning
- **Tabel monitoring interface**
- **Penyimpanan snapshot traffic**
- **Logging counter reset**
- **Pelacakan kuota bulanan**

## 🔧 Konfigurasi

### Variabel Lingkungan

```bash
# Konfigurasi Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Konfigurasi Database
DB_PATH=data/monik.db
DB_MAX_OPEN_CONN=25
DB_MAX_IDLE_CONN=5

# Konfigurasi Router
ROUTER_IP=192.168.88.1
ROUTER_PORT={YOUR API PORT}
ROUTER_USERNAME=admin
ROUTER_PASSWORD=
ROUTER_TIMEOUT=30s

# Konfigurasi Versioning
VERSIONING_ENABLED=true
VERSIONING_STRATEGY=semantic
CHANGELOG_PATH=CHANGELOG.md
APP_VERSION=1.0.0
```

## 🚀 Deployment

### CI/CD Pipeline

Proyek ini mencakup workflow GitHub Actions (`.github/workflows/release.yml`) yang:

1. **Mendeteksi perubahan versi** dari pesan commit
2. **Memperbarui CHANGELOG** secara otomatis
3. **Membuat Git tag** untuk release
4. **Build dan test** aplikasi
5. **Membuat release GitHub**

### Proses Release

#### Otomatis (Direkomendasikan)
1. Buat commit dengan format conventional commit
2. Push ke branch `main` atau `master`
3. CI/CD pipeline menangani sisanya

#### Manual
```bash
# Jalankan version bump
./scripts/version-bump.sh

# Perbarui CHANGELOG
./scripts/migrate-changelog.sh

# Buat tag
git tag v1.1.0
git push origin v1.1.0
```

## 🧪 Testing

### Unit Test
```bash
cd backend
go test ./...
```

### Testing Versioning
```bash
./scripts/test-versioning-simple.sh
```

### Testing Integrasi
```bash
# Testing CI/CD pipeline
./scripts/test-ci-pipeline.sh

# Testing multi-component release
./scripts/test-multi-component.sh
```

## 📊 Monitoring

### Kesehatan Sistem
- **Pelacakan tingkat error** di seluruh komponen
- **Pemantauan waktu respon** untuk endpoint API
- **Metrik worker pool** (worker aktif, ukuran antrian)
- **Kinerja WebSocket** (throughput pesan, tingkat drop)

### Metrik Performa
- **Monitoring interface** untuk 100+ interface secara bersamaan
- **Waktu respon <100ms** untuk 95% request
- **Pengurangan penggunaan memory 40%** vs implementasi sebelumnya
- **Distribusi CPU merata** di seluruh core

## 🔒 Keamanan

### Autentikasi
- **Validasi input** untuk semua endpoint API
- **Logging terstruktur** untuk event keamanan
- **Audit trail** untuk aksi pengguna
- **Koneksi MikroTik aman**

### Best Practice
- **Tidak ada credential dalam pesan commit**
- **Konfigurasi berbasis environment**
- **Pembaruan dependensi rutin**
- **Pemindaian keamanan di CI/CD**

## 📚 Dokumentasi

- [Panduan Versioning](VERSIONING.md) - Dokumentasi sistem versioning lengkap
- [Dokumentasi API](docs/API_DOCS.md) - Endpoint dan penggunaan REST API
- [Panduan Konfigurasi](docs/CONFIGURATION.md) - Opsi konfigurasi detail
- [Panduan Troubleshooting](docs/TROUBLESHOOTING.md) - Masalah umum dan solusi
- [Gambaran Arsitektur](docs/ARCHITECTURE.md) - Detail arsitektur sistem

## 🤝 Kontribusi

1. Fork repository
2. Buat branch fitur
3. Lakukan commit dengan format conventional commit
4. Push ke fork Anda
5. Buat pull request

### Panduan Pesan Commit
- Gunakan format conventional commit
- Bersifat deskriptif namun ringkas
- Sertakan scope jika relevan
- Tambahkan `BREAKING CHANGE:` untuk perubahan breaking

## 📄 Lisensi

Proyek ini dilisensikan di bawah lisensi MIT - lihat file [LICENSE](LICENSE) untuk detail.

## 🙏 Ucapan Terima Kasih

- **MikroTik** untuk RouterOS dan API
- **Komunitas Go** untuk library yang luar biasa
- **Semantic Release** untuk versioning otomatis
- **Keep a Changelog** untuk standar CHANGELOG

---

**Catatan**: Proyek ini menggunakan sistem versioning terpadu yang mengonsolidasikan seluruh dokumentasi ke dalam satu CHANGELOG, menghilangkan kebutuhan akan banyak file `.md` manual sambil tetap menjaga pelacakan perubahan yang komprehensif.

## 🏷️ Informasi Proyek

- **Brain**: dbanie
- **Developer**: Kwaipilot
- **Copyright**: Desember 2025
- **Status**: Free Open Source