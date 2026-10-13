# Compare Report — Strategi Kandidat

Mode: --years 5 --capital 1000000000 --max-positions 29 --slippage-pct 0.2 --max-hold 5 --seed 42

## Tabel Perbandingan

| Strategi | CL Mode | Trades (OOS) | Win Rate | Expectancy | PF (Rp) | PF (%%) | Max DD | Pers Full | Pers OOS | R1 | R2 | Status |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---|
| cbow-structural | structural | 1104 (332) | 27.1% | -1.30% | 0.18 | 0.24 | Rp 411268150 | 0% | 0% | ❌ | ❌ | TIDAK LOLOS: K2 (expectancy OOS=-1.30%) |
| cbow-fixed | fixed | 1098 (330) | 27.3% | -1.29% | 0.19 | 0.24 | Rp 406091761 | 0% | 0% | ❌ | ❌ | TIDAK LOLOS: K2 (expectancy OOS=-1.29%) |
| breakout20 | fixed | 733 (220) | 21.8% | -1.65% | 0.14 | 0.17 | Rp 346388258 | 0% | 0% | ❌ | ❌ | TIDAK LOLOS: K2 (expectancy OOS=-1.65%) |
| pullback20 | fixed | 752 (226) | 26.3% | -1.29% | 0.20 | 0.25 | Rp 299752288 | 0% | 0% | ❌ | ❌ | TIDAK LOLOS: K2 (expectancy OOS=-1.29%) |

## Kriteria Lolos

| # | Kriteria | Ambang |
|---|---:|---:|
| 1 | Jumlah trade | total ≥ 300 dan OOS ≥ 100 |
| 2 | Expectancy OOS (net) | > 0%% |
| 3 | Persentil vs acak | ≥ 95%% (full) **dan** ≥ 95%% (OOS) |
| 4 | Konsistensi per tahun | ≥ 2/3 tahun dengan ≥ 30 trade punya expectancy > 0 |
| 5 | R1 (slippage 0.4%%) | expectancy > 0 |
| 6 | R2 (tanpa 3 saham teratas) | expectancy > 0 |

## Per Strategi

### cbow-structural [structural]
**TIDAK LOLOS: K2 (expectancy OOS=-1.30%)**

- Total: 1104 trade | Win Rate: 27.1% | Expectancy: -1.30% | PF (Rp): 0.18 | PF (%): 0.24
- OOS: 332 trade | Expectancy OOS: -1.34%
- Persentil: full 0% | OOS 0%
- R1 (slippage 0.4%): false | R2 (konsentrasi): false
- Max Drawdown: Rp 411268150 | Avg Hold: 3.2 hari
- Total Return: -41.13% | Modal Akhir: Rp 588731850

| Tahun | Trades | Win Rate | Expectancy | PF (Rp) |
|---|---|---|---|---|
| 2022 | 241 | 25.3% | -1.49% | 0.15 |
| 2023 | 264 | 28.0% | -1.18% | 0.18 |
| 2024 | 241 | 29.0% | -1.07% | 0.23 |
| 2025 | 220 | 25.9% | -1.45% | 0.20 |
| 2026 | 138 | 26.8% | -1.39% | 0.17 |

### cbow-fixed [fixed]
**TIDAK LOLOS: K2 (expectancy OOS=-1.29%)**

- Total: 1098 trade | Win Rate: 27.3% | Expectancy: -1.29% | PF (Rp): 0.19 | PF (%): 0.24
- OOS: 330 trade | Expectancy OOS: -1.31%
- Persentil: full 0% | OOS 0%
- R1 (slippage 0.4%): false | R2 (konsentrasi): false
- Max Drawdown: Rp 406091761 | Avg Hold: 3.2 hari
- Total Return: -40.61% | Modal Akhir: Rp 593908239

| Tahun | Trades | Win Rate | Expectancy | PF (Rp) |
|---|---|---|---|---|
| 2022 | 240 | 26.2% | -1.42% | 0.16 |
| 2023 | 257 | 27.6% | -1.23% | 0.18 |
| 2024 | 241 | 28.2% | -1.12% | 0.22 |
| 2025 | 222 | 27.0% | -1.39% | 0.20 |
| 2026 | 138 | 27.5% | -1.35% | 0.16 |

### breakout20 [fixed]
**TIDAK LOLOS: K2 (expectancy OOS=-1.65%)**

- Total: 733 trade | Win Rate: 21.8% | Expectancy: -1.65% | PF (Rp): 0.14 | PF (%): 0.17
- OOS: 220 trade | Expectancy OOS: -1.97%
- Persentil: full 0% | OOS 0%
- R1 (slippage 0.4%): false | R2 (konsentrasi): false
- Max Drawdown: Rp 346388258 | Avg Hold: 2.1 hari
- Total Return: -34.64% | Modal Akhir: Rp 653611742

| Tahun | Trades | Win Rate | Expectancy | PF (Rp) |
|---|---|---|---|---|
| 2022 | 177 | 21.5% | -1.72% | 0.16 |
| 2023 | 115 | 19.1% | -1.59% | 0.14 |
| 2024 | 141 | 22.0% | -1.21% | 0.18 |
| 2025 | 172 | 23.8% | -1.95% | 0.10 |
| 2026 | 128 | 21.9% | -1.70% | 0.16 |

### pullback20 [fixed]
**TIDAK LOLOS: K2 (expectancy OOS=-1.29%)**

- Total: 752 trade | Win Rate: 26.3% | Expectancy: -1.29% | PF (Rp): 0.20 | PF (%): 0.25
- OOS: 226 trade | Expectancy OOS: -1.56%
- Persentil: full 0% | OOS 0%
- R1 (slippage 0.4%): false | R2 (konsentrasi): false
- Max Drawdown: Rp 299752288 | Avg Hold: 2.8 hari
- Total Return: -29.98% | Modal Akhir: Rp 700247712

| Tahun | Trades | Win Rate | Expectancy | PF (Rp) |
|---|---|---|---|---|
| 2022 | 211 | 27.5% | -1.37% | 0.18 |
| 2023 | 125 | 23.2% | -1.02% | 0.21 |
| 2024 | 155 | 27.7% | -1.07% | 0.25 |
| 2025 | 174 | 25.9% | -1.62% | 0.17 |
| 2026 | 87 | 26.4% | -1.22% | 0.18 |

**⚠️ Lolos backtest bukan jaminan.** Survivorship bias, tidak ada dividen, dan asumsi fill harga close. Wajib paper trading sebelum uang asli.

