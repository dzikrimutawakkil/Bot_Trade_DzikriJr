# Backtest Report — MOMENTUM-WEEKLY (Weekly DCA)

## Parameter
| Parameter | Nilai |
|---|---|
| Strategy | momentum-weekly |
| Symbols | BBRI, BMRI, BBNI, ADRO, PTBA, ITMG, AADI, ADMR, ESSA, AMMN, ANTM, MDKA, INCO, BRMS, MEDC, PGAS, ASII, UNTR, JPFA, CPIN, AALI, TAPG, MAPI, AKRA, ERAA, EXCL, ISAT, TOWR, INKP |
| Capital Awal | Rp 1000000 |
| DCA Bulanan | Rp 1000000 |
| Max Positions | 4 |
| Slippage | 0.10% |
| CL (fixed) | -3.0% |
| TP | +5.0% |
| Max Hold (hari bursa) | 20 |
| Years | 5 |
| Seed | 42 |

**WARNING:** Yahoo Finance data tidak include dividen, tidak dilakukan survivorship-bias adjustment, dan asumsi fill di harga open. Gemini AI tidak terlibat dalam backtest ini.

- Rentang data: 2021-10-13 → 2026-10-13
- Total hari tercatat: 1198
- Total modal di-inject: Rp 62000000 (modal awal + 60 kali DCA Rp 1000000)

## Hasil Akhir
- Modal Akhir: **Rp 6124537138**
- Total Injected: Rp 61000000
- Profit (vs injected): **Rp 6063537138** (9778.29%)
- Max Drawdown: 23.43%

## Kriteria Lolos (K1–K7)
| # | Kriteria | Ambang | Hasil | Lolos? |
|---|---|---|---|---|
| K1 | Total trades | ≥ 100 | 2266 | ✅ |
| K2 | **Expectancy OOS net** | **> 0%** | **1.34%** | **✅** |
| K3 | Win rate OOS | > 45% | 63.9% | ✅ |
| K4 | R1 (slippage 0.3%%) | expectancy OOS > 0 | (lihat compare.md) | — |
| K5 | Max DD | < 50% | 23.43% | ✅ |
| K6 | Konsisten | ≥ 2/3 tahun OOS exp > 0 | 3/3 tahun | ✅ |
| K7 | Beat Baseline | > random3 AND > DCA naif | (lihat compare.md) | — |

## Metrik Lengkap
| Window | Trades | Win Rate | Expectancy | PF (Rp) | PF (%%) | Avg Hold |
|---|---:|---:|---:|---:|---:|---:|
| Full | 2266 | 62.8% | 1.30% | 2.41 | 2.41 | 1.6 |
| In-Sample (sebelum 2024-01-01) | 945 | 61.3% | 1.24% | 2.07 | 2.34 | 1.7 |
| OOS (2024-01-01+) | 1321 | 63.9% | 1.34% | 2.42 | 2.45 | 1.6 |

## Per Tahun (berdasarkan entry date)
| Tahun | Trades | Win Rate | Expectancy | PF (Rp) | Injected | Net PnL |
|---|---:|---:|---:|---:|---:|---:|
| 2021 | 14 | 50.0% | 0.13% | 0.96 | Rp 13000000 | Rp -8742 |
| 2022 | 528 | 61.7% | 1.43% | 2.14 | Rp 12000000 | Rp 34342490 |
| 2023 | 403 | 61.0% | 1.03% | 2.04 | Rp 12000000 | Rp 81878553 |
| 2024 | 482 | 64.3% | 1.21% | 2.10 | Rp 12000000 | Rp 377346684 |
| 2025 | 485 | 64.5% | 1.42% | 2.79 | Rp 12000000 | Rp 1813470218 |
| 2026 | 354 | 62.4% | 1.39% | 2.34 | Rp 12000000 | Rp 4516416683 |

## Exit Reasons
| Reason | Count | Avg PNL %% |
|---|---:|---:|
| CL | 455 | -3.43% |
| EOD_CLOSE | 4 | 0.59% |
| REBAL | 944 | 0.71% |
| TIME | 1 | 0.92% |
| TP | 771 | 5.00% |
| TREND_BROKEN | 91 | -0.33% |

