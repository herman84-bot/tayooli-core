// Waits for the DELETE /wms/transfers/{id} route to reach production, then
// cancels the abandoned probe draft so it stops showing up in the transfer list.
//
// A 405 means the new route has not been deployed yet; a 200 means the draft is
// now CANCELLED. Credentials come from TAYOOLI_EMAIL / TAYOOLI_PASSWORD.
const { baseUrl, requireCredentials, jsonBody } = require('./_live-helpers')

const fe = baseUrl()
const POLL_INTERVAL_MS = 20_000
const MAX_POLLS = 30

const transferId = process.argv[2]
if (!transferId) {
  console.error('Usage: node scripts/cancel-transfer-draft.js <transfer-id>')
  process.exit(2)
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

;(async () => {
  const { email, password } = requireCredentials()
  const login = await fetch(`${fe}/api/v1/auth/login`, jsonBody({ email, password }))
  if (!login.ok) throw new Error(`login failed: ${login.status} ${await login.text()}`)
  const cookie = (login.headers.getSetCookie() || [])
    .map((c) => c.split(';')[0])
    .join('; ')
  const headers = { cookie, 'content-type': 'application/json' }

  for (let i = 0; i < MAX_POLLS; i++) {
    const res = await fetch(`${fe}/api/v1/wms/transfers/${transferId}`, {
      method: 'DELETE',
      headers,
    })
    const body = await res.text()

    if (res.status === 200) {
      console.log('CANCELLED')
      console.log(body)
      process.exit(0)
    }

    if (res.status === 405) {
      console.log(`[${i}] route not deployed yet (405), waiting...`)
    } else if (res.status === 404) {
      console.log('transfer not found (already removed or wrong id)')
      process.exit(1)
    } else if (res.status === 409) {
      console.log('transfer is no longer DRAFT, cannot cancel:', body)
      process.exit(1)
    } else {
      console.log(`[${i}] unexpected status ${res.status}:`, body)
    }

    await sleep(POLL_INTERVAL_MS)
  }

  console.error('TIMEOUT: cancel route still unavailable')
  process.exit(1)
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
