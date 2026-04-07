package portfolio

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"learn-go/internal/config"
	"learn-go/internal/models"
	"learn-go/internal/storage"
	"learn-go/internal/utils"
)

// Fungsi untuk memproses perintah: /antre [KODE] [HARGA] [LOT]
func ProcessAntreCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) < 4 { // Sekarang butuh 4 argumen: /antre TOWR 482 10
		utils.SendSimpleMessage(bot, "❌ Format salah!\nGunakan: `/antre [KODE] [HARGA] [LOT]`\nContoh: `/antre TOWR 482 10`")
		return
	}

	symbol := strings.ToUpper(args[1])
	orderPrice, errPrice := strconv.ParseFloat(args[2], 64)
	if errPrice != nil {
		utils.SendSimpleMessage(bot, "❌ Harga harus berupa angka.")
		return
	}

	lots, errLot := strconv.Atoi(args[3]) // Mengambil parameter Lot
	if errLot != nil || lots <= 0 {
		utils.SendSimpleMessage(bot, "❌ Jumlah LOT harus berupa angka positif.")
		return
	}

	// Simpan ke memori bot
	config.PendingOrders[symbol] = models.ActiveOrder{
		Symbol:     symbol,
		OrderPrice: orderPrice,
		Lot:        lots, // Menyimpan jumlah lot
	}

	// WAJIB: Simpan ke file JSON agar antrean tidak hilang saat bot direstart
	storage.SaveData()

	msg := fmt.Sprintf("🎣 **Jaring Terpasang!**\n\n"+
		"Emiten: **%s**\n"+
		"Harga Antrean: `Rp. %.0f`\n"+
		"Jumlah: `%d Lot`\n\n"+
		"_Saya akan awasi. Kalau harga kabur, saya kasih tahu!_", symbol, orderPrice, lots)
	utils.SendMarkdownMessage(bot, msg)
}

// Fungsi untuk mencabut antrean dari pantauan bot
func ProcessCancelAntreCommand(bot *tgbotapi.BotAPI, args []string) {
	if len(args) < 2 {
		utils.SendSimpleMessage(bot, "❌ Format salah!\nGunakan: `/cancel_antre [KODE]`\nContoh: `/cancel_antre TOWR`")
		return
	}

	symbol := strings.ToUpper(args[1])
	if _, exists := config.PendingOrders[symbol]; exists {
		// Hapus dari memori
		delete(config.PendingOrders, symbol)
		
		// WAJIB: Simpan perubahan ke JSON agar benar-benar terhapus permanen
		storage.SaveData()

		utils.SendSimpleMessage(bot, fmt.Sprintf("🗑️ Pantauan antrean **%s** telah dicabut dari sistem.", symbol))
	} else {
		utils.SendSimpleMessage(bot, fmt.Sprintf("⚠️ Tidak ada antrean aktif untuk **%s**.", symbol))
	}
}