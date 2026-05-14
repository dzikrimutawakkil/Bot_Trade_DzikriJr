package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync" // 🔥 [FIX 1] Import sync untuk Mutex
	"time"

	"github.com/joho/godotenv"

	"learn-go/internal/models"
)

var (
	// Informasi Token & ID - HARUS dari environment variables
	BotToken     string
	MyChatID     int64
	GeminiAPIKey string
)

func init() {
	// Load .env dari root project (where main.go runs from)
	// Ini perlu karena init() di package config dijalankan SEBELUM main() di package main
	// godotenv.Load() di main.go sudah dipanggil, tapi init config jalan lebih dulu
	// Solusi: load .env dengan path absolut ke root project
	execPath, err := os.Executable()
	if err == nil {
		// Load dari direktori tempat executable berjalan (root project)
		projectRoot := filepath.Dir(execPath)
		godotenv.Load(filepath.Join(projectRoot, ".env"))
	}
	// Fallback: coba load dari current working directory
	godotenv.Load()

	// Load credentials dari environment variables
	BotToken = os.Getenv("BOT_TOKEN")
	MyChatIDStr := os.Getenv("MY_CHAT_ID")
	GeminiAPIKey = os.Getenv("GEMINI_API_KEY")

	// Validasi: Panic jika variabel kosong
	if BotToken == "" {
		panic("FATAL: BOT_TOKEN environment variable tidak ditemukan. Buat file .env dengan BOT_TOKEN=...")
	}
	if MyChatIDStr == "" {
		panic("FATAL: MY_CHAT_ID environment variable tidak ditemukan. Buat file .env dengan MY_CHAT_ID=...")
	}
	if GeminiAPIKey == "" {
		panic("FATAL: GEMINI_API_KEY environment variable tidak ditemukan. Buat file .env dengan GEMINI_API_KEY=...")
	}

	// Parse MyChatID dari string ke int64
	var parsedChatID int64
	_, parseErr := fmt.Sscanf(MyChatIDStr, "%d", &parsedChatID)
	if parseErr != nil {
		panic(fmt.Sprintf("FATAL: MY_CHAT_ID bukan angka valid: %s", MyChatIDStr))
	}
	MyChatID = parsedChatID
}

var (
	// Target & Batas Profit/Loss
	TPPercent = 0.07
	CLPercent = 0.03

	YahooTPTrigger = 4.0
	YahooCLTrigger = 1.5
	GoogleTPTarget = 5.0
	GoogleCLTarget = 2.0

	CheckPeriod    = 1 * time.Minute
	EmergencyDelay = 15 * time.Minute

	TrailingStopPercent = 0.025
	TSLActivationTrigger = 2.0

	// Pajak & Fee Broker (Bibit / Stockbit)
	BuyFee  = 0.0015 // 0.15% Beli
	SellFee = 0.0025 // 0.25% Jual

	// Manajemen Modal
	TotalModalTrading = 1000000.0 // Contoh: Rp 1.000.000
	MaxRiskPerTrade   = 0.05       // 5% dari total modal
	
	// [FIX Level 2] Menghindari Hardcode di main.go
	MinPurchaseAmount = 250000.0   // Minimal sisa modal untuk beli saham baru
)

var (
	// [FIX 1] Gembok Pengaman (Mutex) 🔒
	// Digunakan untuk mengunci MyStocks dan PendingOrders saat dibaca/ditulis
	// oleh proses background (monitor) dan proses manual (handler).
	DataMutex sync.RWMutex

	// Wadah penyimpan daftar saham
	PendingOrders = make(map[string]models.ActiveOrder)
	MyStocks      = make(map[string]models.TradingPlan)
	
	// Batas "Harga Kabur"
	RunawayPercent = 0.03 
)

// var Watchlist = []string{
// 	"ARCI", "BRMS", "MDKA", "ANTM",
// 	"ADRO", "PTBA", "HRUM", "INDY", "AADI", "DEWA",
// 	"MEDC", "ENRG", "ESSA", "ELSA",
// 	"MBMA", "INCO", "NCKL",
// 	"BRPT", "PGAS",
// 	"BULL", "TMAS", "SMDR", "HUMI",
// 	"PTPP", "PTRO",
// 	"RAJA",
// 	"MAPA", "CUAN",
// 	"PGEO",
// 	"EXCL",
// 	"BUMI",
// }

var Watchlist = []string{
	// Perbankan BUMN
	"BBRI", "BMRI", "BBNI",

	// Batu Bara & Energi
	"ADRO", "PTBA", "ITMG", "AADI", "ADMR", "ESSA",

	// Tambang Mineral
	"AMMN", "ANTM", "MDKA", "INCO", "BRMS",

	// Minyak & Gas
	"MEDC", "PGAS",

	// Industri Besar
	"ASII", "UNTR",

	// Agribisnis
	"JPFA", "CPIN", "AALI", "TAPG",

	// Ritel & Distribusi
	"MAPI", "AKRA", "ERAA",

	// Telekomunikasi
	"EXCL", "ISAT",

	// Infrastruktur Tower
	"TOWR",

	// Pulp & Paper
	"INKP",
}