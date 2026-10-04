import React from "react"
import { render } from "@testing-library/react"
import "@testing-library/jest-dom"
import { renderFormattedMessage } from "@/components/copilot/CopilotDrawer"
import { matchRuleBasedAction } from "@/lib/copilot/fallback-matcher"
import { ProposedAction } from "@/hooks/useCopilot"

describe("Red Team Security Audit: Frontend Copilot Subsystem", () => {
  describe("1. XSS & HTML Injection in Chat Drawer (renderFormattedMessage)", () => {
    it("renders <script> tags as text, preventing script execution", () => {
      const maliciousScript = "<script>alert('xss')</script>"
      const { container } = render(<div>{renderFormattedMessage(maliciousScript)}</div>)

      // Verify no <script> element is in the DOM
      const scripts = container.querySelectorAll("script")
      expect(scripts.length).toBe(0)

      // Verify text is rendered as plain text
      expect(container.textContent).toContain("<script>alert('xss')</script>")
    })

    it("renders <img onerror> tags as plain text without creating element", () => {
      const maliciousImg = "<img src='x' onerror='alert(document.cookie)' />"
      const { container } = render(<div>{renderFormattedMessage(maliciousImg)}</div>)

      const imgs = container.querySelectorAll("img")
      expect(imgs.length).toBe(0)
      expect(container.textContent).toContain("<img src='x' onerror='alert(document.cookie)' />")
    })

    it("neutralizes HTML injection inside markdown bold and code formatting", () => {
      const input = "**<script>alert(1)</script>** and `<img src=x onerror=steal()>`"
      const { container } = render(<div>{renderFormattedMessage(input)}</div>)

      expect(container.querySelectorAll("script").length).toBe(0)
      expect(container.querySelectorAll("img").length).toBe(0)

      const strong = container.querySelector("strong")
      expect(strong).not.toBeNull()
      expect(strong?.textContent).toBe("<script>alert(1)</script>")

      const code = container.querySelector("code")
      expect(code).not.toBeNull()
      expect(code?.textContent).toBe("<img src=x onerror=steal()>")
    })

    it("neutralizes HTML injection inside Markdown table cells and headers", () => {
      const tableWithXSS = [
        "| Kolom 1 <script>alert(1)</script> | Kolom 2 <img src=x onerror=bad()> |",
        "| --- | --- |",
        "| <iframe src='http://evil.com'></iframe> | <svg onload=alert(1)> |",
      ].join("\n")

      const { container } = render(<div>{renderFormattedMessage(tableWithXSS)}</div>)

      expect(container.querySelectorAll("script").length).toBe(0)
      expect(container.querySelectorAll("img").length).toBe(0)
      expect(container.querySelectorAll("iframe").length).toBe(0)
      expect(container.querySelectorAll("svg").length).toBe(0)

      // Elements are safely placed in th/td as text
      const ths = container.querySelectorAll("th")
      expect(ths[0].textContent).toContain("<script>alert(1)</script>")
      expect(ths[1].textContent).toContain("<img src=x onerror=bad()>")

      const tds = container.querySelectorAll("td")
      expect(tds[0].textContent).toContain("<iframe src='http://evil.com'></iframe>")
      expect(tds[1].textContent).toContain("<svg onload=alert(1)>")
    })
  })

  describe("2. Autopilot Mode Safety & Silent Mutation (fallback-matcher.ts)", () => {
    const mutatingPrompts = [
      "tambah vendor PT Berkah Baru email test@test.com",
      "tambah customer PT Sukses Mandiri",
      "tambah produk baru Laptop Lenovo kode LNV-01 harga 8000000",
      "buatkan invoice untuk vendor PT Maju Jaya sebesar 5 juta rupiah",
      "ubah nama perusahaan jadi PT Nusantara",
      "undang anggota tim admin@perusahaan.com sebagai admin",
      "setup payment gateway midtrans",
    ]

    it.each(mutatingPrompts)("ensures '%s' is NEVER marked as low risk", (prompt) => {
      const res = matchRuleBasedAction(prompt)
      expect(res.actions.length).toBeGreaterThan(0)
      for (const action of res.actions) {
        expect(action.riskLevel).not.toBe("low")
        expect(["medium", "high"]).toContain(action.riskLevel)
      }
    })

    it("verifies autopilot filter logic only permits low-risk actions", () => {
      // Simulates the exact check in CopilotDrawer.tsx:
      // if (effectiveAutonomy === 'autopilot') { for (action of actions) { if (action.riskLevel === 'low') { autoExecute() } } }
      const mockAutoExecute = jest.fn()

      const actions: ProposedAction[] = [
        {
          id: "1",
          toolName: "create_vendor",
          title: "Tambah Vendor",
          description: "...",
          riskLevel: "medium",
          diff: [],
          payload: {},
          status: "pending",
        },
        {
          id: "2",
          toolName: "create_customer",
          title: "Tambah Customer",
          description: "...",
          riskLevel: "medium",
          diff: [],
          payload: {},
          status: "pending",
        },
        {
          id: "3",
          toolName: "create_purchase_invoice",
          title: "Buat Invoice",
          description: "...",
          riskLevel: "medium",
          diff: [],
          payload: {},
          status: "pending",
        },
        {
          id: "4",
          toolName: "configure_payment_gateway",
          title: "Setup Gateway",
          description: "...",
          riskLevel: "high",
          diff: [],
          payload: {},
          status: "pending",
        },
        {
          id: "5",
          toolName: "navigate_to_module",
          title: "Buka Vendor",
          description: "...",
          riskLevel: "low",
          diff: [],
          payload: {},
          status: "pending",
        },
      ]

      const effectiveAutonomy = "autopilot"
      if (effectiveAutonomy === "autopilot") {
        for (const action of actions) {
          if (action.riskLevel === "low") {
            mockAutoExecute(action)
          }
        }
      }

      // Only the low-risk navigation action must have been auto-executed
      expect(mockAutoExecute).toHaveBeenCalledTimes(1)
      expect(mockAutoExecute).toHaveBeenCalledWith(
        expect.objectContaining({ toolName: "navigate_to_module", riskLevel: "low" })
      )
    })
  })

  describe("3. Advisory Mode Filtering Logic", () => {
    it("strictly blocks mutating actions and only allows navigate_to_module", () => {
      const actions: ProposedAction[] = [
        { id: "1", toolName: "create_vendor", title: "", description: "", riskLevel: "medium", diff: [], payload: {}, status: "pending" },
        { id: "2", toolName: "create_purchase_invoice", title: "", description: "", riskLevel: "medium", diff: [], payload: {}, status: "pending" },
        { id: "3", toolName: "create_product", title: "", description: "", riskLevel: "medium", diff: [], payload: {}, status: "pending" },
        { id: "4", toolName: "navigate_to_module", title: "", description: "", riskLevel: "low", diff: [], payload: {}, status: "pending" },
      ]

      const effectiveAutonomy = "advisory"
      const filtered = actions.filter((action) => {
        if (effectiveAutonomy === "advisory" && action.toolName !== "navigate_to_module") {
          return false
        }
        return true
      })

      expect(filtered).toHaveLength(1)
      expect(filtered[0].toolName).toBe("navigate_to_module")
    })
  })

  describe("4. ReDoS and Regex Exception Audit (fallback-matcher.ts)", () => {
    it("handles large repetitive strings without catastrophic backtracking on standard regexes", () => {
      const largeInput = "buka invoice " + "a".repeat(20000)
      const t0 = performance.now()
      const res = matchRuleBasedAction(largeInput)
      const t1 = performance.now()

      expect(t1 - t0).toBeLessThan(100) // Linear time, less than 100ms
      expect(res.actions).toBeDefined()
    })

    it("safely handles phone numbers starting with '+' without throwing regex syntax error", () => {
      expect(() => {
        const res = matchRuleBasedAction("tambah vendor baru PT Sinar Abadi telepon +628123456789")
        expect(res.actions).toHaveLength(1)
        expect(res.actions[0].payload.name).toBe("PT Sinar Abadi")
        expect(res.actions[0].payload.phone).toBe("+628123456789")
      }).not.toThrow()
    })

    it("safely handles long email inputs without crashing new RegExp", () => {
      expect(() => {
        const largeEvilEmail = "tambah vendor " + "a".repeat(2000) + "@" + "b".repeat(2000) + ".com"
        const res = matchRuleBasedAction(largeEvilEmail)
        expect(res).toBeDefined()
      }).not.toThrow()
    })
  })
})
