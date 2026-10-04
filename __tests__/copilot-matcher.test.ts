import { matchRuleBasedAction } from "@/lib/copilot/fallback-matcher"

describe("Copilot Fallback Matcher (Zero-AI Rule Engine)", () => {
  it("matches company profile update requests with high risk and confirmation", () => {
    const res = matchRuleBasedAction("Tolong ubah nama perusahaan jadi PT Sukses")
    expect(res.actions).toHaveLength(1)
    expect(res.actions[0].toolName).toBe("update_company_profile")
    expect(res.actions[0].riskLevel).toBe("high")
    expect(res.actions[0].requiresConfirmationText).toBe("KONFIRMASI")
  })

  it("extracts email and role when inviting team members", () => {
    const res = matchRuleBasedAction("Undang anggota tim budi@perusahaan.com jadi accountant")
    expect(res.actions).toHaveLength(1)
    expect(res.actions[0].toolName).toBe("invite_team_member")
    expect(res.actions[0].payload.email).toBe("budi@perusahaan.com")
    expect(res.actions[0].payload.role).toBe("accountant")
  })

  it("classifies payment gateway configuration as high risk with confirmation required", () => {
    const res = matchRuleBasedAction("Tolong setup payment gateway midtrans")
    expect(res.actions).toHaveLength(1)
    expect(res.actions[0].toolName).toBe("configure_payment_gateway")
    expect(res.actions[0].riskLevel).toBe("high")
    expect(res.actions[0].requiresConfirmationText).toBe("KONFIRMASI")
    expect(res.actions[0].payload.provider).toBe("midtrans")
  })

  it("asks for vendor name when input lacks specific name (no junk proposal)", () => {
    const res = matchRuleBasedAction("Saya mau tambah vendor baru untuk suplai kertas")
    expect(res.actions).toHaveLength(0)
    expect(res.reply.toLowerCase()).toContain("nama")

    // Grounded request creates proposal properly
    const groundedRes = matchRuleBasedAction("Saya mau tambah vendor baru bernama PT Kertas Nusantara")
    expect(groundedRes.actions).toHaveLength(1)
    expect(groundedRes.actions[0].toolName).toBe("create_vendor")
    expect(groundedRes.actions[0].payload.name).toBe("PT Kertas Nusantara")
    expect(groundedRes.actions[0].riskLevel).toBe("medium")
  })

  it("creates purchase invoice and NEVER triggers create_vendor when user asks for invoice", () => {
    const res = matchRuleBasedAction("buatkan invoice untuk vendor PT Maju Jaya sebesar 5 juta rupiah")
    expect(res.actions).toHaveLength(1)
    expect(res.actions[0].toolName).toBe("create_purchase_invoice")
    expect(res.actions[0].riskLevel).toBe("medium")
    expect(res.actions[0].payload.vendor_name).toBe("PT Maju Jaya")
    expect(res.actions[0].payload.amount).toBe(5000000)
    expect(res.actions[0].payload.currency).toBe("IDR")
  })

  it("evaluates navigation first for 'buka vendor', 'buka halaman vendors', 'lihat vendor' (Scenario 2)", () => {
    for (const phrase of ["buka vendor", "buka halaman vendors", "lihat vendor"]) {
      const res = matchRuleBasedAction(phrase)
      expect(res.actions).toHaveLength(1)
      expect(res.actions[0].toolName).toBe("navigate_to_module")
      expect(res.actions[0].payload.path).toBe("/dashboard/vendors")
    }

    const resInvoice = matchRuleBasedAction("buka invoice")
    expect(resInvoice.actions).toHaveLength(1)
    expect(resInvoice.actions[0].toolName).toBe("navigate_to_module")
    expect(resInvoice.actions[0].payload.path).toBe("/dashboard/invoices")

    const resPO = matchRuleBasedAction("buka po")
    expect(resPO.actions).toHaveLength(1)
    expect(resPO.actions[0].toolName).toBe("navigate_to_module")
    expect(resPO.actions[0].payload.path).toBe("/dashboard/purchase-orders")
  })

  it("handles interrogative queries without triggering create_vendor (Scenario 3)", () => {
    const res = matchRuleBasedAction("vendor apa yang paling sering digunakan?")
    expect(res.actions).toHaveLength(0)
    expect(res.reply).toBeTruthy()
    expect(res.actions.some((a) => a.toolName === "create_vendor")).toBe(false)
  })

  it("extracts clean vendor name and email when registering vendor", () => {
    const res = matchRuleBasedAction("tambah vendor PT Berkah email vendor@berkah.com")
    expect(res.actions).toHaveLength(1)
    expect(res.actions[0].toolName).toBe("create_vendor")
    expect(res.actions[0].riskLevel).toBe("medium")
    expect(res.actions[0].payload.name).toBe("PT Berkah")
    expect(res.actions[0].payload.email).toBe("vendor@berkah.com")
  })

  it("cleans stop-words and amount when registering vendor (Scenario 4)", () => {
    const res = matchRuleBasedAction("tambah vendor baru PT Sinar Abadi sebesar 10jt telepon 08123456789")
    expect(res.actions).toHaveLength(1)
    expect(res.actions[0].toolName).toBe("create_vendor")
    expect(res.actions[0].payload.name).toBe("PT Sinar Abadi")
    expect(res.actions[0].payload.phone).toBe("08123456789")
  })

  it("matches financial dashboard summary request for 'ringkasan keuangan' and 'summary kas' (Scenario 5)", () => {
    for (const phrase of ["ringkasan keuangan", "summary kas"]) {
      const res = matchRuleBasedAction(phrase)
      expect(res.actions).toHaveLength(1)
      expect(res.actions[0].toolName).toBe("get_dashboard_summary")
      expect(res.actions[0].riskLevel).toBe("low")
    }
  })

  it("matches navigation to accounting chart of accounts", () => {
    const res = matchRuleBasedAction("Buka modul chart of account")
    expect(res.actions).toHaveLength(1)
    expect(res.actions[0].toolName).toBe("navigate_to_module")
    expect(res.actions[0].payload.path).toBe("/accounting/chart-of-accounts")
  })

  it("responds empathetically and warmly to greetings", () => {
    const res = matchRuleBasedAction("halo")
    expect(res.actions).toHaveLength(0)
    expect(res.reply).toContain("Halo! Senang bisa menyapa Anda")

    const res2 = matchRuleBasedAction("selamat pagi")
    expect(res2.reply).toContain("Halo! Senang bisa menyapa Anda")
  })

  it("provides helpful default guidance when query does not match standard patterns", () => {
    const res = matchRuleBasedAction("cuaca hari ini bagaimana?")
    expect(res.actions).toHaveLength(0)
    expect(res.reply).toContain("Saya siap membantu Anda di workspace ini")
  })
})
