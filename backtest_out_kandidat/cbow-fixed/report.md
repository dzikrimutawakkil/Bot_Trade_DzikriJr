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
- Modal Akhir: **Rp 593908239**
- Total Return: **-40.61%** (gross, posisi terbuka dihitung dengan close terakhir)

## Funnel Sinyal
| Code | Count | % |
|---|---|
| BLOCKED_ARB | 12 | 0.0% |
| BLOCKED_PUMP | 1916 | 6.0% |
| GREEN_BOUNCE | 1165 | 3.7% |
| HAMMER | 234 | 0.7% |
| NONE | 13764 | 43.3% |
| PISAU | 6518 | 20.5% |
| PUCUK | 2968 | 9.3% |
| SIAGA | 1184 | 3.7% |
| TANGGUNG | 4013 | 12.6% |

Total hari-saham: 31774 | Lolos filter IHSG: 18010

### Total
- Trade: 1098 | Win Rate: 27.3% | Avg Win: Rp 309087 | Avg Loss: Rp 625085
- Expectancy: -1.29% | PF (Rp): 0.19 | PF (%): 0.24 | Avg Hold: 3.2 hari bursa
- Max Drawdown: Rp 406091761
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 397 trade, avg -3.37%
  - CL_GAP: 36 trade, avg -5.77%
  - TSL: 137 trade, avg 0.06%
  - TSL_GAP: 243 trade, avg 0.10%
  - TP1: 31 trade, avg 3.66%
  - TIME: 254 trade, avg -0.07%

### In-Sample
- Trade: 768 | Win Rate: 27.3% | Avg Win: Rp 332798 | Avg Loss: Rp 677536
- Expectancy: -1.28% | PF (Rp): 0.18 | PF (%): 0.26 | Avg Hold: 3.3 hari bursa
- Max Drawdown: Rp 309629159
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 284 trade, avg -3.36%
  - CL_GAP: 23 trade, avg -5.86%
  - TSL: 91 trade, avg -0.01%
  - TSL_GAP: 160 trade, avg 0.09%
  - TP1: 26 trade, avg 3.67%
  - TIME: 184 trade, avg -0.02%

### Out-of-Sample
- Trade: 330 | Win Rate: 27.3% | Avg Win: Rp 253761 | Avg Loss: Rp 503135
- Expectancy: -1.31% | PF (Rp): 0.19 | PF (%): 0.21 | Avg Hold: 3.0 hari bursa
- Max Drawdown: Rp 97932926
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 113 trade, avg -3.38%
  - CL_GAP: 13 trade, avg -5.60%
  - TSL: 46 trade, avg 0.20%
  - TSL_GAP: 83 trade, avg 0.11%
  - TP1: 5 trade, avg 3.63%
  - TIME: 70 trade, avg -0.20%

## Per Tahun
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) | PF (%) |
|---|---|---|---|---|---|
| 2022 | 240 | 26.2% | -1.42% | 0.16 | 0.22 |
| 2023 | 257 | 27.6% | -1.23% | 0.18 | 0.26 |
| 2024 | 241 | 28.2% | -1.12% | 0.22 | 0.29 |
| 2025 | 222 | 27.0% | -1.39% | 0.20 | 0.25 |
| 2026 | 138 | 27.5% | -1.35% | 0.16 | 0.17 |

## Baseline Acak
- Simulasi acak 200x, rata-rata expectancy: 0.20%
- Strategi expectancy: -1.29%
- Persentil strategi vs acak (full): **0%**

