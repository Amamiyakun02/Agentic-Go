---
title: ViennaGo
emoji: 🏆
colorFrom: blue
colorTo: purple
sdk: docker
pinned: false
license: mit
short_description: backend server for vienna ai base go / gin
---

# AgenticGo (Vienna AI - Go Backend)

AgenticGo adalah backend server berkinerja tinggi berbasis **Go (Gin Gonic)** untuk asisten AI **Vienna**. Project ini mengimplementasikan koneksi dua arah real-time menggunakan **Gemini Live API (WebSocket)** untuk audio streaming, pemanggilan tools (function calling) terpadu, dan manajemen sesi yang sinkron dengan chat teks.

Seluruh stack teknologi API dan database yang digunakan dalam project ini memanfaatkan **layanan/tier gratis (Free Tier)**.

---

## 🎯 Tujuan Project
Menyediakan backend voice assistant yang super responsif, aman dari race-condition, hemat sumber daya (efisiensi memory & CPU) dengan memanfaatkan fitur unggulan Go seperti **Goroutines**, **Channels**, dan **Context-based cancellation**.

---

## ✨ Fitur Utama
1. **Gemini Live API Bidirectional Audio**: Aliran audio dua arah real-time tanpa penundaan (low latency).
2. **Unified Tool Calling (Function Calling)**:
   - **RAG (Retrieval-Augmented Generation)**: Pencarian otomatis dokumen internal via Qdrant Vector DB.
   - **Device Control**: Navigasi dan kontrol device Android (buka aplikasi, klik teks, input text, baca layar).
   - **Spotify Integration**: Play, pause, dan skip track langsung melalui perintah suara.
3. **Session Synchronization**: Menghubungkan sesi Live Voice dengan riwayat chat teks sehingga agen memiliki konteks percakapan penuh.
4. **Heartbeat Mechanism**: Ping/pong otomatis setiap 25 detik untuk menjaga stabilitas WebSocket.
5. **Thread-Safe WebSocket Writes**: Menggunakan sinkronisasi mutex untuk mencegah tabrakan data (race conditions) saat menulis ke koneksi WebSocket Gemini.
6. **Graceful Goroutine Cleanup**: Lifecycle goroutine dikendalikan menggunakan `context.WithCancel` untuk mencegah kebocoran memori (goroutine leaks).

---

## 🛠️ Stack Layanan Gratis (Free Tier APIs)
Semua API dan database yang digunakan dalam konfigurasi ini bersifat gratis:
* **Gemini API & Gemini Live Preview**: Menggunakan model `gemini-2.0-flash-exp` (gratis pada Google AI Studio preview tier).
* **Qdrant Cloud**: Vector database gratis untuk pencarian dokumen/RAG.
* **MongoDB Atlas**: Database gratis untuk menyimpan riwayat percakapan secara permanen.
* **Supabase**: Backend database & object storage gratis.
* **Redis Labs Cloud**: Layanan Redis cache gratis.
* **Fonnte**: API Gateway untuk integrasi pesan otomatis ke WhatsApp.

---

## 📋 Panduan Konfigurasi & Setup

### 1. Prasyarat (Prerequisites)
Pastikan Anda sudah menginstal:
* [Go](https://go.dev/doc/install) (versi 1.22 atau yang lebih baru)

### 2. File Lingkungan (`.env`)
Salin file `.env.example` menjadi `.env` di direktori root project:
```bash
cp .env.example .env
```

Isi variabel di dalam `.env` dengan kredensial gratis Anda:
* **Gemini API Key**: Dapatkan dari [Google AI Studio](https://aistudio.google.com/).
* **Mongo DB URL & Credentials**: Buat cluster gratis di [MongoDB Atlas](https://www.mongodb.com/products/platform/atlas-database).
* **Qdrant DB API Key & URL**: Buat cluster gratis di [Qdrant Cloud](https://qdrant.to/cloud).
* **Redis Host, Port & Password**: Buat database Redis gratis di [Redis Labs Cloud](https://redislabs.com/).
* **Supabase credentials**: Buat project gratis di [Supabase](https://supabase.com/).

### 3. Menjalankan Aplikasi secara Lokal
Unduh dependensi project terlebih dahulu:
```bash
go mod tidy
```

Jalankan server:
```bash
go run main.go
```
Secara default, server Gin akan berjalan di port `8080` (atau port yang disesuaikan dalam konfigurasi Anda).

### 4. Menjalankan Pengujian (Testing)
Pengujian unit dipisahkan dalam folder `/test` (diabaikan dari pelacakan git agar repository tetap bersih). Anda dapat menjalankan tes secara lokal menggunakan perintah:
```bash
go test ./test/... -v
```

---

## 🔒 Keamanan Git
File `.env` berisi kunci sensitif yang tidak boleh di-push ke GitHub. Project ini telah dikonfigurasi dengan `.gitignore` untuk mengecualikan `.env` dan folder `/test`. Selalu gunakan `.env.example` sebagai referensi saat melakukan pembagian konfigurasi.
