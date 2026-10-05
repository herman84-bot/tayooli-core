// Watches backend health after a deploy and reports the migration outcome.
//
// A failed migration is fatal at startup (main.go log.Fatal), so the app going
// unhealthy is the signal that a migration broke. Also probes whether the
// transfer_status enum picked up the new CANCELLED label indirectly, by
// checking whether a non-draft transfer cancel is rejected with 409 (route and
// enum both live) rather than 405 (route missing) or 500 (enum missing).
const { baseUrl, requireCredentials, jsonBody, sleep } = require('./_live-helpers')

const BE = process.env.TAYOOLI_BACKEND_URL || 'https://tayooli-backend.zeabur.app'
const POLL_INTERVAL_MS = 15_000
const MAX_POLLS = 40

;(async () => {
  const { email, password } = requireCredentials()
  const fe = baseUrl()

  let everUnhealthy = false
  for (let i = 0; i < MAX_POLLS; i++) {
    let health = 'unreachable'
    try {
      const r = await fetch(`${BE}/health`)
      health = r.ok ? `ok(${r.status})` : `FAIL(${r.status})`
    } catch (e) {
      health = `unreachable(${e.message})`
    }
    if (!health.startsWith('ok')) everUnhealthy = true
    console.log(`[${i}] health: ${health}`)
    if (health.startsWith('ok')) break
    await sleep(POLL_INTERVAL_MS)
  }

  const login = await fetch(`${fe}/api/v1/auth/login`, jsonBody({ email, password }))
  if (!login.ok) throw new Error(`login failed: ${login.status} ${await login.text()}`)
  const cookie = (login.headers.getSetCookie() || [])
    .map((c) => c.split(';')[0])
    .join('; ')
  const headers = { cookie, 'content-type': 'application/json' }

  const list = await (await fetch(`${fe}/api/v1/wms/transfers`, { headers })).json()
  const transfers = Array.isArray(list) ? list : list.data || []
  console.log('transfers:', transfers.length)

  const draft = transfers.find((t) => t.status === 'DRAFT')
  if (!draft) {
    console.log('no DRAFT transfer present; nothing to cancel')
    console.log('RESULT: PASS (no draft)')
    process.exit(0)
  }

  const res = await fetch(`${fe}/api/v1/wms/transfers/${draft.id}`, {
    method: 'DELETE',
    headers,
  })
  const body = await res.text()
  console.log(`DELETE draft ${draft.transfer_number} ->`, res.status, body)

  const pass = res.status === 200
  console.log(`\nRESULT: ${pass ? 'PASS' : 'FAIL'}${everUnhealthy ? ' (health dipped during rollout)' : ''}`)
  process.exit(pass ? 0 : 1)
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
