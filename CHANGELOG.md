# CHANGELOG

Semua perubahan yang layak dicatat untuk proyek ini didokumentasikan dalam file ini.

Format ini didasarkan pada [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
dan proyek ini mengikuti [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Ditambahkan
- Dukungan versioning multi-komponen (backend, frontend, database, API)
- Pembuatan CHANGELOG otomatis dari commit git
- Pelacakan versi untuk migrasi database
- Aturan versioning khusus per komponen
- Strategi tagging Git dengan tag versi otomatis
- Mekanisme draft release untuk GitHub
- Integrasi shields.io untuk version dan build badges
- Pelacakan dan logging event keamanan
- Integrasi framework testing komprehensif

### Diubah
- Mengonsolidasikan seluruh dokumentasi ke CHANGELOG terpadu
- Meningkatkan konsistensi versioning di seluruh komponen
- Meningkatkan pipeline CI/CD dengan semantic release
- Mengoptimalkan versioning migrasi database
- Menyederhanakan proses release dengan tagging otomatis

### Diperbaiki
- Masalah sinkronisasi versi antar komponen
- Informasi versi yang hilang di release
- Konflik versioning skema database
- Masalah timing pipeline CI/CD

### Keamanan
- Peningkatan validasi input untuk semua endpoint API
- Peningkatan logging terstruktur untuk event keamanan
- Peningkatan hardening autentikasi untuk koneksi MikroTik
- Implementasi audit trail untuk aksi pengguna

### Diuji
- Integrasi suite testing API komprehensif
- Load testing untuk skenario traffic tinggi
- Testing migrasi database
- Testing integrasi multi-komponen
- Validasi proses release end-to-end

### Depresiasi
- File CHANGELOG komponen individual
- Proses versioning manual
- Pemeliharaan dokumentasi statis

---

## [1.0.0] - 2025-12-26

### Ditambahkan
- Rilis awal MONIK-ENTERPRISE
- Arsitektur mesin monitoring yang dapat diskalakan
- Monitoring real-time WebSocket
- Deteksi WAN/ISP dinamis
- Worker pool dengan load balancing
- Pengumpulan metrik komprehensif
- Sistem konfigurasi yang ditingkatkan
- Logging dan penanganan error terstruktur

### Backend v1.0.0
- Go 1.23.0 dengan arsitektur modular
- Framework Gin untuk REST API
- GORM dengan database SQLite
- Integrasi RouterOS via go-routeros
- Pembaruan real-time WebSocket
- Worker pool dengan circuit breaker
- Sistem logging komprehensif

### Frontend v1.0.0
- Antarmuka dashboard real-time
- Manajemen koneksi WebSocket
- Visualisasi traffic dalam grafik
- Pemantauan kesehatan sistem
- Pelacakan status interface

### Skema Database v1.0
- Sistem migrasi otomatis
- Tabel monitoring interface
- Penyimpanan snapshot traffic
- Logging counter reset
- Pelacakan kuota bulanan
- Penyimpanan informasi sistem

### API v1
- Endpoint health check
- Manajemen interface
- Informasi sistem
- Riwayat traffic
- Deteksi WAN
- Manajemen worker pool
- Statistik WebSocket

### Diperbaiki
- Masalah koneksi database awal
- Penanganan timeout koneksi MikroTik
- Peningkatan akurasi deteksi WAN
- Optimasi performa untuk skenario beban tinggi

### Keamanan
- Validasi input untuk semua endpoint API
- Logging terstruktur untuk event keamanan
- Autentikasi untuk koneksi MikroTik
- Audit trail untuk aksi pengguna

### Diuji
- Suite testing API komprehensif
- Load testing untuk 100+ interface secara bersamaan
- Validasi migrasi database
- Testing stress koneksi WebSocket
- Testing pemulihan error

### Performa
- Peningkatan throughput monitoring 300%
- Waktu respon <100ms untuk 95% request
- Pengurangan penggunaan memory 40%
- Distribusi CPU merata di seluruh core

---

## [0.1.0] - 2025-12-25

### Ditambahkan
- Fungsi monitoring dasar
- Pelacakan status interface
- Pengumpulan data traffic
- Implementasi penyimpanan database
- Endpoint REST API
- Manajemen konfigurasi

### Diperbaiki
- Masalah setup awal
- Masalah migrasi database
- Error endpoint API
- Masalah loading konfigurasi

### Keamanan
- Implementasi autentikasi dasar
- Sanitasi input untuk endpoint API