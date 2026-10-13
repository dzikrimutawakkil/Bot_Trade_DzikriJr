# Spec: Shadow Log Sinyal & Hasil (Validasi Nilai Tambah AI) (v1.0)
Dibuat: 2026-10-12
Status: 🔲 TODO
Bergantung pada: Spec 05 (`internal/signals`), Spec 04 (`GetBars`, `GEMINI_MODEL`)

## Tujuan
Mengumpulkan dataset sinyal harian **tanpa uang**: untuk setiap saham watchlist yang lolos filter teknikal, catat verdict teknikal, rekomendasi Gemini, lalu isi otomatis hasil harga 1/3/5 hari kemudian. Dalam 1–2 bulan kamu bisa menjawab: *apakah filter AI menambah nilai dibanding teknikal saja?*

Hunter Algorithm yang ada (cron 15:40) berhenti setelah 3 saham "BELI" atau 10 panggilan AI, jadi datanya tidak lengkap dan bias. Shadow log berjalan **terpisah**, setelah market tutup, dan tidak mengubah perilaku Hunter.

## Perubahan

### 1. Package `internal/shadow`
File data: `data/signals_log.csv` (folder `data/` ada di `.gitignore`). Kolom:
```
date,symbol,score,code,market_state,close,signal_low,plan_cl,plan_tp1,ai_reco,ai_sentiment,ai_confidence,model,
ret_1d,ret_3d,ret_5d,max_up_5d,max_dd_5d,filled
```
- `date`: tanggal bar sinyal (WIB, `YYYY-MM-DD`). Kunci unik: `(date, symbol)` (tidak menulis ganda jika job berjalan dua kali).
- `code` dari `signals.Signal.Code`; `plan_cl`/`plan_tp1` dari `rules.Default`.
- `ai_reco`: `BELI | TAHAN | HINDARI | NA` (parse dari teks Gemini: "REKOMENDASI: BELI/TAHAN/HINDARI"); `ai_sentiment` memakai `extractSentimentScore`; tambah `extractConfidence` untuk "AI Confidence: NN%". Jika parsing gagal → `NA`, kolom lain kosong, jangan crash.
- `model`: `config.GeminiModel`.
- Kolom hasil kosong sampai `filled=true`. Definisi: entry = `close` hari sinyal; `ret_Nd = (close hari+N / close - 1)×100`; `max_up_5d` = (max High 5 hari berikutnya / close − 1)×100; `max_dd_5d` = (min Low 5 hari berikutnya / close − 1)×100. Tidak dikurangi fee (fee dikurangi saat ringkasan).

### 2. Cron (WIB, hanya hari bursa lewat `utils.IsTradingDay`)
- **16:30 `RecordSignals`:** ambil bar semua saham `config.Watchlist` + IHSG (`GetBars`, `CleanBars`), pastikan bar terakhir = hari ini (jika tidak, lewati dan log). `signals.Evaluate`; untuk `Score >= 7` (termasuk yang akan ditolak AI) panggil `research.GetDeepAnalysis` (berita + teknikal seperti Hunter), jeda 15 detik antar panggilan, maksimal 25 panggilan per hari; simpan baris. Saham dengan `Score < 7` **tidak** dicatat. Jalankan di goroutine sendiri; jangan memblokir cron lain.
- **16:50 `FillOutcomes`:** untuk baris `filled=false` yang sudah ≥ 5 hari bursa berlalu, ambil bar, isi kolom hasil, set `filled=true`. Jika data belum cukup, biarkan.
- Tulis file secara atomic (tmp + rename); akses file dilindungi mutex package.

### 3. Perintah `/shadow` (Telegram)
Ringkasan dari baris `filled=true` (hanya pemilik chat):
- Jumlah sinyal, rentang tanggal.
- Tabel per `ai_reco` (BELI/TAHAN/HINDARI) dan per `code`: `n`, win rate `ret_3d` setelah fee round-trip (`BuyFee + SellFee`), rata-rata `ret_3d`, rata-rata `max_dd_5d`.
- Satu baris perbandingan: rata-rata `ret_3d` untuk `ai_reco=BELI` vs `HINDARI` pada `code ∈ {HAMMER, GREEN_BOUNCE}`.
- Jika `n < 30` per grup, tampilkan peringatan "sampel kecil, jangan disimpulkan".
Logika ringkasan dipisah ke fungsi murni `Summarize(rows []Row) Summary` agar bisa dites.

## Test
- Parse `ai_reco`/sentiment/confidence dari teks contoh di prompt `analyst.go` (format emoji + Markdown) dan dari teks rusak (→ `NA`).
- `FillOutcomes` pada bar sintetis: close sinyal 1000; close hari+1/3/5 = 1010/1030/1020; High max 1050; Low min 985 → `ret_1d 1.0`, `ret_3d 3.0`, `ret_5d 2.0`, `max_up_5d 5.0`, `max_dd_5d -1.5`.
- Kunci unik: `RecordSignals` dua kali pada tanggal yang sama tidak menggandakan baris.
- `Summarize`: dataset kecil dengan hasil yang dihitung manual (win rate setelah fee 0.40%).

## Files
`internal/shadow/{shadow.go,parse.go,summary.go,*_test.go}`, `internal/telegram/handler.go` (`/shadow`), `cmd/bot/main.go` (cron), `internal/research/` (ekspor `extractSentimentScore` atau pindahkan ke `shadow/parse.go`), `.gitignore`.

## Checklist
- [x] `go build ./... && go vet ./... && go test ./...` bersih tanpa `.env`
- [x] Hunter 15:40 tidak berubah perilakunya
- [x] Job 16:30 dan 16:50 tidak berjalan di hari libur/akhir pekan
- [x] `/shadow` menampilkan peringatan sampel kecil bila perlu
- [x] Parse gagal tidak menghentikan job

## Di luar scope
Eksekusi otomatis, notifikasi per sinyal, dashboard.

## Catatan implementasi
- `internal/shadow/shadow.go`: Row struct + LoadRows/SaveRows (atomic tmp+rename) + FillRow + HasRow + AddOrUpdateRow + TradingDaysBetween
- `internal/shadow/parse.go`: ParseAIReco (regex REKOMENDASI), ParseSentiment (delegasi ke research.ExtractSentimentScore), ParseConfidence (regex AI Confidence)
- `internal/shadow/summary.go`: Summarize (pure function, GroupStats per (ai_reco, code), perbandingan BELI vs HINDARI untuk HAMMER+GREEN_BOUNCE, SmallSampleWarning jika n < 30)
- `internal/shadow/cron.go`: RecordSignals (16:30, Score >= 7, maks 25 AI calls, jeda 15 detik, goroutine) + FillOutcomes (16:50, >= 5 trading days, atomic write)
- `cmd/bot/main.go`: 2 cron entries baru "30 16 * * 1-5" dan "50 16 * * 1-5" dg IsTradingDay guard + goroutine
- `internal/telegram/handler.go`: case "/shadow" → ProcessShadowCommand → shadow.Summarize → Markdown
- `internal/research/handler_research.go`: extractSentimentScore → ExtractSentimentScore (exported)
- `data/` sudah ada di .gitignore
- Fee round-trip = 0.40% (0.15% + 0.25%)

