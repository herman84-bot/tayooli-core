# Pull Request

## Summary

<!-- Jelaskan perubahan utama. -->

## Agent Flow Checklist

### 1. CTO Agent

- [ ] PRD mini dibuat
- [ ] Scope in/out jelas
- [ ] Acceptance criteria testable
- [ ] Risk dan assumption tercatat

### 2. Senior Software Engineer Agent

- [ ] Implementasi sesuai PRD
- [ ] Kode clean, typed, dan maintainable
- [ ] Tidak ada generated artifact, binary, coverage, log, atau `.env` ikut commit
- [ ] Implementation report dibuat

### 3. QA Agent

- [ ] Test plan dibuat
- [ ] Acceptance criteria divalidasi
- [ ] Regression area dicek
- [ ] QA report status PASS

### 4. Red Team Agent

- [ ] Auth/authz dicek jika relevan
- [ ] Tenant isolation dicek jika relevan
- [ ] Input validation dan injection risk dicek
- [ ] Secrets/logging dicek
- [ ] Red Team report status PASS/RISK_ACCEPTED

## Commands Run

```bash
# contoh:
# npm run lint
# npm run test -- --passWithNoTests
# npm run build
# cd backend/go-core && go test ./...
# git diff --check
```

## Screenshots / Evidence

<!-- Tambahkan jika UI berubah. -->

## Risk Notes

- Security risk:
- Regression risk:
- Migration risk:
- Rollback plan:
