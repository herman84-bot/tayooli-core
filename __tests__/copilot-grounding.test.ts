import { matchRuleBasedAction } from "@/lib/copilot/fallback-matcher"

// Grounding overhaul: nilai EXACT dari user, tanpa placeholder, tanpa karangan.
describe("Copilot Grounding Overhaul (RED — harus hijau setelah fix)", () => {
  it("vendor lengkap -> args exact 4 field (nama/email/telepon/alamat)", () => {
    const res = matchRuleBasedAction(
      "Tambah vendor baru dengan nama PT Sinar Terang Sejati, email halo@sinarterang.id, telepon 081222333444, alamat Jl Merdeka 1 Bandung"
    )
    expect(res.actions).toHaveLength(1)
    const p = res.actions[0].payload as Record<string, string>
    expect(p.name).toBe("PT Sinar Terang Sejati")
    expect(p.email).toBe("halo@sinarterang.id")
    expect(p.phone).toBe("081222333444")
    expect(p.address || "").toContain("Jl Merdeka 1 Bandung")
  })

  it("vendor tanpa nama -> NOL aksi + tanya ulang (jangan karang placeholder)", () => {
    for (const msg of ["tambah vendor baru", "buat vendor baru", "daftarkan vendor"]) {
      const res = matchRuleBasedAction(msg)
      expect(res.actions).toHaveLength(0)
      expect(res.reply.toLowerCase()).toContain("nama")
      const payloadNames = (res.actions || []).map((a) => (a.payload as any)?.name)
      expect(payloadNames).not.toContain("(Isi Nama Vendor)")
    }
  })

  it("frasa bebas tanpa nama jelas -> NOL aksi (jangan jadikan junk sebagai nama)", () => {
    const res = matchRuleBasedAction("Saya mau tambah vendor baru untuk suplai kertas")
    expect(res.actions).toHaveLength(0)
    expect(res.reply.toLowerCase()).toContain("nama")
  })

  it("company tanpa nama jelas -> NOL aksi + tanya ulang", () => {
    const res = matchRuleBasedAction("ubah nama perusahaan")
    expect(res.actions).toHaveLength(0)
    expect(res.reply.toLowerCase()).toContain("nama")
  })

  it("customer tanpa nama -> NOL aksi + tanya ulang", () => {
    const res = matchRuleBasedAction("tambah customer baru")
    expect(res.actions).toHaveLength(0)
    expect(res.reply.toLowerCase()).toContain("nama")
  })
})
