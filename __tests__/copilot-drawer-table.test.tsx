import React from "react"
import { render } from "@testing-library/react"
import "@testing-library/jest-dom"
import { renderFormattedMessage } from "@/components/copilot/CopilotDrawer"

describe("CopilotDrawer Table Rendering (Scenario 6)", () => {
  it("renders table with pipes into valid <table>, <thead>, <tbody>, <tr>, <td> elements", () => {
    const markdown = [
      "Berikut adalah rincian anggaran yang diajukan:",
      "| Deskripsi | Jumlah |",
      "| --- | --- |",
      "| Pengadaan Laptop | Rp 15.000.000 |",
      "| ATK Kantor | Rp 500.000 |",
      "Silakan konfirmasi jika sudah sesuai.",
    ].join("\n")

    const { container } = render(<div>{renderFormattedMessage(markdown)}</div>)

    // Verify <table> exists
    const table = container.querySelector("table")
    expect(table).not.toBeNull()
    expect(table?.tagName.toLowerCase()).toBe("table")

    // Verify <thead> exists
    const thead = table?.querySelector("thead")
    expect(thead).not.toBeNull()
    expect(thead?.tagName.toLowerCase()).toBe("thead")

    // Verify <th> headers
    const thElements = thead?.querySelectorAll("th")
    expect(thElements?.length).toBe(2)
    expect(thElements?.[0].textContent?.trim()).toBe("Deskripsi")
    expect(thElements?.[1].textContent?.trim()).toBe("Jumlah")

    // Verify <tbody> exists
    const tbody = table?.querySelector("tbody")
    expect(tbody).not.toBeNull()
    expect(tbody?.tagName.toLowerCase()).toBe("tbody")

    // Verify <tr> rows in tbody
    const trElements = tbody?.querySelectorAll("tr")
    expect(trElements?.length).toBe(2)

    // Verify <td> cells in tbody
    const tdElements = tbody?.querySelectorAll("td")
    expect(tdElements?.length).toBe(4)
    expect(tdElements?.[0].textContent?.trim()).toBe("Pengadaan Laptop")
    expect(tdElements?.[1].textContent?.trim()).toBe("Rp 15.000.000")
    expect(tdElements?.[2].textContent?.trim()).toBe("ATK Kantor")
    expect(tdElements?.[3].textContent?.trim()).toBe("Rp 500.000")
  })

  it("renders pipe-delimited table rows even without explicit separator line", () => {
    const markdown = [
      "| Deskripsi | Jumlah |",
      "| Biaya Server GCP | Rp 2.500.000 |",
    ].join("\n")

    const { container } = render(<div>{renderFormattedMessage(markdown)}</div>)

    const table = container.querySelector("table")
    expect(table).not.toBeNull()

    const thead = table?.querySelector("thead")
    expect(thead).not.toBeNull()

    const tbody = table?.querySelector("tbody")
    expect(tbody).not.toBeNull()

    const tdElements = tbody?.querySelectorAll("td")
    expect(tdElements?.length).toBe(2)
    expect(tdElements?.[0].textContent?.trim()).toBe("Biaya Server GCP")
    expect(tdElements?.[1].textContent?.trim()).toBe("Rp 2.500.000")
  })
})
