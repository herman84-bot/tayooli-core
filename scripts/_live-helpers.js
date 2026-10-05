// Shared helpers for Tayooli live smoke scripts.
//
// Credentials are read from the environment so nothing sensitive is committed:
//   TAYOOLI_EMAIL     login email
//   TAYOOLI_PASSWORD  login password
// Optional:
//   TAYOOLI_BASE_URL  default https://tayooli.my.id
//   CDP_URL           Playwright CDP endpoint, default http://127.0.0.1:9100

const DEFAULT_BASE_URL = 'https://tayooli.my.id'
const DEFAULT_CDP_URL = 'http://127.0.0.1:9100'

function baseUrl() {
  return process.env.TAYOOLI_BASE_URL || DEFAULT_BASE_URL
}

function cdpUrl() {
  return process.env.CDP_URL || DEFAULT_CDP_URL
}

/** Read login credentials from env, failing loudly instead of falling back to a hardcoded secret. */
function requireCredentials() {
  const email = process.env.TAYOOLI_EMAIL
  const password = process.env.TAYOOLI_PASSWORD
  if (!email || !password) {
    throw new Error('Set TAYOOLI_EMAIL and TAYOOLI_PASSWORD before running this script.')
  }
  return { email, password }
}

function jsonBody(body) {
  return {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(body),
  }
}

/** Log in against the API and return a Cookie header value. */
async function loginCookie(fe = baseUrl()) {
  const { email, password } = requireCredentials()
  const res = await fetch(`${fe}/api/v1/auth/login`, jsonBody({ email, password }))
  if (!res.ok) throw new Error(`login failed: ${res.status} ${await res.text()}`)
  return (res.headers.getSetCookie() || []).map((c) => c.split(';')[0]).join('; ')
}

/** Header set for authenticated API calls. */
function authHeaders(fe = baseUrl()) {
  return loginCookie(fe).then((cookie) => ({ cookie, 'content-type': 'application/json' }))
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

module.exports = {
  DEFAULT_BASE_URL,
  DEFAULT_CDP_URL,
  baseUrl,
  cdpUrl,
  requireCredentials,
  jsonBody,
  loginCookie,
  authHeaders,
  sleep,
}
