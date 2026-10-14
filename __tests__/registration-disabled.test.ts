/**
 * @jest-environment node
 */
import { NextRequest } from "next/server"
import { POST, GET, PUT, DELETE } from "@/app/api/v1/[...path]/route"
import * as dedicatedRoute from "@/app/api/v1/auth/register/route"

describe("Registration Disabled Verification", () => {
  it("rejects POST /api/v1/auth/register with 403 and REGISTRATION_DISABLED", async () => {
    const req = new NextRequest("http://localhost:3000/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify({
        full_name: "Attacker or New User",
        email: "newuser@example.com",
        password: "Password123!",
      }),
      headers: {
        "content-type": "application/json",
      },
    })

    const params = Promise.resolve({ path: ["auth", "register"] })
    const res = await POST(req, { params })
    expect(res.status).toBe(403)

    const data = await res.json()
    expect(data.error).toBeDefined()
    expect(data.error.code).toBe("REGISTRATION_DISABLED")
    expect(data.error.message).toMatch(/dinonaktifkan/i)
  })

  it("rejects GET /api/v1/auth/register with 403", async () => {
    const req = new NextRequest("http://localhost:3000/api/v1/auth/register", {
      method: "GET",
    })

    const params = Promise.resolve({ path: ["auth", "register"] })
    const res = await GET(req, { params })
    expect(res.status).toBe(403)

    const data = await res.json()
    expect(data.error?.code).toBe("REGISTRATION_DISABLED")
  })

  it("rejects trailing slash /auth/register/ with 403", async () => {
    const req = new NextRequest("http://localhost:3000/api/v1/auth/register/", {
      method: "POST",
      body: JSON.stringify({ email: "test@example.com" }),
    })

    const params = Promise.resolve({ path: ["auth", "register", ""] })
    const res = await POST(req, { params })
    expect(res.status).toBe(403)

    const data = await res.json()
    expect(data.error?.code).toBe("REGISTRATION_DISABLED")
  })

  it("dedicated route /api/v1/auth/register blocks GET, POST, PUT, DELETE with 403", async () => {
    const getRes = await dedicatedRoute.GET()
    expect(getRes.status).toBe(403)
    const getData = await getRes.json()
    expect(getData.error.code).toBe("REGISTRATION_DISABLED")

    const postRes = await dedicatedRoute.POST()
    expect(postRes.status).toBe(403)

    const putRes = await dedicatedRoute.PUT()
    expect(putRes.status).toBe(403)

    const deleteRes = await dedicatedRoute.DELETE()
    expect(deleteRes.status).toBe(403)
  })
})
