// Final end-to-end check of the deployed cancel-transfer feature.
const { baseUrl, requireCredentials, jsonBody } = require('./_live-helpers')

const fe = baseUrl()

;(async () => {
  const { email, password } = requireCredentials()
  const login = await fetch(`${fe}/api/v1/auth/login`, jsonBody({ email, password }))
  if (!login.ok) throw new Error(`login failed: ${login.status} ${await login.text()}`)
  const cookie = (login.headers.getSetCookie() || [])
    .map((c) => c.split(';')[0])
    .join('; ')
  const headers = { cookie, 'content-type': 'application/json' }

  const list = await (await fetch(`${fe}/api/v1/wms/transfers`, { headers })).json()
  const transfers = Array.isArray(list) ? list : list.data || []
  console.log('statuses:', transfers.map((t) => `${t.transfer_number}=${t.status}`).join(', '))

  const drafts = transfers.filter((t) => t.status === 'DRAFT')
  const cancelled = transfers.filter((t) => t.status === 'CANCELLED')
  console.log(`DRAFT=${drafts.length} CANCELLED=${cancelled.length}`)

  // The cancelled transfer must be terminal: cancelling again is a 409, and it
  // can never be submitted for approval again.
  const target = cancelled[0]
  let terminalOk = true
  if (target) {
    const again = await fetch(`${fe}/api/v1/wms/transfers/${target.id}`, {
      method: 'DELETE',
      headers,
    })
    const againBody = await again.text()
    console.log('re-cancel ->', again.status, againBody)
    terminalOk = again.status === 409

    const submit = await fetch(`${fe}/api/v1/wms/transfers/${target.id}/submit`, {
      method: 'POST',
      headers,
    })
    console.log('submit cancelled ->', submit.status, (await submit.text()).slice(0, 140))
    terminalOk = terminalOk && submit.status >= 400
  }

  // Unknown id must be a clean 404, not a 500.
  const missing = await fetch(
    `${fe}/api/v1/wms/transfers/00000000-0000-0000-0000-000000000000`,
    { method: 'DELETE', headers }
  )
  console.log('cancel unknown id ->', missing.status, (await missing.text()).slice(0, 100))
  const notFoundOk = missing.status === 404

  const pass = drafts.length === 0 && cancelled.length > 0 && terminalOk && notFoundOk
  console.log(`\nRESULT: ${pass ? 'PASS' : 'FAIL'}`)
  process.exit(pass ? 0 : 1)
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
