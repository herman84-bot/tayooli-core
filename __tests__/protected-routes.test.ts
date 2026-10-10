import * as fs from "fs"
import * as path from "path"
import { isProtectedPath, PROTECTED_PREFIXES } from "@/lib/auth/protected-routes"

describe("protected routes guard", () => {
  it("protects every authenticated module and its sub-paths", () => {
    for (const p of ["/dashboard", "/pos", "/wms", "/wms/transfers", "/products/123", "/settings"]) {
      expect(isProtectedPath(p)).toBe(true)
    }
  })

  it("leaves public pages open", () => {
    for (const p of ["/", "/login", "/register", "/forgot-password", "/reset-password", "/verify-email", "/api/v1/auth/login"]) {
      expect(isProtectedPath(p)).toBe(false)
    }
  })

  it("does not match look-alike prefixes", () => {
    expect(isProtectedPath("/poss")).toBe(false)
    expect(isProtectedPath("/dashboarding")).toBe(false)
  })

  it("covers every route folder under app/(app)", () => {
    const dir = path.join(__dirname, "..", "app", "(app)")
    const folders = fs.readdirSync(dir, { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => "/" + d.name)
    for (const f of folders) expect(isProtectedPath(f)).toBe(true)
  })

  it("middleware matcher lists every protected prefix", () => {
    const src = fs.readFileSync(path.join(__dirname, "..", "middleware.ts"), "utf8")
    for (const p of PROTECTED_PREFIXES) expect(src).toContain(`"${p}/:path*"`)
  })
})
