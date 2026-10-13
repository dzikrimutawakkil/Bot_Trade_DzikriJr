# Backtest Report — MOMENTUM-WEEKLY (Weekly DCA)

## Parameter
| Parameter | Nilai |
|---|---|
| Strategy | momentum-weekly |
| Symbols | BBRI, BMRI, BBNI, ADRO, PTBA, ITMG, AADI, ADMR, ESSA, AMMN, ANTM, MDKA, INCO, BRMS, MEDC, PGAS, ASII, UNTR, JPFA, CPIN, AALI, TAPG, MAPI, AKRA, ERAA, EXCL, ISAT, TOWR, INKP |
| Capital Awal | Rp 1000000 |
| DCA Bulanan | Rp 0 |
| Max Positions | 3 |
| Slippage | 0.20% |
| CL (fixed) | -3.0% |
| TP | +5.0% |
| Max Hold (hari bursa) | 20 |
| Years | 5 |
| Seed | 42 |

**WARNING:** Yahoo Finance data tidak include dividen, tidak dilakukan survivorship-bias adjustment, dan asumsi fill di harga open. Gemini AI tidak terlibat dalam backtest ini.

- Rentang data: 2021-10-13 → 2026-10-13
- Total hari tercatat: 1198
- Total modal di-inject: Rp 1000000 (modal awal + 60 kali DCA Rp 0)

## Hasil Akhir
- Modal Akhir: **Rp 162838646**
- Total Injected: Rp 1000000
- Profit (vs injected): **Rp 161838646** (16183.86%)
- Max Drawdown: 26.18%

## Kriteria Lolos (K1–K7)
| # | Kriteria | Ambang | Hasil | Lolos? |
|---|---|---|---|---|
| K1 | Total trades | ≥ 100 | 1566 | ✅ |
| K2 | **Expectancy OOS net** | **> 0%** | **1.19%** | **✅** |
| K3 | Win rate OOS | > 45% | 58.7% | ✅ |
| K4 | R1 (slippage 0.3%%) | expectancy OOS > 0 | (lihat compare.md) | — |
| K5 | Max DD | < 50% | 26.18% | ✅ |
| K6 | Konsisten | ≥ 2/3 tahun OOS exp > 0 | 3/3 tahun | ✅ |
| K7 | Beat Baseline | > random3 AND > DCA naif | (lihat compare.md) | — |

## Metrik Lengkap
| Window | Trades | Win Rate | Expectancy | PF (Rp) | PF (%%) | Avg Hold |
|---|---:|---:|---:|---:|---:|---:|
| Full | 1566 | 58.6% | 1.18% | 2.11 | 2.03 | 1.7 |
| In-Sample (sebelum 2024-01-01) | 650 | 58.5% | 1.18% | 2.02 | 2.07 | 1.7 |
| OOS (2024-01-01+) | 916 | 58.7% | 1.19% | 2.11 | 2.01 | 1.7 |

## Per Tahun (berdasarkan entry date)
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) | Injected | Net PnL |
|---|---:|---:|---:|---:|---:|---:|
| 2021 | 10 | 60.0% | 0.48% | 1.27 | Rp 1000000 | Rp 12808 |
| 2022 | 344 | 57.6% | 1.28% | 1.96 | Rp 0 | Rp 2555258 |
| 2023 | 296 | 59.5% | 1.09% | 2.06 | Rp 0 | Rp 5307043 |
| 2024 | 322 | 57.5% | 1.02% | 1.69 | Rp 0 | Rp 13390034 |
| 2025 | 353 | 58.6% | 1.19% | 2.16 | Rp 0 | Rp 48315931 |
| 2026 | 241 | 60.6% | 1.41% | 2.18 | Rp 0 | Rp 114047856 |

## Exit Reasons
| Reason | Count | Avg PNL %% |
|---|---:|---:|
| CL | 444 | -3.44% |
| EOD_CLOSE | 3 | 0.71% |
| REBAL | 441 | 0.62% |
| TIME | 1 | 0.59% |
| TP | 623 | 5.04% |
| TREND_BROKEN | 54 | -0.62% |

