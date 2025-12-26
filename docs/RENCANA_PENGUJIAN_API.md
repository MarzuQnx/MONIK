# RENCANA PENGUJIAN API MONITORING MIKROTIK

## Ringkasan Proyek

Proyek ini adalah aplikasi monitoring MikroTik berbasis Go yang menyediakan API untuk memantau antarmuka jaringan, statistik sistem, dan data lalu lintas secara real-time. Aplikasi menggunakan arsitektur berbasis layanan dengan database SQLite dan WebSocket untuk pembaruan real-time.

## Struktur Proyek

```
backend/
├── cmd/monik/main.go              # Entry point aplikasi
├── internal/
│   ├── api/handlers.go           # Handler API
│   ├── router/router.go          # Routing API
│   ├── models/models.go          # Model data
│   ├── database/database.go      # Koneksi database
│   ├── service/
│   │   ├── mikrotik.go           # Koneksi MikroTik
│   │   ├── service.go            # Layanan monitoring
│   │   ├── wan_detection.go      # Deteksi WAN
│   │   ├── worker_pool.go        # Pool worker
│   │   └── websocket.go          # WebSocket manager
│   └── config/config.go          # Konfigurasi aplikasi
└── data/monik.db                 # Database SQLite
```

## Daftar Route API

### 1. Health Check
- **Endpoint**: `GET /health`
- **Deskripsi**: Memeriksa status kesehatan aplikasi
- **Parameter**: Tidak ada
- **Response**: Status aplikasi

### 2. Interface Management
- **Endpoint**: `GET /api/v1/interfaces`
- **Deskripsi**: Mendapatkan semua antarmuka jaringan
- **Parameter**: Tidak ada
- **Response**: Daftar antarmuka dengan statistik

- **Endpoint**: `GET /api/v1/interfaces/:name`
- **Deskripsi**: Mendapatkan detail antarmuka tertentu
- **Parameter**: 
  - `name` (path): Nama antarmuka
- **Response**: Detail antarmuka

### 3. System Information
- **Endpoint**: `GET /api/v1/system`
- **Deskripsi**: Mendapatkan informasi sistem router
- **Parameter**: Tidak ada
- **Response**: Informasi sistem (identity, board, version, dll)

### 4. Traffic History
- **Endpoint**: `GET /api/v1/traffic/:interface`
- **Deskripsi**: Mendapatkan riwayat lalu lintas antarmuka
- **Parameter**:
  - `interface` (path): Nama antarmuka
  - `limit` (query, opsional): Jumlah data yang diinginkan (default: 100)
- **Response**: Riwayat lalu lintas dalam format time-series

### 5. Test Data Management
- **Endpoint**: `POST /api/v1/populate-test-data`
- **Deskripsi**: Mengisi database dengan data uji
- **Parameter**: Tidak ada
- **Response**: Konfirmasi penambahan data

### 6. WebSocket Real-time Updates
- **Endpoint**: `GET /api/v1/ws`
- **Deskripsi**: Koneksi WebSocket untuk pembaruan real-time
- **Parameter**: Tidak ada
- **Response**: Koneksi WebSocket

### 7. WAN Detection
- **Endpoint**: `GET /api/v1/wan-interface`
- **Deskripsi**: Mendeteksi antarmuka WAN otomatis
- **Parameter**: Tidak ada
- **Response**: Nama antarmuka WAN yang terdeteksi

- **Endpoint**: `GET /api/v1/wan-stats`
- **Deskripsi**: Mendapatkan statistik deteksi WAN
- **Parameter**: Tidak ada
- **Response**: Statistik deteksi WAN

### 8. Worker Pool Management
- **Endpoint**: `GET /api/v1/worker-status`
- **Deskripsi**: Mendapatkan status pool worker
- **Parameter**: Tidak ada
- **Response**: Status worker, antrian, dan metrik

- **Endpoint**: `POST /api/v1/submit-job`
- **Deskripsi**: Mengirimkan pekerjaan monitoring ke pool
- **Parameter** (JSON body):
  - `interface_name`: Nama antarmuka
  - `type`: Jenis pekerjaan
  - `timeout`: Timeout dalam detik
  - `max_retries`: Jumlah retry maksimal
- **Response**: Konfirmasi pengiriman pekerjaan

### 9. WebSocket Statistics
- **Endpoint**: `GET /api/v1/websocket-stats`
- **Deskripsi**: Mendapatkan statistik koneksi WebSocket
- **Parameter**: Tidak ada
- **Response**: Jumlah klien, subscription, dan channel aktif

## Struktur Data Model

### Interface Model
```go
type Interface struct {
    ID                uint      `json:"id"`
    InterfaceName     string    `json:"interface_name"`
    RxBytes           uint64    `json:"rx_bytes"`
    TxBytes           uint64    `json:"tx_bytes"`
    RxRate            float64   `json:"rx_rate"` // Mbps
    TxRate            float64   `json:"tx_rate"` // Mbps
    LastSeen          time.Time `json:"last_seen"`
    CounterResetCount int       `json:"counter_reset_count"`
    Status            string    `json:"status"` // up, down, unknown
    Comment           string    `json:"comment"`
}
```

### TrafficSnapshot Model
```go
type TrafficSnapshot struct {
    ID            uint      `json:"id"`
    InterfaceName string    `json:"interface_name"`
    Timestamp     time.Time `json:"timestamp"`
    RxBytes       uint64    `json:"rx_bytes"`
    TxBytes       uint64    `json:"tx_bytes"`
    RxRate        float64   `json:"rx_rate"` // Mbps
    TxRate        float64   `json:"tx_rate"` // Mbps
    TotalBytes    uint64    `json:"total_bytes"`
    CounterReset  bool      `json:"counter_reset"`
}
```

## Skenario Pengujian

### 1. Pengujian Koneksi Database

#### Skenario 1.1: Database Normal
- **Deskripsi**: Database tersedia dan dapat diakses
- **Langkah**:
  1. Pastikan file `data/monik.db` ada
  2. Uji endpoint yang membutuhkan database
  3. Verifikasi respon sukses
- **Expected Result**: Semua endpoint database berfungsi normal

#### Skenario 1.2: Database Tidak Ada
- **Deskripsi**: File database tidak ditemukan
- **Langkah**:
  1. Hapus atau rename file `data/monik.db`
  2. Jalankan aplikasi
  3. Uji endpoint database
- **Expected Result**: Aplikasi membuat database baru dan migrasi berjalan

#### Skenario 1.3: Database Rusak
- **Deskripsi**: File database korup
- **Langkah**:
  1. Corrupt file database
  2. Jalankan aplikasi
  3. Amati penanganan error
- **Expected Result**: Aplikasi menangani error dengan baik

### 2. Pengujian Koneksi MikroTik

#### Skenario 2.1: Koneksi MikroTik Sukses
- **Deskripsi**: Router MikroTik dapat diakses
- **Langkah**:
  1. Pastikan router MikroTik aktif
  2. Konfigurasi koneksi benar di `.env`
  3. Uji endpoint interface dan system
- **Expected Result**: Data interface dan system berhasil diambil

#### Skenario 2.2: Router Tidak Dapat Diakses
- **Deskripsi**: Router MikroTik offline atau tidak dapat dihubungi
- **Langkah**:
  1. Matikan router atau ubah IP di `.env`
  2. Uji endpoint yang membutuhkan koneksi MikroTik
- **Expected Result**: Error koneksi ditangani dengan baik

#### Skenario 2.3: Autentikasi Gagal
- **Deskripsi**: Username/password MikroTik salah
- **Langkah**:
  1. Ubah kredensial di `.env` menjadi salah
  2. Uji endpoint MikroTik
- **Expected Result**: Error autentikasi ditangani dengan baik

### 3. Pengujian Real-time Monitoring

#### Skenario 3.1: WebSocket Connection
- **Deskripsi**: Koneksi WebSocket berhasil
- **Langkah**:
  1. Buka koneksi WebSocket ke `/api/v1/ws`
  2. Amati aliran data real-time
  3. Uji subscription ke interface tertentu
- **Expected Result**: Data real-time mengalir dengan baik

#### Skenario 3.2: Multiple WebSocket Clients
- **Deskripsi**: Banyak klien terhubung ke WebSocket
- **Langkah**:
  1. Buka beberapa koneksi WebSocket
  2. Amati performa server
  3. Uji distribusi data ke semua klien
- **Expected Result**: Semua klien menerima data dengan baik

### 4. Pengujian Worker Pool

#### Skenario 4.1: Load Normal
- **Deskripsi**: Beban kerja normal pada worker pool
- **Langkah**:
  1. Submit beberapa job monitoring
  2. Amati distribusi kerja
  3. Periksa metrik worker
- **Expected Result**: Job diproses dengan baik, load seimbang

#### Skenario 4.2: Load Tinggi
- **Deskripsi**: Banyak job dikirim sekaligus
- **Langkah**:
  1. Submit banyak job dalam waktu singkat
  2. Amati queue dan worker
  3. Periksa penanganan overload
- **Expected Result**: Sistem menangani overload dengan baik

### 5. Pengujian WAN Detection

#### Skenario 5.1: WAN Interface Terdeteksi
- **Deskripsi**: Sistem berhasil mendeteksi antarmuka WAN
- **Langkah**:
  1. Pastikan ada antarmuka dengan traffic keluar
  2. Uji endpoint deteksi WAN
  3. Verifikasi hasil deteksi
- **Expected Result**: Antarmuka WAN terdeteksi dengan benar

#### Skenario 5.2: Tidak Ada WAN Interface
- **Deskripsi**: Tidak ada antarmuka yang memenuhi kriteria WAN
- **Langkah**:
  1. Matikan semua koneksi internet
  2. Uji endpoint deteksi WAN
- **Expected Result**: Sistem menangani kondisi tanpa WAN

## Format Curl Commands

### Health Check
```bash
curl -X GET http://localhost:8080/health
```

### Interface Management
```bash
# Get all interfaces
curl -X GET http://localhost:8080/api/v1/interfaces

# Get specific interface
curl -X GET http://localhost:8080/api/v1/interfaces/ether1
```

### System Information
```bash
curl -X GET http://localhost:8080/api/v1/system
```

### Traffic History
```bash
# Get traffic history with default limit
curl -X GET http://localhost:8080/api/v1/traffic/ether1

# Get traffic history with custom limit
curl -X GET "http://localhost:8080/api/v1/traffic/ether1?limit=50"
```

### Test Data
```bash
curl -X POST http://localhost:8080/api/v1/populate-test-data
```

### WAN Detection
```bash
# Get WAN interface
curl -X GET http://localhost:8080/api/v1/wan-interface

# Get WAN stats
curl -X GET http://localhost:8080/api/v1/wan-stats
```

### Worker Pool
```bash
# Get worker status
curl -X GET http://localhost:8080/api/v1/worker-status

# Submit job
curl -X POST http://localhost:8080/api/v1/submit-job \
  -H "Content-Type: application/json" \
  -d '{
    "interface_name": "ether1",
    "type": "monitoring",
    "timeout": 30,
    "max_retries": 3
  }'
```

### WebSocket Stats
```bash
curl -X GET http://localhost:8080/api/v1/websocket-stats
```

### WebSocket Connection (JavaScript)
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onopen = () => {
  console.log('Connected to WebSocket');
  // Subscribe to interface updates
  ws.send(JSON.stringify({
    type: 'subscribe',
    interface: 'ether1'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};
```

## Setup Testing Environment

### 1. Konfigurasi Lingkungan
```bash
# Salin file konfigurasi
cp backend/.env.example backend/.env

# Edit konfigurasi sesuai lingkungan Anda
# Pastikan koneksi MikroTik benar
```

### 2. Menjalankan Aplikasi
```bash
cd backend
go run cmd/monik/main.go
```

### 3. Tools Testing yang Direkomendasikan
- **Postman**: Untuk testing API endpoint
- **curl**: Untuk testing cepat via command line
- **websocat**: Untuk testing WebSocket via command line
- **wrk**: Untuk load testing
- **Go test**: Untuk unit testing

### 4. Script Testing Otomatis
```bash
#!/bin/bash
# test-api.sh

BASE_URL="http://localhost:8080"

echo "=== Testing Health Check ==="
curl -s $BASE_URL/health | jq .

echo -e "\n=== Testing Interfaces ==="
curl -s $BASE_URL/api/v1/interfaces | jq .

echo -e "\n=== Testing System Info ==="
curl -s $BASE_URL/api/v1/system | jq .

echo -e "\n=== Testing Traffic History ==="
curl -s "$BASE_URL/api/v1/traffic/ether1?limit=10" | jq .

echo -e "\n=== Testing WAN Detection ==="
curl -s $BASE_URL/api/v1/wan-interface | jq .

echo -e "\n=== Testing Worker Status ==="
curl -s $BASE_URL/api/v1/worker-status | jq .

echo -e "\n=== Testing WebSocket Stats ==="
curl -s $BASE_URL/api/v1/websocket-stats | jq .
```

## Kriteria Keberhasilan

### 1. Fungsionalitas Dasar
- [ ] Semua endpoint API merespon dengan status 200
- [ ] Data yang dikembalikan sesuai format yang diharapkan
- [ ] Error handling bekerja dengan baik untuk kondisi error

### 2. Koneksi Database
- [ ] Database dapat dibaca dan ditulis
- [ ] Migrasi database berjalan otomatis
- [ ] Error database ditangani dengan baik

### 3. Koneksi MikroTik
- [ ] Koneksi ke router MikroTik berhasil
- [ ] Data interface dan system dapat diambil
- [ ] Error koneksi ditangani dengan baik

### 4. Real-time Monitoring
- [ ] WebSocket connection berhasil
- [ ] Data real-time mengalir dengan lancar
- [ ] Multiple client dapat terhubung

### 5. Performa
- [ ] Response time endpoint < 2 detik
- [ ] Worker pool dapat menangani beban kerja
- [ ] Memory usage stabil dalam jangka panjang

## Dokumentasi Tambahan

- [API Documentation](docs/API_DOCS.md)
- [Configuration Guide](docs/CONFIGURATION.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Architecture Overview](docs/ARCHITECTURE.md)