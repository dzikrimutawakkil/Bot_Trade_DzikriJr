# Spec: Perbaikan Kecil & Ketahanan (v1.0)
Dibuat: 2026-10-12
Status: ✅ DONE
Bergantung pada: Spec 01 ✅, Spec 02 ✅

Setiap item berdiri sendiri. Kerjakan berurutan F1 → F8; commit per item.

## F1. Penyimpanan data aman — `internal/storage/persistance.go`
- `SaveData`: ganti `Lock()` dengan `RLock()` (hanya membaca map). Tulis ke `stocks.json.tmp` lalu `os.Rename` ke `stocks.json` (atomic). Ganti `ioutil.*` dengan `os.ReadFile/WriteFile`.
- `LoadData`: jika file ada tetapi JSON gagal diparse (dan format lama juga gagal), **jangan** lanjut dengan data kosong secara diam-diam. Salin file ke `stocks.json.corrupt-<unix>`, log error jelas, lalu mulai kosong.
- Test: save → load round-trip; file korup → `.corrupt-*` terbentuk.

## F2. `/reset` butuh konfirmasi — `handler_portfolio.go`, `telegram/handler.go`
- `/reset` saja → balas "Ketik `/reset confirm` untuk menghapus SEMUA pantauan (posisi + antrean)".
- `/reset confirm` → kosongkan `MyStocks` **dan** `PendingOrders`, `SaveData()`.
- Routing di `HandleMessages`: `/reset` kini masuk ke switch perintah berargumen.

## F3. Filter ANTI-ARB & teks — `internal/research/analyst.go` (`GetStockScore`)
- Filter 1 memakai perubahan **close-ke-close** kemarin: `(yestClose - prevClose)/prevClose*100 <= -10` dengan `prevClose = cleanCloses[len-3]` (panjang data sudah dijamin ≥ 20). Konstanta `antiARBDropPct = 10.0` di bagian atas file.
- Ganti teks verdict Filter 1 menjadi persis:
  `⛔ **ANTI-ARB — saham turun lebih dari 10% kemarin**\nAlasan: Saham ini baru saja jatuh tajam kemarin. Risiko lanjut turun tinggi walaupun ada pantulan hari ini. Jangan disentuh!`
- Ganti teks verdict Filter 2 menjadi persis:
  `⛔ **ANTI-DISTRIBUTION TRAP — naik %.2f%% dari low hari ini**\nAlasan: Harga sudah ditarik naik lebih dari 3.5%% dari titik terendah hari ini. Risiko distribusi dan koreksi tajam tinggi. Jangan entry!`
- Hapus semua karakter non-Indonesia/non-Inggris yang nyasar dan singkatan yang salah ("Airborne Rescue").
- Konstanta `pumpFromLowPct = 3.5`. **Jangan ubah nilai**: Spec 05 akan mengukur efeknya.

## F4. HTTP & Gemini — `internal/market/*.go`, `internal/research/*.go`
- Satu `var httpClient = &http.Client{Timeout: 15 * time.Second}` di package `market`, dipakai `GetLivePrice` dan `GetHistoricalPrices`. `FetchTechnicalData` memakai client yang sama (lihat F5).
- `GetLivePrice`: guard `len(result.Indicators.Quote) == 0` sebelum `Quote[0]` (sekarang bisa panic).
- `GetHistoricalPrices`: tutup `resp.Body` secara eksplisit di setiap iterasi retry (hapus `defer` di dalam loop).
- `GetDeepAnalysis` dan `getPortfolioEvaluation`: `defer cancel()` / `defer client.Close()` jangan di dalam loop retry; bungkus satu percobaan dalam fungsi/closure sehingga `defer` berjalan per percobaan.

## F5. Satu pipeline data teknikal
- `models.Bar{Time time.Time; Open, High, Low, Close, Volume float64}`.
- Tambah `Timestamp []int64 \`json:"timestamp"\`` pada `YahooChartResponse.Result[0]`.
- `market.GetHistoricalPrices(symbol)` tetap ada; tambah `market.GetBars(symbol, yahooRange string) ([]models.Bar, error)` (range mis. "3mo", "5y") dan `market.CleanBars(raw []models.Bar) []models.Bar` (buang bar dengan Open/High/Low/Close/Volume ≤ 0).
- `GetStockScore` dan `FetchTechnicalData` sama-sama memakai `GetBars` + `CleanBars` (hapus fetch Yahoo sendiri di `FetchTechnicalData`, dan hapus duplikasi filter). **Format teks laporan `FetchTechnicalData` tidak boleh berubah.**
- Tambah test `CleanBars` (bar dengan Volume 0 atau Close nil/0 dibuang).

## F6. Hari libur bursa — `internal/utils/utils.go`
- File opsional `holidays.json`: array string `"YYYY-MM-DD"` (zona WIB). Dimuat saat start (`utils.LoadHolidays(path)`); jika file tidak ada, perilaku sama seperti sekarang.
- `IsMarketOpen()` mengembalikan false pada hari libur. Tambah `IsTradingDay(t time.Time) bool` (bukan Sabtu-Minggu/libur) dan bungkus semua cron di `main.go` (`ProcessPortfolioEvaluation`, Hunter) dengan pemeriksaan itu.
- Sertakan `holidays.example.json` berisi 2 tanggal contoh.
- Test: Sabtu → false; tanggal di daftar → false; hari kerja biasa → true.

## F7. Model Gemini dapat dikonfigurasi
- `config.GeminiModel`: dari env `GEMINI_MODEL`, default `"gemini-flash-latest"`. Dibaca di `LoadEnv`. Dipakai di `analyst.go` dan `evaluation.go` (hapus string literal).
- Log nama model pada setiap panggilan (`[AI] model=%s ...`). Tambahkan `GEMINI_MODEL=` ke `.env.example` (opsional/boleh kosong).

## F8. `main.go`
- Periksa error `c.AddFunc(...)` (log fatal bila gagal).
- Hapus komentar/variabel mati. Jangan mengubah jadwal.

## Test & kualitas
- `go build ./... && go vet ./... && go test ./...` bersih tanpa `.env`.
- Tidak ada `defer` di dalam loop retry (`grep` manual).

## Files
`internal/storage/persistance.go`, `internal/portfolio/handler_portfolio.go`, `internal/telegram/handler.go`, `internal/research/{analyst,research,handler_research}.go`, `internal/market/*.go`, `internal/models/types.go`, `internal/utils/utils.go`, `internal/config/config.go`, `portfolio/evaluation.go`, `cmd/bot/main.go`, `holidays.example.json`, `.env.example`.

## Checklist
- [x] F1 atomic save + penanganan file korup
- [x] F2 `/reset confirm`
- [x] F3 ANTI-ARB close-ke-close + teks bersih (tanpa karakter Mandarin)
- [x] F4 timeout HTTP, guard `Quote[0]`, tidak ada `defer` dalam loop
- [x] F5 `GetBars/CleanBars` dipakai oleh `GetStockScore` dan `FetchTechnicalData`
- [x] F6 hari libur
- [x] F7 `GEMINI_MODEL`
- [x] F8 error cron dicek
- [x] Build/vet/test bersih

## Di luar scope
Mengubah ambang filter (3.5%, 10%), migrasi SDK Gemini, antrean/averaging otomatis.

## Catatan implementasi
F1: `SaveData` now uses RLock+atomic write via `.tmp`+rename. `LoadData` copies corrupt files to `.corrupt-<unix>`.
F2: `/reset` → asks confirm; `/reset confirm` clears both `MyStocks` AND `PendingOrders`.
F3: ANTI-ARB uses `clean[len-3].Close` as prevClose (close-to-close). Verdicts cleaned.
F4: Single `httpClient` in `market` package. `Quote[0]` guard added. All `resp.Body.Close()` are explicit (no defer in loop).
F5: `GetBars`+`CleanBars` unified pipeline; `GetStockScore` and `FetchTechnicalData` both use it.
F6: `LoadHolidays()` + `IsTradingDay()` + cron jobs wrapped with trading-day check.
F7: `config.GeminiModel` from `GEMINI_MODEL` env, default `gemini-flash-latest`. Used in `analyst.go` and `evaluation.go`.
F8: All `c.AddFunc` errors now cause `log.Fatalf`.
