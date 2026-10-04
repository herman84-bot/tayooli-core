# Agent Development System

Dokumen ini mendefinisikan sistem kerja multi-agent untuk menjaga repository tetap clean, secure, maintainable, dan tidak bottleneck.

## Hierarchy

```text
User Prompt
   |
   v
1. CTO Agent
   - memahami kebutuhan user
   - membuat gagasan, scope, PRD mini, acceptance criteria
   - memecah pekerjaan menjadi task teknis yang aman dikerjakan
   |
   v
2. Senior Software Engineer Agent
   - menerjemahkan PRD menjadi implementasi
   - menjaga architecture, readability, typing, error handling, dan migration safety
   - membuat implementation report
   |
   v
3. Quality Assurance Agent
   - membuat dan menjalankan test plan
   - validasi workflow, regression risk, build/lint/test
   - jika fail, mengembalikan bug report ke Senior Software Engineer
   |
   v
4. Red Team Agent / Penetration Tester
   - threat modeling dan security review
   - cek auth, authorization, tenant isolation, input validation, secrets, injection, SSRF, XSS, CSRF
   - jika fail, mengembalikan security finding ke Senior Software Engineer
   |
   v
Ready for PR / Merge
```

## Core Rules

1. Setiap perubahan harus punya owner agent yang jelas.
2. Tidak boleh langsung coding tanpa PRD/acceptance criteria dari CTO Agent.
3. Tidak boleh dianggap selesai tanpa QA pass.
4. Tidak boleh merge jika Red Team menemukan finding severity `critical` atau `high`.
5. Jika QA atau Red Team fail, task kembali ke Senior Software Engineer sampai pass.
6. Agent tidak boleh menyimpan secret/token/API key di repo.
7. Perubahan database harus memikirkan rollback, migration order, dan tenant isolation.
8. Perubahan API harus menjaga backward compatibility atau menyediakan migration path.

## Definition of Done

Sebuah task dinyatakan selesai jika memenuhi semua poin berikut:

- CTO Agent sudah membuat PRD mini dan acceptance criteria.
- Senior Software Engineer sudah implementasi sesuai scope.
- Kode sudah diformat dan tidak memperkenalkan lint/build error.
- Unit/integration/e2e test yang relevan sudah dijalankan.
- QA Agent sudah menandai hasil `PASS`.
- Red Team Agent sudah menandai hasil `PASS` atau hanya menemukan low-risk accepted findings.
- PR checklist sudah lengkap.
- Tidak ada secret atau generated artifact yang ikut commit.

## Documents

- [Workflow](./workflow.md)
- [CTO Agent](./cto-agent.md)
- [Senior Software Engineer Agent](./senior-software-engineer-agent.md)
- [Quality Assurance Agent](./quality-assurance-agent.md)
- [Red Team Agent](./red-team-agent.md)
- [Quality Gates](./quality-gates.md)
- [Handoff Templates](./templates/)
