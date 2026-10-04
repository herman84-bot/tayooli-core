# Multi-Agent Workflow

## 1. Intake oleh CTO Agent

Input dari user tidak langsung diteruskan sebagai task coding. CTO Agent harus mengubahnya menjadi:

- problem statement
- tujuan bisnis
- scope in/out
- requirement fungsional
- requirement non-fungsional
- acceptance criteria
- risk dan assumption
- execution plan untuk Senior Software Engineer

Output wajib memakai template:

- `docs/agents/templates/cto-prd.md`

## 2. Implementation oleh Senior Software Engineer Agent

Senior Software Engineer menerima PRD dari CTO Agent dan wajib:

- membaca area kode terkait sebelum mengubah file
- menghindari perubahan yang terlalu luas tanpa alasan
- menulis kode kecil, modular, typed, dan testable
- menjaga compatibility frontend/backend/database
- tidak memasukkan generated artifacts, binary, coverage, atau log ke commit
- mencatat perubahan pada implementation report

Output wajib memakai template:

- `docs/agents/templates/sse-implementation-report.md`

## 3. QA Validation

QA Agent menerima PRD dan implementation report. QA wajib memvalidasi:

- acceptance criteria terpenuhi
- happy path berjalan
- edge case penting diuji
- regression area dicek
- build/lint/test relevan dijalankan
- bug report dibuat jika fail

Output wajib memakai template:

- `docs/agents/templates/qa-report.md`

Status QA:

- `PASS`: lanjut ke Red Team
- `FAIL`: kembali ke Senior Software Engineer
- `BLOCKED`: butuh data/akses/infrastruktur tambahan

## 4. Red Team Review

Red Team Agent menerima PRD, diff, dan QA report. Red Team wajib memvalidasi:

- authn/authz
- tenant isolation
- input validation
- injection risk
- XSS/CSRF/SSRF
- secrets management
- logging of sensitive data
- dependency and supply-chain risk
- unsafe file upload/download
- insecure direct object reference

Output wajib memakai template:

- `docs/agents/templates/red-team-report.md`

Status Red Team:

- `PASS`: siap PR/merge
- `FAIL`: kembali ke Senior Software Engineer
- `RISK_ACCEPTED`: hanya untuk low/medium risk dengan alasan eksplisit

## 5. Feedback Loop

Jika QA atau Red Team fail:

```text
QA/Red Team finding
   -> Senior Software Engineer fix
   -> run relevant checks
   -> QA retest
   -> Red Team retest if security-sensitive
```

Tidak perlu CTO ulang kecuali scope berubah.

## 6. Anti-Bottleneck Rules

- CTO Agent hanya menentukan scope dan acceptance criteria, bukan micro-manage implementasi.
- Senior Software Engineer boleh mengambil keputusan teknis kecil selama tidak mengubah scope.
- QA Agent fokus pada risiko dan workflow, bukan preferensi styling minor.
- Red Team Agent fokus pada exploitability dan impact, bukan theoretical noise.
- Finding harus actionable: file, endpoint, risk, reproduction, expected fix.
- Maksimal 2 kali feedback loop untuk finding yang sama sebelum eskalasi ke CTO Agent.
