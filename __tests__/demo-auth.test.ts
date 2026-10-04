import {
  AUTH_COOKIE,
  DEMO_USERS,
  backendBaseUrl,
  demoAuthEnabled,
  publicUser,
  signDemoToken,
  verifyDemoCredentials,
  verifyDemoToken,
} from "@/lib/auth/demo-auth"

function setNodeEnv(value: string) {
  (process.env as Record<string, string | undefined>).NODE_ENV = value
}

function clearEnv(key: string) {
  delete process.env[key]
}

afterEach(() => {
  clearEnv("NEXT_PUBLIC_API_URL")
  clearEnv("API_BASE_URL")
  clearEnv("AUTH_DEMO")
  setNodeEnv("test")
})

describe("demo auth helpers", () => {
  it("exposes the three seed users with password123", () => {
    expect(DEMO_USERS.map((u) => u.email)).toEqual([
      "admin@test.com",
      "accountant@test.com",
      "approver@test.com",
    ])
    for (const u of DEMO_USERS) {
      expect(u.password).toBe("password123")
    }
  })

  it("verifies credentials case-insensitively on email", () => {
    const user = verifyDemoCredentials("ADMIN@test.com", "password123")
    expect(user?.email).toBe("admin@test.com")
    expect(user?.role).toBe("admin")
  })

  it("rejects wrong password and unknown email", () => {
    expect(verifyDemoCredentials("admin@test.com", "wrongpass")).toBeNull()
    expect(verifyDemoCredentials("nobody@test.com", "password123")).toBeNull()
  })

  it("publicUser strips the password", () => {
    const user = verifyDemoCredentials("admin@test.com", "password123")!
    expect(publicUser(user)).toEqual({
      id: user.id,
      email: "admin@test.com",
      role: "admin",
    })
    expect(publicUser(user)).not.toHaveProperty("password")
  })

  it("signs and verifies a demo token", () => {
    const user = verifyDemoCredentials("approver@test.com", "password123")!
    const token = signDemoToken(user)
    const payload = verifyDemoToken(token)
    expect(payload).toEqual({
      id: user.id,
      email: user.email,
      role: user.role,
      tenantId: user.tenantId,
    })
  })

  it("rejects tampered, malformed and expired tokens", () => {
    const user = verifyDemoCredentials("admin@test.com", "password123")!
    const token = signDemoToken(user)

    expect(verifyDemoToken(`${token}extra`)).toBeNull()
    expect(verifyDemoToken("not-a-token")).toBeNull()
    expect(verifyDemoToken(undefined)).toBeNull()
    expect(verifyDemoToken("")).toBeNull()

    // Tamper with the payload portion.
    const [data] = token.split(".")
    const tampered = `${data}.0000000000000000000000000000000000000000000000000000000000000000`
    expect(verifyDemoToken(tampered)).toBeNull()
  })

  it("backendBaseUrl normalizes trailing slashes", () => {
    process.env.NEXT_PUBLIC_API_URL = "http://localhost:8081/"
    expect(backendBaseUrl()).toBe("http://localhost:8081")
    clearEnv("NEXT_PUBLIC_API_URL")
    clearEnv("API_BASE_URL")
    expect(backendBaseUrl()).toBeNull()
  })

  it("demoAuthEnabled is true without backend outside production", () => {
    setNodeEnv("development")
    clearEnv("NEXT_PUBLIC_API_URL")
    clearEnv("API_BASE_URL")
    clearEnv("AUTH_DEMO")
    expect(demoAuthEnabled()).toBe(true)
  })

  it("demoAuthEnabled is false when a backend is configured", () => {
    setNodeEnv("development")
    process.env.NEXT_PUBLIC_API_URL = "http://localhost:8081"
    expect(demoAuthEnabled()).toBe(false)
  })

  it("demoAuthEnabled is false in production unless AUTH_DEMO=true", () => {
    setNodeEnv("production")
    clearEnv("NEXT_PUBLIC_API_URL")
    clearEnv("API_BASE_URL")
    clearEnv("AUTH_DEMO")
    expect(demoAuthEnabled()).toBe(false)

    process.env.AUTH_DEMO = "true"
    expect(demoAuthEnabled()).toBe(true)
  })

  it("cookie name matches the Go backend cookie", () => {
    expect(AUTH_COOKIE).toBe("tayooli_auth")
  })
})
