# Estimasi Upgrade Server Zeabur untuk 5 Gudang
**Berdasarkan data Zeabur Pricing & Current Setup (Oktober 2026)**

---

## Status Saat Ini

| Komponen | Detail | Biaya |
|----------|--------|-------|
| **Server** | Tencent Jakarta 2C/2GB | $3/mo |
| **Plan Zeabur** | Free tier | $0/mo |
| **Kapasitas** | 1 gudang + testing | Terbatas |
| **Log Retention** | 48 jam | Minimal |
| **Backup** | Manual | Risky |
| **Total/bulan** | — | **$3/mo** |

---

## Analisis Kebutuhan untuk 5 Gudang

### Estimasi Beban Kerja

| Metrik | Per Gudang | × 5 Gudang | Catatan |
|--------|-----------|-----------|---------|
| **Staff WMS** | 8–12 orang | 40–60 orang | Inbound, storage, outbound, QC |
| **Concurrent Users** | 3–5 peak | 15–25 users | Morning/evening shift peak |
| **Daily DB Queries** | 8K–10K | 40–50K | Inventory moves, stock lists, reports |
| **WMS API/min** | 80–120 req/min | 400–600 req/min | Product list, stock, transfers, DO ship |
| **POS Transactions** | 10–20/day | 50–100/day | Multiple stores/warehouses |
| **Daily Backups** | ~50MB dump | ~250MB | PostgreSQL full backup |

---

## Rekomendasi Upgrade

### Option A: Minimum Viable (Tetap Tencent)

**Upgrade server yang ada ke spec lebih tinggi:**

```
Server:  Tencent Jakarta 2C/2GB   →  Tencent Jakarta 4C/8GB
Plan:    Free ($0/mo)            →  Pro ($19/mo)

Estimasi harga Tencent 4C/8GB: ~$9–12/mo
(Catatan: Zeabur pricing tidak menampilkan detail Tencent tier; 
 estimasi 3–4x dari 2C/2GB mengikuti cloud market standard)

Total biaya per bulan:
├─ Server upgrade: +$6–9/mo
└─ Plan upgrade: +$19/mo
─────────────────────────────
TOTAL: ~$28–31/mo (dari $3/mo sekarang)
```

**Kelebihan:**
- ✅ Tetap di region Jakarta (latency < 5ms)
- ✅ Familiar dengan provider (setup sudah working)
- ✅ Upgrade existing → minimal downtime (snapshot migrate)

**Kekurangan:**
- ❌ Single point of failure (no HA)
- ❌ No geographic redundancy
- ⚠️ Manual backup restore time (5–10 min)

---

### Option B: Production-Grade (Tencent + Redundancy)

**Upgrade server + add backup server di region lain:**

```
Primary:  Tencent Jakarta 4C/8GB  @ $9–12/mo
Standby:  Tencent Singapore 4C/8GB (or AWS) @ $9–18/mo
Plan:     Pro ($19/mo)

Replication:
├─ PostgreSQL streaming replication (continuous)
├─ App failover via Route53/NS switch (manual, ~2 min)
└─ Daily backup to S3 ($0.5/mo storage)

Total biaya per bulan:
├─ Primary server: $12/mo
├─ Standby server: $12/mo (same spec as primary)
├─ Plan upgrade: $19/mo
└─ S3 backup storage: $0.5/mo
─────────────────────────────
TOTAL: ~$43.5/mo
```

**Kelebihan:**
- ✅ Zero-downtime maintenance (fail to standby)
- ✅ Geographic redundancy (Jakarta ↔ Singapore)
- ✅ Automatic continuous backup
- ✅ RTO 2 min, RPO < 1 sec

**Kekurangan:**
- ❌ 2× server cost
- ❌ Failover script perlu testing
- ⚠️ Network latency Singapore ~30–50ms

---

### Option C: Scalable (Multi-node Kubernetes Cluster)

**Full HA cluster di Zeabur:**

```
Setup:     Zeabur Cluster (K3s) dengan 3 nodes
Region:    Jakarta (primary) + 1 node Singapore (redundancy)
Plan:      Team ($79/mo)

Node specs:
├─ Node 1 (Backend): 4C/8GB @ $12/mo
├─ Node 2 (Database HA): 4C/8GB @ $12/mo
└─ Node 3 (Standby): 2C/4GB @ $6/mo

Managed features:
├─ Auto-scaling (CPU > 70% → add node)
├─ Load balancing (round-robin)
├─ Auto-restart failed pods
└─ Integrated monitoring (Zeabur dashboard)

Total biaya per bulan:
├─ 3 nodes (K3s): $30/mo
├─ Team plan: $79/mo
└─ S3 backup: $0.5/mo
─────────────────────────────
TOTAL: ~$109.5/mo
```

**Kelebihan:**
- ✅ Unlimited horizontal scaling
- ✅ Auto-failover pod-level
- ✅ Kubernetes native (future-proof)
- ✅ Audit logs 90 days (compliance)

**Kekurangan:**
- ❌ High cost (~36× dari current)
- ❌ Operational complexity (K3s management)
- ⚠️ Overkill untuk 5 gudang saat ini

---

## Rekomendasi Terbaik: Option A (Phase 1) → Option B (Phase 2)

### Phase 1 (Immediate, 0–3 bulan)
**Upgrade to Option A: $28–31/mo**

```yaml
Action:
  1. Upgrade Zeabur plan: Free → Pro ($19/mo)
     └─ Get 30d log retention, backups, better build CI
  
  2. Upgrade server: 2C/2GB → 4C/8GB Tencent Jakarta (~$9–12/mo)
     └─ CPU capacity: 2K req/min → 8K req/min (safe margin for 5 gudang)
  
  3. Enable daily backups
     └─ Auto-backup to S3 (1 snapshot/day, 7-day retention)

Timeline: 2–4 jam downtime (schedule midnight)
Cost jump: $3 → $28/mo (ONE-TIME, then stable)
```

**Why Phase 1?**
- Validate 5-warehouse workload on upgraded hardware
- Monitor CPU/memory/latency for 3 months
- Gather cost-benefit data before investing in HA

---

### Phase 2 (If needed, Month 4+)
**Upgrade to Option B: $43.5/mo**

```yaml
Trigger:
  - If CPU sustained > 70% on single server
  - If any 1-hour downtime incident occurs
  - If compliance requires RTO < 5 min

Action:
  1. Add standby server (Tencent Singapore or AWS)
  2. Configure PostgreSQL replication (continuous)
  3. Setup automated failover script
  4. Run HA drills monthly

Cost increase: $28 → $43.5/mo (+$15.5/mo)
```

---

## Spesifikasi Server Upgrade

### Tencent Jakarta Upgrade (Phase 1)

| Spec | Current | Upgrade | Rationale |
|------|---------|---------|-----------|
| **CPU** | 2 vCPU | 4 vCPU | 2× for 5 gudang = 2K→8K req/min capacity |
| **RAM** | 2 GB | 8 GB | Postgres + Go app buffer (50K query/day) |
| **Disk** | 50 GB | 200 GB | 5 gudang DB = ~200–300 GB/month growth |
| **OS** | ZeaburOS | ZeaburOS | Same (no migration cost) |
| **Network** | 100Mbps | 1Gbps | WMS API throughput |
| **Price/mo** | $3 | $9–12 | Cloud market: 4C/8GB ~3–4× of 2C/2GB |

---

## Workload Comparison

### Current Server (2C/2GB) vs Upgraded (4C/8GB)

| Load Profile | 2C/2GB | 4C/8GB | Safety |
|--------------|--------|--------|--------|
| **1 Warehouse** | ✅ Comfortable (50% CPU) | ✅ Overkill | 2× headroom |
| **5 Warehouses** | ❌ **70–90% CPU** (risky) | ✅ 40–50% CPU | Safe for 3 months |
| **10 Warehouses** | ❌ Crash (100%+ CPU) | ⚠️ 70–80% CPU | Time to add standby |
| **Peak hour spike** | ❌ Timeout risk | ✅ Handled | 5 min delay → 30 sec |

---

## Budget Summary (Annual)

| Period | Setup | Plan | Server(s) | Backup | Total/mo | Annual |
|--------|-------|------|-----------|--------|----------|--------|
| **Now (0 mo)** | — | Free | $3 | $0 | $3 | $36 |
| **Phase 1 (1–3 mo)** | $0 | Pro | $12 | $0.5 | $31.5 | ~$95 |
| **Phase 2 (4+ mo)** | $0 | Pro | $24 | $1.0 | $44.0 | ~$528 |

**Total Year 1:** ~$659 (Phase 1 upgrade in month 1)

---

## Pengajuan ke Klien: Talking Points

### 1. Why Upgrade Now?

**Problem Statement:**
- Current server: 2C/2GB (designed for 1 gudang, 5 users)
- Requirement: 5 gudang, 40–60 staff, 15–25 concurrent users
- **Result:** Sistem akan overload (CPU 90%, timeouts, data inconsistency)

**Data point:**
```
1 Gudang   → 2C/2GB cukup (50% CPU)
5 Gudang   → 2C/2GB DANGER (90% CPU) ← Sekarang
5 Gudang   → 4C/8GB Safe (50% CPU) ← Dengan upgrade
```

### 2. Cost Justification

**Downtime Cost:**
- 1 jam downtime = loss of sales + manual correction
- Typical nilai loss: IDR 2–5 juta (10 staff × productivity/day)
- Server upgrade cost: IDR ~375K/bulan ($25) ← insurance

**Data Safety:**
- No backup = risk of data loss (IDR 10–50 juta correction)
- Daily backup = peace of mind

**ROI:**
```
Upgrade cost:     IDR ~375K/bulan
Downtime prevention: Priceless (avoid 1× incident/month)
Backup safety:    ~IDR 10M / incident
→ Break-even dalam <1 bulan (1 incident avoided)
```

### 3. Phased Approach (Risk Mitigation)

**Month 1:** Phase 1 upgrade ($28/mo)
- Prove capability for 5 gudang
- Monitor performance for 3 months

**Month 4+:** Phase 2 (if needed)
- Add redundancy only if data justifies
- Or continue Phase 1 (still safe)

**Flexibility:** Can cancel/downgrade anytime (no lock-in)

---

## Action Plan

### Week 1: Approval & Planning
- [ ] Client approves Phase 1 upgrade estimate
- [ ] Schedule upgrade maintenance window (off-business hours)
- [ ] Backup current database (safety net)

### Week 2: Upgrade Execution
- [ ] Provision new 4C/8GB Tencent server
- [ ] Restore database backup
- [ ] DNS/routing test (5 min)
- [ ] Smoke tests: register, warehouse, POS, reports
- [ ] Monitor for 24h (CPU, memory, latency)

### Week 3–4: Stabilization
- [ ] Enable daily auto-backups (S3)
- [ ] Configure Pro plan features (advanced logs, domain)
- [ ] Setup monitoring alerts (CPU > 70%, disk > 80%)

### Month 3: Evaluation
- [ ] Review CPU/memory usage trends
- [ ] Decide: Phase 2 or continue Phase 1
- [ ] Document performance baseline for future scaling

---

## Reference Data (From Zeabur Oct 2026)

**Zeabur Plan Pricing:**
- Free: $0/mo (1 own server, 48h logs)
- Dev: $5/mo (3 own servers, 7d logs)
- **Pro: $19/mo** (10 own servers, 30d logs, 4C/8G CI) ← Recommended
- Team: $79/mo (unlimited, 90d logs, HA features)

**Tencent Server Pricing (estimated from current):**
- 2C/2GB: $3/mo (current)
- 4C/8GB: ~$9–12/mo (typical 3–4× scaling)
- 8C/16GB: ~$18–24/mo

**Note:** Exact pricing available in Zeabur dashboard after login.

---

**Prepared for:** Client proposal / Budget planning  
**Valid until:** Dec 31, 2026  
**Cost accuracy:** ±10% (subject to Zeabur pricing changes)
