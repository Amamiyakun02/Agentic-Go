# --- Stage 1: Builder ---
FROM golang:alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Install git untuk mengunduh dependencies (opsional namun disarankan)
RUN apk add --no-cache git

# Salin go.mod dan go.sum
COPY go.mod go.sum ./

# Unduh semua dependencies
RUN go mod download

# Salin seluruh kode aplikasi ke dalam container
COPY . .

# Build aplikasi Go menjadi file static executable bernama 'agentic_server'
# CGO_ENABLED=0 memastikan binary tidak membutuhkan library C dari OS (100% static)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agentic_server .


# --- Stage 2: Production ---
FROM alpine:latest  

# Install certificate root agar aplikasi bisa melakukan HTTPS request (OpenAI, Qdrant, dll)
# dan tzdata untuk mengatur zona waktu.
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Salin file eksekusi dari stage builder
COPY --from=builder /app/agentic_server .

# Salin file swagger docs jika diperlukan (opsional, sudah masuk ke dalam binary sebenarnya)
# Tapi untuk amannya kita tidak perlu menyalin docs, karena swag telah men-generate docs.go yang dikompilasi.

# Konfigurasi Environment Variables standar Produksi
ENV TZ=Asia/Jakarta
ENV GIN_MODE=release

# Expose port tempat aplikasi berjalan
EXPOSE 7860

# Perintah untuk menjalankan executable
CMD ["./agentic_server"]
