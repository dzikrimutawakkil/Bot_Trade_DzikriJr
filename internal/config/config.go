package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"learn-go/internal/models"
	"learn-go/internal/rules"
)

var (
	BotToken     string
	MyChatID     int64
	GeminiAPIKey string
	GeminiModel  string
)

// LoadEnv memuat .env dan credential dari environment variables.
// Mengembalikan error yang menyebut SEMUA variabel yang hilang/tidak valid.
// Tidak melakukan panic.
func LoadEnv() error {
	execPath, err := os.Executable()
	if err == nil {
		projectRoot := filepath.Dir(execPath)
		godotenv.Load(filepath.Join(projectRoot, ".env"))
	}
	godotenv.Load()

	BotToken = os.Getenv("BOT_TOKEN")
	MyChatIDStr := os.Getenv("MY_CHAT_ID")
	GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	GeminiModel = os.Getenv("GEMINI_MODEL")
	if GeminiModel == "" {
		GeminiModel = "gemini-flash-latest"
	}

	var missing []string
	if BotToken == "" {
		missing = append(missing, "BOT_TOKEN")
	}
	if MyChatIDStr == "" {
		missing = append(missing, "MY_CHAT_ID")
	}
	if GeminiAPIKey == "" {
		missing = append(missing, "GEMINI_API_KEY")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing environment variables: %v", missing)
	}

	parsedChatID, parseErr := strconv.ParseInt(MyChatIDStr, 10, 64)
	if parseErr != nil {
		return fmt.Errorf("MY_CHAT_ID is not a valid number: %s", MyChatIDStr)
	}
	MyChatID = parsedChatID

	return nil
}

var (
	CheckPeriod    = 1 * time.Minute
	EmergencyDelay = 15 * time.Minute

	TotalModalTrading = 1000000.0
)

var (
	DataMutex sync.RWMutex

	PendingOrders = make(map[string]models.ActiveOrder)
	MyStocks      = make(map[string]models.TradingPlan)
)

// CapitalInUse menghitung total modal yang sedang terpakai di pasar.
func CapitalInUse() float64 {
	DataMutex.RLock()
	defer DataMutex.RUnlock()
	var total float64
	for _, plan := range MyStocks {
		total += plan.EntryPrice * float64(plan.Lots) * 100 * (1 + rules.Default.BuyFee)
	}
	return total
}

// RemainingCapital menghitung sisa modal yang belum terpakai.
func RemainingCapital() float64 {
	return TotalModalTrading - CapitalInUse()
}

var Watchlist = []string{
	"BBRI", "BMRI", "BBNI",
	"ADRO", "PTBA", "ITMG", "AADI", "ADMR", "ESSA",
	"AMMN", "ANTM", "MDKA", "INCO", "BRMS",
	"MEDC", "PGAS",
	"ASII", "UNTR",
	"JPFA", "CPIN", "AALI", "TAPG",
	"MAPI", "AKRA", "ERAA",
	"EXCL", "ISAT",
	"TOWR",
	"INKP",
}
