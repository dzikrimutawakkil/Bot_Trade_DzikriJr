Tentu, ini format Markdown yang rapi dan profesional. Kamu bisa langsung *copy* dan *paste* blok di bawah ini ke dalam file `README.md` milikmu:

```markdown
## 📁 Struktur Direktori (Feature-Based Design)

Proyek ini menggunakan struktur berbasis fitur (Domain-Driven Design skala kecil) untuk menjaga kode tetap bersih, terorganisir, dan mudah di-*maintain*.

```text
Bot_Trade_DzikriJr/
├── cmd/
│   └── bot/
│       └── main.go                 
├── internal/
│   ├── config/                     
│   │   └── utils.go               
│   ├── market/                     
│   │   ├── scrapper.go             
│   │   └── service.go              
│   ├── models/                     
│   │   └── types.go                
│   ├── portfolio/                  
│   │   ├── evaluation.go           
│   │   └── monitor.go              
│   ├── research/                   
│   │   ├── analyst.go              
│   │   ├── research.go             
│   │   └── summary.go              
│   ├── storage/                    
│   │   └── persistance.go          
│   └── telegram/                   
│       ├── handler.go              
│       ├── handler_portfolio.go    
│       └── handler_research.go     
```

### 🏗️ Penjelasan Modul

* **`cmd/bot/` (Jantung Aplikasi)**
    * `main.go`: *Entry point* aplikasi. Hanya bertugas untuk inisialisasi Cron, Bot Telegram, dan menjalankan *server*.
* **`internal/config/` (Pengaturan Global)**
    * `utils.go`: Menyimpan variabel konstanta dan konfigurasi seperti `BotToken`, `GeminiAPIKey`, dan `MyChatID`.
* **`internal/telegram/` (Pintu Masuk User)**
    * `handler.go`: *Routing* utama untuk memproses pesan yang masuk.
    * `handler_portfolio.go`: Menangani perintah terkait portofolio pengguna (`/buy`, `/sell`).
    * `handler_research.go`: Menangani perintah terkait riset saham (`/research`).
* **`internal/portfolio/` (Dompet & Evaluasi)**
    * `evaluation.go`: Berisi logika untuk menghitung persentase untung/rugi (*floating profit/loss*) portofolio.
    * `monitor.go`: Sistem pemantau harga otomatis untuk fitur *Early Warning System* (EWS).
* **`internal/market/` (Pengambil Data Luar)**
    * `service.go`: Layanan untuk memanggil dan memproses data dari API Yahoo Finance.
    * `scrapper.go`: Logika untuk melakukan *scraping* harga dari Google Finance dan membaca RSS Google News.
* **`internal/research/` (Otak AI)**
    * `research.go`: Logika *screening* untuk mencari sinyal dan indikator teknikal.
    * `analyst.go`: Fungsi pemanggilan Gemini AI untuk mendapatkan *insight* rekomendasi.
    * `summary.go`: Berfungsi membuat ringkasan sentimen berita fundamental.
* **`internal/storage/` (Database Lokal)**
    * `persistance.go`: Menangani baca/tulis (*I/O*) data `myStocks` ke dalam file JSON lokal.
* **`internal/models/` (Bentuk Data)**
    * `types.go`: Pusat penyimpanan seluruh *Struct* (contoh: `PortfolioItem`, `HistoricalData`) agar terhindar dari *circular dependency* antar *package*.
```