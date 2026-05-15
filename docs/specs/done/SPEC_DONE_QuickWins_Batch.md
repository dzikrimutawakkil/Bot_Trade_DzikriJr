# ACTIVE SPEC — DzikriJr Quick Wins Batch

## 📅 Spec Date: 2026-05-15
## 🎯 Scope: P1, P2, P3 (semua Quick Wins sekaligus)

---

## P1 — Race Condition Safety di `storage/persistance.go`

### Problem
`SaveData()` dan `LoadData()` mengakses file `stocks.json` tanpa proteksi `config.DataMutex`. Saat cron (15:40/16:20) dan command Telegram (/buy, /sell) berjalan paralel, concurrent write ke JSON bisa menyebabkan file corrupt atau data race.

### Acceptance Criteria
- [ ] `SaveData()` dibungkus `config.DataMutex.Lock()` sebelum write, `Unlock()` setelah selesai (defer)
- [ ] `LoadData()` dibungkus `config.DataMutex.RLock()` sebelum read, `RLock`/`RUnlock` setelah selesai
- [ ] Tidak ada goroutine lain yang akses `stocks.json` tanpa mutex — semua akses lewat satu pintu (`persistance.go`)
- [ ] Build passes: `go build ./...`

---

## P2 — Retry Logic dengan Exponential Backoff

### Problem
HTTP calls ke Yahoo Finance dan Gemini API tidak punya retry. Timeout di jam kritis = silent fail, user kehilangan rekomendasi.

### Scope
1. **`market/yahoo-finance-service.go`** — fungsi fetch OHLCV
2. **`research/research.go`** — Gemini API call di `GetDeepAnalysis()`

### Retry Strategy
- Max attempts: **3x**
- Backoff: **exponential** — attempt 1: 1s delay, attempt 2: 2s delay, attempt 3: 4s delay
- Gunakan `time.Sleep` + simple for-loop (bukan library baru — keep dependencies minimal)
- Return error hanya setelah semua attempt gagal
- Log setiap attempt failure ke console

### Acceptance Criteria
- [ ] Yahoo Finance fetch retry 3x dengan exponential backoff
- [ ] Gemini API call retry 3x dengan exponential backoff
- [ ] Console log jelas: attempt number + error message
- [ ] Build passes: `go build ./...`

---

## P3 — Gemini Error Notification ke User Telegram

### Problem
Jika `GetDeepAnalysis()` gagal (API error, timeout), aplikasi hanya log ke console. User tidak tahu rekomendasi gagal — modal idle tanpa alasan.

### Scope
- **`research/research.go`** — `GetDeepAnalysis()` call site

### Solution
Setelah semua retry gagal di P2, kirim notifikasi error ke user via `utils.SendSimpleMessage()` sebelum function return error.

### Message format:
```
⚠️ Gagal mengambil analisis untuk {symbol}
Coba lagi nanti atau hubungi admin.
```

### Acceptance Criteria
- [ ] Gemini error → user Telegram notified
- [ ] Message mengandung symbol yang gagal
- [ ] Build passes: `go build ./...`

---

## 📦 File yang Dimodifikasi
- `internal/storage/persistance.go` (P1)
- `internal/market/yahoo-finance-service.go` (P2)
- `internal/research/research.go` (P2 + P3)

## 🚫 Di luar scope
- Unit test (bukan Quick Win — masuk Major Project)
- Cache MA20/MA5 (Low Effort tapi Low Impact relatif terhadap 3 task di atas)
- Multiple chat ID support