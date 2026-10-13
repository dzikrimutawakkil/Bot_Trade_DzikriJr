# Compare Report — Weekly Momentum vs Baselines

Mode: --strategy momentum-weekly --years 5 --capital 1000000 --dca-monthly 1000000 --max-positions 4 --slippage-pct 0.1 --seed 42

## Tabel Perbandingan

| Strategi | Final Equity | Injected | Profit | Return | Annualized | Max DD | Trades |
|---|---:|---:|---:|---:|---:|---:|---:|
## Tabel Perbandingan

Catatan: kolom *Equity* dan *Profit* dihitung dari rentang OOS saja (2024-01-01 → akhir backtest). *Return vs Injected* dan *Expectancy per Trade* membandingkan strategi dengan apples-to-apples.

| Strategi | Equity (OOS akhir) | Injected (OOS) | Profit (OOS) | Return vs Injected | Expectancy per Trade | Annualized | Max DD | Trades |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| **MOMENTUM-WEEKLY (OOS)** | Rp 6124537138 | Rp 61000000 | Rp 6063537138 | **9940.22%** | **1.34%** | 1954.59% | 23.43% | 1321 |
| RANDOM3 | Rp 11076465 | Rp 62000000 | Rp -50923535 | -82.13% | -0.41% | -16.42% | 32.76% | 4377 |
| DCA_NAIF | Rp 64948209 | Rp 62000000 | Rp 2948209 | 4.76% | 0.00% | 0.95% | 4.54% | 0 |
| BUY_HOLD | Rp 1163321 | Rp 1000000 | Rp 163321 | 16.33% | 0.00% | 3.26% | 18.95% | 0 |

## Kriteria Lolos (K1–K7)

| # | Kriteria | Ambang | Hasil |
|---|---|---|---|
| K1 | Total trades | ≥ 100 | 2266 → ✅ |
| K2 | **Expectancy OOS net** | **> 0%** | **1.34%** → ✅ |
| K3 | Win rate OOS | > 45% | 63.9% → ✅ |
| K4 | R1 (slippage 0.3%) | expectancy OOS > 0 | 1.04% → ✅ |
| K5 | Max DD | < 50% | 23.43% → ✅ |
| K6 | Konsisten | ≥ 2/3 tahun OOS exp > 0 | 3/3 tahun → ✅ |
| K7 | Beat baseline | > random3 AND > DCA naif | ✅ |
  - momentum exp=1.34% vs random3 exp=-0.41% → ✅
  - momentum return=9778.29% vs DCA naif return=4.76% → ✅

**⚠️ Catatan penting:** Backtest ini tidak include dividen, ada survivorship bias (watchlist diseleksi setelah tahu performa), dan asumsi fill di harga open. Hasil OOS positif di IDX dengan fee IDX masih harus divalidasi paper trading.
