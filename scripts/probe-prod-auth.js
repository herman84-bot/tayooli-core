// Step 2-3: backend health, products, and auth probes against a deployment.
//
// Uses TAYOOLI_EMAIL / TAYOOLI_PASSWORD from the environment; pass
// TAYOOLI_REGISTER_EMAIL to additionally attempt a registration probe.
const { baseUrl, requireCredentials, jsonBody } = require('./_live-helpers')

const FE = baseUrl()
const BE = process.env.TAYOOLI_BACKEND_URL || 'https://tayooli-backend.zeabur.app'

async function call(url, opts = {}) {
  const r = await fetch(url, opts)
  const text = await r.text()
  return { status: r.status, text: text.slice(0, 200), cookies: r.headers.getSetCookie?.() ?? [] }
}

;(async () => {
  console.log('BE /health            ', JSON.stringify(await call(`${BE}/health`)))
  console.log('FE /api/v1/products   ', JSON.stringify(await call(`${FE}/api/v1/products`)))

  const registerEmail = process.env.TAYOOLI_REGISTER_EMAIL
  const registerName = process.env.TAYOOLI_REGISTER_NAME || 'QA User'
  if (registerEmail) {
    const { password } = requireCredentials()
    const reg = await call(
      `${FE}/api/v1/auth/register`,
      jsonBody({ email: registerEmail, password, name: registerName })
    )
    console.log(`register ${registerEmail}`, reg.status, reg.text)
  }

  const { email, password } = requireCredentials()
  const l = await call(`${FE}/api/v1/auth/login`, jsonBody({ email, password }))
  const cookie = l.cookies.map((c) => c.split(';')[0]).join('; ')
  // A real Go JWT has 3 dot-separated parts; a demo token has 2 (data.sig).
  const tok = (cookie.match(/tayooli_auth=([^;]+)/) || [])[1] || ''
  const kind = tok.split('.').length === 3 ? 'REAL Go JWT' : tok ? 'DEMO token' : 'no cookie'
  const me = await call(`${FE}/api/v1/auth/me`, { headers: { cookie } })
  const prods = await call(`${FE}/api/v1/products`, { headers: { cookie } })
  console.log(
    `login ${email.padEnd(20)} ${l.status} cookie=${kind} | me ${me.status} ${me.text.slice(0, 90)} | products ${prods.status}`
  )

  const ok = l.status === 200 && kind === 'REAL Go JWT' && prods.status === 200
  console.log(`\nRESULT: ${ok ? 'PASS' : 'FAIL'}`)
  process.exit(ok ? 0 : 1)
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
