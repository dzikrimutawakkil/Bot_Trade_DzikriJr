# Backtest Report — CBOW [structural]

## Parameter
| Parameter | Nilai |
|---|---|
| Strategy | cbow |
| CL Mode | structural |
| Symbols | BBRI, BMRI, BBNI, ADRO, PTBA, ITMG, AADI, ADMR, ESSA, AMMN, ANTM, MDKA, INCO, BRMS, MEDC, PGAS, ASII, UNTR, JPFA, CPIN, AALI, TAPG, MAPI, AKRA, ERAA, EXCL, ISAT, TOWR, INKP |
| Capital | Rp 1000000000 |
| Max Positions | 29 |
| Slippage | 0.20% |
| Max Hold Days | 5 |
| Min Score | 8 |
| Pump Filter | true |
| ARB Filter | true |
| Ignore Market | false |
| Seed | 42 |

**WARNING:** Yahoo Finance data is not dividend-adjusted. Watchlist selected AFTER knowing which stocks performed well (survivorship bias). Gemini AI is not part of this backtest.

## Hasil Akhir
- Modal Akhir: **Rp 700247712**
- Total Return: **-29.98%** (gross, posisi terbuka dihitung dengan close terakhir)

## Funnel Sinyal
| Code | Count | % |
|---|---|
| NONE | 30814 | 97.1% |
| PULLBACK_MA20 | 931 | 2.9% |

Total hari-saham: 31745 | Lolos filter IHSG: 17982

### Total
- Trade: 752 | Win Rate: 26.3% | Avg Win: Rp 368831 | Avg Loss: Rp 672889
- Expectancy: -1.29% | PF (Rp): 0.20 | PF (%): 0.25 | Avg Hold: 2.8 hari bursa
- Max Drawdown: Rp 299752288
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 284 trade, avg -3.37%
  - CL_GAP: 34 trade, avg -4.39%
  - TSL: 99 trade, avg 0.07%
  - TSL_GAP: 206 trade, avg 0.07%
  - TP1: 22 trade, avg 3.65%
  - TIME: 107 trade, avg 0.33%

### In-Sample
- Trade: 526 | Win Rate: 27.2% | Avg Win: Rp 391205 | Avg Loss: Rp 689865
- Expectancy: -1.17% | PF (Rp): 0.21 | PF (%): 0.29 | Avg Hold: 2.9 hari bursa
- Max Drawdown: Rp 208578952
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 183 trade, avg -3.37%
  - CL_GAP: 24 trade, avg -4.62%
  - TSL: 75 trade, avg -0.03%
  - TSL_GAP: 144 trade, avg 0.08%
  - TP1: 20 trade, avg 3.66%
  - TIME: 80 trade, avg 0.35%

### Out-of-Sample
- Trade: 226 | Win Rate: 24.3% | Avg Win: Rp 310657 | Avg Loss: Rp 634869
- Expectancy: -1.56% | PF (Rp): 0.16 | PF (%): 0.17 | Avg Hold: 2.4 hari bursa
- Max Drawdown: Rp 91476460
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 101 trade, avg -3.37%
  - CL_GAP: 10 trade, avg -3.83%
  - TSL: 24 trade, avg 0.37%
  - TSL_GAP: 62 trade, avg 0.05%
  - TP1: 2 trade, avg 3.58%
  - TIME: 27 trade, avg 0.27%

## Per Tahun
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) | PF (%) |
|---|---|---|---|---|---|
| 2022 | 211 | 27.5% | -1.37% | 0.18 | 0.24 |
| 2023 | 125 | 23.2% | -1.02% | 0.21 | 0.30 |
| 2024 | 155 | 27.7% | -1.07% | 0.25 | 0.32 |
| 2025 | 174 | 25.9% | -1.62% | 0.17 | 0.19 |
| 2026 | 87 | 26.4% | -1.22% | 0.18 | 0.23 |

## Baseline Acak
- Simulasi acak 200x, rata-rata expectancy: 0.20%
- Strategi expectancy: -1.29%
- Persentil strategi vs acak (full): **0%**

