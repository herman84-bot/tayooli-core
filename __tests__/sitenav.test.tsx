import { act, fireEvent, render, screen } from "@testing-library/react"
import { SiteNav } from "@/components/landing/SiteNav"

// Simulated absolute positions of each landing section.
const SECTION_TOPS: Record<string, number> = {
  "#fitur": 600,
  "#cara-kerja": 1700,
  "#harga": 2600,
  "#faq": 3400,
}
const DOC_HEIGHT = 4000
const VIEWPORT = 800

function Scaffold() {
  return (
    <div>
      <SiteNav />
      <div id="fitur" />
      <div id="cara-kerja" />
      <div id="harga" />
      <div id="faq" />
    </div>
  )
}

const origRect = Element.prototype.getBoundingClientRect

beforeAll(() => {
  // Sections report a viewport-relative top derived from the fake scrollY.
  Element.prototype.getBoundingClientRect = function (this: Element) {
    const id = this.id
    const absTop = id ? SECTION_TOPS[`#${id}`] : undefined
    if (absTop !== undefined) {
      const top = absTop - window.scrollY
      return {
        top,
        bottom: top + 400,
        left: 0,
        right: 0,
        width: 100,
        height: 400,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      } as DOMRect
    }
    return origRect.call(this)
  }
  Object.defineProperty(window, "innerHeight", { value: VIEWPORT, configurable: true })
  Object.defineProperty(document.documentElement, "scrollHeight", {
    value: DOC_HEIGHT,
    configurable: true,
  })
})

afterAll(() => {
  Element.prototype.getBoundingClientRect = origRect
})

async function scrollTo(y: number) {
  Object.defineProperty(window, "scrollY", { value: y, configurable: true })
  await act(async () => {
    fireEvent.scroll(window)
    await new Promise((r) => setTimeout(r, 40)) // let the rAF-throttled handler run
  })
}

function activeHrefs(): string[] {
  return screen
    .getAllByRole("link")
    .filter((l) => l.getAttribute("aria-current") === "true")
    .map((l) => l.getAttribute("href") ?? "")
}

describe("SiteNav — scroll-spy landing nav", () => {
  it("renders all four anchor links plus the wordmark", () => {
    render(<Scaffold />)
    for (const label of ["Fitur", "Cara Kerja", "Harga", "FAQ"]) {
      expect(screen.getByRole("link", { name: label })).toHaveAttribute(
        "href",
        `#${label === "Cara Kerja" ? "cara-kerja" : label === "Harga" ? "harga" : label === "FAQ" ? "faq" : "fitur"}`
      )
    }
    expect(screen.getByText("Tayooli")).toBeInTheDocument()
  })

  it("starts with no section active at the top of the page", async () => {
    render(<Scaffold />)
    await scrollTo(0)
    expect(activeHrefs()).toEqual([])
  })

  it("clicking a link activates it immediately and moves the indicator", async () => {
    render(<Scaffold />)
    await scrollTo(0)

    const harga = screen.getByRole("link", { name: "Harga" })
    expect(harga.getAttribute("aria-current")).toBeNull()

    fireEvent.click(harga)

    expect(harga).toHaveAttribute("aria-current", "true")
    expect(harga.className).toContain("text-primary")
    const bar = harga.querySelector('[aria-hidden="true"]')
    expect(bar).not.toBeNull()
    expect(bar!.className).toContain("opacity-100")

    // Clicking another link moves the state.
    fireEvent.click(screen.getByRole("link", { name: "Fitur" }))
    expect(harga.getAttribute("aria-current")).toBeNull()
    expect(screen.getByRole("link", { name: "Fitur" })).toHaveAttribute(
      "aria-current",
      "true"
    )
  })

  it("marks the section that crosses the probe line while scrolling", async () => {
    render(<Scaffold />)
    await scrollTo(0)
    expect(activeHrefs()).toEqual([])

    // #fitur (abs 600) crosses probe 112 → active
    await scrollTo(700)
    expect(activeHrefs()).toEqual(["#fitur"])

    // #cara-kerja (abs 1700) crosses probe 112 → active
    await scrollTo(1700)
    expect(activeHrefs()).toEqual(["#cara-kerja"])

    // #harga (abs 2600) → active
    await scrollTo(2600)
    expect(activeHrefs()).toEqual(["#harga"])
  })

  it("forces the last section active at the bottom of the page", async () => {
    render(<Scaffold />)
    await scrollTo(0)
    expect(activeHrefs()).toEqual([])

    // scrollY + viewport >= doc height → #faq forced active
    await scrollTo(DOC_HEIGHT - VIEWPORT + 4)
    expect(activeHrefs()).toEqual(["#faq"])
  })
})
