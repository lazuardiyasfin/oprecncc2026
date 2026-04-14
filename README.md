# Modul 1: Docker

### Deskripsi Singkat
Service "sederhana" yang telah dibuat adalah HTTP server berbasis Go yang melayani temporary file sharing. User dapat meng-upload suatu file dan membagikan link download yang hanya tersedia selama 24 jam sebelum dihapus oleh sistem.

### Endpoint `/health`
Endpoint `/health` pada dasarnya digunakan baik oleh komputer maupun manusia untuk mengecek apakah service sistem yang berkaitan dalam kondisi "live" dan "ready". Pada aplikasi ini, HTTP server akan mengirimkan HTTP status code OK (200) dan konten berupa JSON string yang berisi status "health" sistem secara keseluruhan.

![screenshot /health](/assets/image.png)

![screenshot /health lewat curl](/assets/image-1.png)

### Endpoint-Endpoint Lain

Selain endpoint `/health`, ada juga endpoint-endpoint lain yang dilayani oleh service:

- `/`: Melayani `index.html`, berisi form untuk upload file.

    ![screenshot /](/assets/image-2.png)

- `/upload`: Melayani POST request dengan tipe konten `multipart/form-data` dari user. Pada screenshot terlihat response `text/html`, tetapi sebenarnya endpoint ini tidak dapat diakses langsung dengan GET request (akan mengirimkan status 400).

    ![screenshot /upload](/assets/image-3.png)

- `/d/{ID}`: Melayani GET request terhadap file yang ingin didownload. 

    ![screenshot /d/](/assets/image-4.png)

### Proses Build Docker

Berikut adalah tahapan build dan run Docker yang saya lakukan sebelumnya:

1. Pembuatan Dockerfile dan .dockerignore

    Dockerfile berisi perintah-perintah yang dilakukan untuk membangun Docker image yang akan digunakan nanti saat Docker Compose. 
    
    Secara struktur, saya menerapkan multi-stage build sebagai berikut.
    ```dockerfile
    # Stage 1: Build
    FROM golang:1.26.2-alpine3.23 AS builder
    
    # ...

    # Stage 2: Compress
    FROM hatamiarash7/upx:latest AS compressor

    # ...

    # Stage 3: Final
    FROM alpine:3.23 

    # ...
    ```
    Saya menggunakan tiga image berbeda untuk melakukan proses building: 
    - `golang:1.26.2-alpine3.23` untuk mengkompilasi kode Go menjadi file binary.
    - `hatamiarash7/upx:latest` untuk melakukan kompresi terhadap file binary pada stage 1 sebelumnya. 
    - `alpine:3.23` sebagai runtime environment yang ringan.

    Untuk mempercepat kompilasi, saya menggunakan beberapa trik berikut:
    - Melakukan `go mod download` berdasarkan `go.mod` dan `go.sum` sebelum file-file lain.
    - Pemanfaatan caching untuk mempercepat build selanjutnya.
        ```dockerfile
        RUN --mount=type=cache,target=/go/pkg/mod \
            --mount=type=cache,target=/root/.cache/go-build \
        ``` 

    Beberapa upaya saya lakukan untuk memperkecil ukuran image:
    - Mematikan cross-compilation C dan menghapus informasi debugging.
        ```dockerfile
        RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .
        ```    
    - Kompresi menggunakan UPX.
        ```dockerfile
        RUN upx --best --lzma -o /workspace/main-compressed /workspace/main
        ```
    - Menambahkan `.dockerignore` untuk memperkecil ukuran build context secara signifikan, yang saya sesuaikan dengan workspace saya sebelumnya.

    Cara-cara di atas menghasilkan image seperti berikut:

    ![docker images](/assets/image-5.png)
    
    Di atas itu, saya menggunakan non-root privilege untuk menjalankan aplikasi sebagai bentuk upaya mengamankan service.

    ```dockerfile
    RUN addgroup -S appgroup && adduser -S appuser -G appgroup
    ```

2. Pembuatan docker-compose.yaml

    Docker Compose digunakan untuk melakukan container orchestration. Meskipun container yang dikelola hanya satu, tetap saja penggunaan Docker Compose ini memudahkan manajemen siklus hidup kontainer. Oleh karena itu, file konfigurasi yaml-nya cukup sederhana:
    ```yaml
    name: temp-file-sharing
    services:
    app:
        build: .
        container_name: temp-file-sharing
        ports:
        - "${HOST_PORT}:8080"
        volumes:
        - app-data:/app/data
        environment:
        - APP_URL=${APP_URL}
        - APP_PORT=8080
        - UPLOAD_DIR=/app/data/uploads
        - DB_PATH=/app/data/app.db
        restart: unless-stopped
        healthcheck:
        test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
        interval: 30s
        timeout: 10s
        retries: 3

    volumes:
    app-data:
    ```

### Proses Deployment 

Untuk deployment, saya menggunakan VM instance yang telah saya buat sebelumnya. Sebelum itu, harus dipastikan terlebih dahulu apakah Network Security Group untuk VM tersebut memiliki Inbound Port Rules HTTP (80).

![alt text](/assets/image-6.png)

Selanjutnya, saya melakukan SSH terhadap VM tersebut dan melakukan beberapa perintah berikut:

- `sudo apt upgrade && sudo apt update -y`
- Instalasi docker menggunakan script konvensional:
    ```bash
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    ```
- Setup docker rootless dan mengikuti arahannya:
    ```bash
    dockerd-rootless-setuptool.sh

    sudo apt install -y uidmap
    ```
- Melakukan cloning repository:
    ```bash
    git clone https://github.com/lazuardiyasfin/oprecncc2026/
    cd oprecncc2026
    git checkout modul1
    ```
- Buat .env baru:
    ```bash
    cp .env.example .env
    ```
- Terakhir, build image dan jalankan di belakang layar menggunakan Docker Compose:
    ```bash
    docker compose up -d --build
    ```