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
- Modal Akhir: **Rp 653611742**
- Total Return: **-34.64%** (gross, posisi terbuka dihitung dengan close terakhir)

## Funnel Sinyal
| Code | Count | % |
|---|---|
| BREAKOUT20 | 758 | 2.4% |
| NONE | 30987 | 97.6% |

Total hari-saham: 31745 | Lolos filter IHSG: 17982

### Total
- Trade: 733 | Win Rate: 21.8% | Avg Win: Rp 367123 | Avg Loss: Rp 707029
- Expectancy: -1.65% | PF (Rp): 0.14 | PF (%): 0.17 | Avg Hold: 2.1 hari bursa
- Max Drawdown: Rp 346388258
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 341 trade, avg -3.38%
  - CL_GAP: 31 trade, avg -4.84%
  - TSL: 100 trade, avg 0.07%
  - TSL_GAP: 192 trade, avg 0.24%
  - TP1: 12 trade, avg 3.57%
  - TIME: 57 trade, avg -0.05%

### In-Sample
- Trade: 513 | Win Rate: 23.0% | Avg Win: Rp 389111 | Avg Loss: Rp 726847
- Expectancy: -1.51% | PF (Rp): 0.16 | PF (%): 0.19 | Avg Hold: 2.2 hari bursa
- Max Drawdown: Rp 241189443
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 227 trade, avg -3.39%
  - CL_GAP: 19 trade, avg -4.37%
  - TSL: 77 trade, avg 0.15%
  - TSL_GAP: 142 trade, avg 0.22%
  - TP1: 10 trade, avg 3.57%
  - TIME: 38 trade, avg -0.11%

### Out-of-Sample
- Trade: 220 | Win Rate: 19.1% | Avg Win: Rp 305344 | Avg Loss: Rp 663052
- Expectancy: -1.97% | PF (Rp): 0.11 | PF (%): 0.12 | Avg Hold: 1.9 hari bursa
- Max Drawdown: Rp 105198815
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 114 trade, avg -3.37%
  - CL_GAP: 12 trade, avg -5.58%
  - TSL: 23 trade, avg -0.21%
  - TSL_GAP: 50 trade, avg 0.29%
  - TP1: 2 trade, avg 3.61%
  - TIME: 19 trade, avg 0.07%

## Per Tahun
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) | PF (%) |
|---|---|---|---|---|---|
| 2022 | 177 | 21.5% | -1.72% | 0.16 | 0.17 |
| 2023 | 115 | 19.1% | -1.59% | 0.14 | 0.18 |
| 2024 | 141 | 22.0% | -1.21% | 0.18 | 0.24 |
| 2025 | 172 | 23.8% | -1.95% | 0.10 | 0.11 |
| 2026 | 128 | 21.9% | -1.70% | 0.16 | 0.18 |

## Baseline Acak
- Simulasi acak 200x, rata-rata expectancy: 0.20%
- Strategi expectancy: -1.65%
- Persentil strategi vs acak (full): **0%**

