# Specification Index

Semua dokumen spesifikasi diurutkan berdasarkan status: DONE → TODO → REJECTED.

---

## ✅ DONE

| ID | Title | Status | Priority | Notes |
|----|-------|--------|----------|-------|
| [SPEC-01](done/SPEC_DONE_01_Foundation.md) | Foundation (Testability + Package Murni) | ✅ Done | Quick Win | LoadEnv(), pricing package, CapitalInUse() |
| [SPEC-02](done/SPEC_DONE_02_RulesSingleSource.md) | Rules Single Source (TP/CL/TSL Unified) | ✅ Done | Quick Win | internal/rules, Evaluate(), SuggestPlan() |
| [SPEC-03](done/SPEC_DONE_03_LoggerStatsFix.md) | Logger + CSV Migration + Statistik Fix | ✅ Done | Quick Win | CSV v3, MigrateTradeHistory(), /stats fix |
| [SPEC-04](done/SPEC_DONE_04_SmallFixes.md) | Small Fixes (Analyst Refactor, Pricing Package) | ✅ Done | Fill-In | analyst.go thin wrapper, signals package |
| [SPEC-05](done/SPEC_DONE_05_Backtester.md) | Backtester Filter Teknikal C-BoW v1.0 | ✅ Done | Major Project | internal/backtest, IS/OOS, ablation |
| [SPEC-06](done/SPEC_DONE_06_ShadowLog.md) | Shadow Log Sinyal & Hasil (Validasi AI) | ✅ Done | Quick Win | internal/shadow, cron 16:30+16:50, /shadow |
| [SPEC-07](done/SPEC_DONE_07_StrategyCandidates.md) | Uji Strategi Kandidat (Breakout20, PullbackMA20) | ✅ Done — GAGAL | 4/4 strategi tidak lolos, root cause: sinyal teknikal tidak prediktif |
| [SPEC-08.1](done/SPEC_DONE_08_1_MomentumWeekly.md) | Weekly Momentum Rotation | ✅ Done — LOLOS | 2266 trades, OOS exp=1.34%, WR=63.9%, DD=23.43%, beat baselines |
| [SPEC-09](done/SPEC_DONE_09_MinHoldingPeriod.md) | Min Holding Period Before Rebalance | ❌ Done — REJECTED | MinHold=2 reduced REBAL 53% but failed 4/5 criteria |

---

## 📝 TODO (Quick Wins — Phase 1 Optimization)

| ID | Title | Priority | Estimated Effort | Target Impact |
|----|-------|----------|------------------|---------------|
| [SPEC-10](SPEC_TODO_10_VolumeConfirmation.md) | Volume Confirmation Filter | 🚀 Quick Win | 3-5 jam | WR 63.9%→68%, CL 455→350 trades |
| [SPEC-11](SPEC_TODO_11_VolatilityFilter.md) | Intraday Volatility Filter | 🚀 Quick Win | 2-3 jam | Avg hold 2d→4-5d, reduce noise |

**Combined expected impact (S10+S11):**
- Expectancy: 1.34% → **~1.8-2.0%**
- Win rate: 63.9% → **~68-70%**
- Max DD: 23.43% → **~20-22%**
- Avg hold: 2 days → **4-5 days**

---

## 🏗️ FUTURE (Phase 2-3)

Belum ada SPEC, masih di backlog ideation:
- **Phase 2:** Sector rotation layer, dynamic position sizing
- **Phase 3:** Market regime detection (IHSG trend), defensive mode

---

## ⛔ REJECTED

| ID | Title | Reason |
|----|-------|--------|
| [SPEC-09](done/SPEC_DONE_09_MinHoldingPeriod.md) | Min Holding Period Before Rebalance | Failed 4/5 criteria. Constraint degraded edge—weekly momentum requires rapid rotation. |

---

## Notes

- Semua SPEC mengikuti format: Problem → Success Criteria → Technical Design → Checklist → Risk
- Kriteria lolos wajib measurable (K1-K7) dengan threshold eksplisit
- Setiap SPEC selesai dipindahkan ke `done/` dan diupdate di `CLAUDE.md`
- SPEC TODO diurutkan berdasarkan prioritas eksekusi (Quick Wins first)
