import { render, screen, fireEvent, within } from "@testing-library/react"
import { PricingSection } from "@/components/landing/PricingSection"

function getCard(name: string): HTMLElement {
  const card = screen
    .getAllByRole("radio")
    .find((el) => within(el).queryByText(name, { exact: true }) !== null)
  if (!card) throw new Error(`Kartu "${name}" tidak ditemukan`)
  return card
}

describe("PricingSection", () => {
  it("menampilkan 3 tier: Starter, Bisnis, Enterprise", () => {
    render(<PricingSection />)
    expect(screen.getByRole("radiogroup", { name: "Pilih paket" })).toBeInTheDocument()
    expect(getCard("Starter")).toBeInTheDocument()
    expect(getCard("Bisnis")).toBeInTheDocument()
    expect(getCard("Enterprise")).toBeInTheDocument()
  })

  it("Bisnis terpilih secara default (badge Terpilih + aria-checked)", () => {
    render(<PricingSection />)
    expect(getCard("Bisnis")).toHaveAttribute("aria-checked", "true")
    expect(getCard("Bisnis")).toHaveTextContent("Terpilih")
    expect(getCard("Starter")).toHaveAttribute("aria-checked", "false")
  })

  it("klik kartu Starter memindahkan tanda Terpilih ke Starter", () => {
    render(<PricingSection />)
    fireEvent.click(getCard("Starter"))
    expect(getCard("Starter")).toHaveAttribute("aria-checked", "true")
    expect(getCard("Starter")).toHaveTextContent("Terpilih")
    expect(getCard("Bisnis")).toHaveAttribute("aria-checked", "false")
    expect(getCard("Bisnis")).not.toHaveTextContent("Terpilih")
  })

  it("klik kartu Enterprise memindahkan tanda Terpilih ke Enterprise", () => {
    render(<PricingSection />)
    fireEvent.click(getCard("Enterprise"))
    expect(getCard("Enterprise")).toHaveAttribute("aria-checked", "true")
    expect(getCard("Enterprise")).toHaveTextContent("Terpilih")
    expect(getCard("Bisnis")).toHaveAttribute("aria-checked", "false")
  })

  it("tekan tombol keyboard (Enter) pada kartu ikut memilih", () => {
    render(<PricingSection />)
    const starter = getCard("Starter")
    fireEvent.keyDown(starter, { key: "Enter" })
    expect(starter).toHaveAttribute("aria-checked", "true")
  })
})
