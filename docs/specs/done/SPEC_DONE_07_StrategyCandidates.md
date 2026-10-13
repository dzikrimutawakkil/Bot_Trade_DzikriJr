# Spec: Uji Strategi Kandidat — Breakout 20 Hari & Pullback MA20 (v1.0)
Dibuat: 2026-10-13
Status: ✅ CLOSED — Tidak Satupun Strategi Lolos

## Hasil Backtest (Mode: --years 5 --capital 1000000000 --max-positions 29 --slippage-pct 0.2 --max-hold 5 --seed 42)

### Tabel Perbandingan

| Strategi | CL Mode | Trades (OOS) | Win Rate | Expectancy | PF (Rp) | PF (%) | Max DD | Status |
|---|---|---:|---:|---:|---:|---:|---:|---|
| cbow-structural | structural | 1104 (332) | 27.1% | -1.30% | 0.18 | 0.24 | Rp 411.268.150 | TIDAK LOLOS |
| cbow-fixed | fixed | 1098 (330) | 27.3% | -1.29% | 0.19 | 0.24 | Rp 406.091.761 | TIDAK LOLOS |
| breakout20 | fixed | 733 (220) | 21.8% | -1.65% | 0.14 | 0.17 | Rp 346.388.258 | TIDAK LOLOS |
| pullback20 | fixed | 752 (226) | 26.3% | -1.29% | 0.20 | 0.25 | Rp 299.752.288 | TIDAK LOLOS |

### Kriteria Lolos

| # | Kriteria | Ambang | Hasil |
|---|---:|---:|---|
| 1 | Jumlah trade | total ≥ 300 dan OOS ≥ 100 | ✅ Semua lolos (733-1104 trade) |
| 2 | Expectancy OOS | > 0% | ❌ Semua gagal (-1.29% s/d -1.65%) |
| 3 | Persentil vs acak | ≥ 95% (full) dan ≥ 95% (OOS) | ❌ Semua 0% |
| 4 | Konsistensi per tahun | ≥ 2/3 tahun dengan expectancy > 0 | ❌ Semua 0/5 tahun |
| 5 | R1 (slippage 0.4%) | expectancy > 0 | ❌ Semua gagal |
| 6 | R2 (tanpa 3 saham teratas) | expectancy > 0 | ❌ Semua gagal |

### Breakdown Per Tahun

#### cbow-fixed (kontrol)
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) |
|---|---|---|---|---|
| 2022 | 240 | 26.2% | -1.42% | 0.16 |
| 2023 | 257 | 27.6% | -1.23% | 0.18 |
| 2024 | 241 | 28.2% | -1.12% | 0.22 |
| 2025 | 222 | 27.0% | -1.39% | 0.20 |
| 2026 | 138 | 27.5% | -1.35% | 0.16 |

#### breakout20
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) |
|---|---|---|---|---|
| 2022 | 177 | 21.5% | -1.72% | 0.16 |
| 2023 | 115 | 19.1% | -1.59% | 0.14 |
| 2024 | 141 | 22.0% | -1.21% | 0.18 |
| 2025 | 172 | 23.8% | -1.95% | 0.10 |
| 2026 | 128 | 21.9% | -1.70% | 0.16 |

#### pullback20
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) |
|---|---|---|---|---|
| 2022 | 211 | 27.5% | -1.37% | 0.18 |
| 2023 | 125 | 23.2% | -1.02% | 0.21 |
| 2024 | 155 | 27.7% | -1.07% | 0.25 |
| 2025 | 174 | 25.9% | -1.62% | 0.17 |
| 2026 | 87 | 26.4% | -1.22% | 0.18 |

## Kesimpulan

**Tidak satupun strategi lolos K2 (expectancy OOS > 0%).**

### Key Findings
1. **Sinyal teknis tidak prediktif** — 4 strategi berbeda (C-BoW, breakout, pullback, kontrol) semua menghasilkan expectancy negatif dan persentil 0% vs random entry.
2. **Random entry lebih baik** — baseline acak rata-rata expectancy +0.19% (vs -1.3% sinyal).
3. **Distribusi loss/win ratio terlalu buruk** — PF (Rp) hanya 0.14-0.20, artinya rata-rata loss ~5-7x win.
4. **Konsisten tiap tahun** — negatif di SEMUA 5 tahun, bukan unlucky period.

### Root Cause Analysis
- Konsep "beli pantulan" / breakout / pullback tidak memiliki edge di pasar Indonesia/Jakarta IDX
- Tidak ada evidence bahwa indikator teknikal per saham (MA positioning, volume spike) prediktif dalam konteks ini
- Biaya transaksi (fee + slippage) menggerus edge yang tidak ada

### Lessons Learned
- Scientific integrity: spec pra-registrasi + kriteria lolos sebelum melihat hasil menghasilkan keputusan clean
- Survivorship bias: data tidak include delisted stocks, real performance likely worse
- Tidak ada dividen dalam backtest, real return would be higher in practice

⚠️ **Lolos backtest bukan jaminan.** Survivorship bias, tidak ada dividen, dan asumsi fill harga close. Wajib paper trading sebelum uang asli.

## Output Files
- `backtest_out_kandidat/compare.md` — laporan perbandingan lengkap
- `backtest_out_kandidat/cbow-structural/` — trades.csv + report.md
- `backtest_out_kandidat/cbow-fixed/` — trades.csv + report.md
- `backtest_out_kandidat/breakout20/` — trades.csv + report.md
- `backtest_out_kandidat/pullback20/` — trades.csv + report.md

## Checklist
- [x] `go build ./... && go vet ./... && go test ./...` bersih tanpa `.env`
- [x] `go run ./cmd/backtest --compare --years 5 --capital 1000000000 --max-positions 29 --out D:\Dzikri\learn-go\backtest_out_kandidat` menghasilkan `compare.md` dan folder per strategi
- [x] Parameter strategi sesuai spec (tidak ada angka yang diubah)
- [x] Baseline acak memakai CL yang sama dengan strategi
- [x] Kriteria lolos tercetak per strategi di `compare.md`
- [x] PF (%) benar; tidak ada catatan ablation basi
