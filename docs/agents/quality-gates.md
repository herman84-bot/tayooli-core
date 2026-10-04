# Quality Gates

Quality gates adalah checkpoint wajib agar development cepat tetapi tetap aman.

## Gate 0 — Scope Ready

Owner: CTO Agent

Required:

- Problem statement jelas.
- Scope in/out jelas.
- Acceptance criteria testable.
- Risk dan assumption tercatat.

Exit criteria:

- PRD mini selesai.

## Gate 1 — Implementation Ready

Owner: Senior Software Engineer Agent

Required:

- Kode sesuai PRD.
- Tidak ada generated artifacts/log/binary ikut perubahan.
- Checks relevan dijalankan.
- Implementation report selesai.

Exit criteria:

- Siap QA.

## Gate 2 — QA Pass

Owner: QA Agent

Required:

- Acceptance criteria tervalidasi.
- Test plan dijalankan.
- Build/lint/test relevan pass atau failure dijelaskan.
- Regression area dicek.

Exit criteria:

- QA status `PASS`.

## Gate 3 — Security Pass

Owner: Red Team Agent

Required:

- Threat model dilakukan.
- Auth/authz/tenant/input/secrets/injection dicek sesuai area perubahan.
- Tidak ada critical/high finding.

Exit criteria:

- Red Team status `PASS` atau `RISK_ACCEPTED` untuk risiko non-blocking.

## Gate 4 — Merge Ready

Owner: Maintainer / CTO Agent

Required:

- PR checklist lengkap.
- CI pass.
- No unresolved blocking comments.
- Commit bersih.

Exit criteria:

- Merge allowed.
