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
- Modal Akhir: **Rp 588731850**
- Total Return: **-41.13%** (gross, posisi terbuka dihitung dengan close terakhir)

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
- Trade: 1104 | Win Rate: 27.1% | Avg Win: Rp 310493 | Avg Loss: Rp 626218
- Expectancy: -1.30% | PF (Rp): 0.18 | PF (%): 0.24 | Avg Hold: 3.2 hari bursa
- Max Drawdown: Rp 411268150
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 422 trade, avg -3.24%
  - CL_GAP: 36 trade, avg -5.77%
  - TSL: 131 trade, avg 0.09%
  - TSL_GAP: 245 trade, avg 0.06%
  - TP1: 32 trade, avg 3.67%
  - TIME: 238 trade, avg -0.03%

### In-Sample
- Trade: 772 | Win Rate: 27.2% | Avg Win: Rp 329019 | Avg Loss: Rp 675425
- Expectancy: -1.29% | PF (Rp): 0.18 | PF (%): 0.25 | Avg Hold: 3.3 hari bursa
- Max Drawdown: Rp 312146736
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 297 trade, avg -3.19%
  - CL_GAP: 27 trade, avg -5.69%
  - TSL: 88 trade, avg 0.02%
  - TSL_GAP: 161 trade, avg 0.07%
  - TP1: 26 trade, avg 3.67%
  - TIME: 173 trade, avg -0.00%

### Out-of-Sample
- Trade: 332 | Win Rate: 26.8% | Avg Win: Rp 266781 | Avg Loss: Rp 512415
- Expectancy: -1.34% | PF (Rp): 0.19 | PF (%): 0.22 | Avg Hold: 3.0 hari bursa
- Max Drawdown: Rp 101292890
- Persentil vs Acak (full): 0% | OOS: 0%
- Avg PNL per Exit:
  - CL: 125 trade, avg -3.38%
  - CL_GAP: 9 trade, avg -6.00%
  - TSL: 43 trade, avg 0.23%
  - TSL_GAP: 84 trade, avg 0.05%
  - TP1: 6 trade, avg 3.64%
  - TIME: 65 trade, avg -0.09%

## Per Tahun
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) | PF (%) |
|---|---|---|---|---|---|
| 2022 | 241 | 25.3% | -1.49% | 0.15 | 0.21 |
| 2023 | 264 | 28.0% | -1.18% | 0.18 | 0.27 |
| 2024 | 241 | 29.0% | -1.07% | 0.23 | 0.30 |
| 2025 | 220 | 25.9% | -1.45% | 0.20 | 0.24 |
| 2026 | 138 | 26.8% | -1.39% | 0.17 | 0.18 |

## Baseline Acak
- Simulasi acak 200x, rata-rata expectancy: 0.20%
- Strategi expectancy: -1.30%
- Persentil strategi vs acak (full): **0%**

