import React from "react"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { CategoryCombobox } from "@/components/products/CategoryCombobox"
import * as hooks from "@/hooks/useProductCategories"

jest.mock("@/hooks/useProductCategories")

const mockCategories = [
  { id: "cat-1", tenant_id: "t-1", name: "Makanan", created_at: "2026-01-01", product_count: 5 },
  { id: "cat-2", tenant_id: "t-1", name: "Minuman", created_at: "2026-01-01", product_count: 12 },
]

describe("CategoryCombobox", () => {
  const mockCreate = jest.fn()

  beforeEach(() => {
    jest.clearAllMocks()
    ;(hooks.useProductCategories as jest.Mock).mockReturnValue({
      data: mockCategories,
      isLoading: false,
    })
    ;(hooks.useCreateCategory as jest.Mock).mockReturnValue({
      mutateAsync: mockCreate,
      isPending: false,
    })
  })

  it("renders placeholder when value is null", () => {
    render(<CategoryCombobox value={null} onChange={jest.fn()} placeholder="Pilih kategori..." />)
    expect(screen.getByText("Pilih kategori...")).toBeInTheDocument()
  })

  it("renders selected category name when value matches an id", () => {
    render(<CategoryCombobox value="cat-2" onChange={jest.fn()} />)
    expect(screen.getByText("Minuman")).toBeInTheDocument()
  })

  it("opens dropdown and allows selecting an existing category", () => {
    const handleChange = jest.fn()
    render(<CategoryCombobox value={null} onChange={handleChange} />)

    fireEvent.click(screen.getByRole("button", { name: /kategori produk/i }))
    expect(screen.getByText("Makanan")).toBeInTheDocument()
    expect(screen.getByText("Minuman")).toBeInTheDocument()

    fireEvent.click(screen.getByText("Makanan"))
    expect(handleChange).toHaveBeenCalledWith("cat-1", "Makanan")
  })

  it("allows selecting 'Tanpa Kategori' to clear category", () => {
    const handleChange = jest.fn()
    render(<CategoryCombobox value="cat-1" onChange={handleChange} allowClear={true} />)

    fireEvent.click(screen.getByRole("button", { name: /kategori produk: Makanan/i }))
    fireEvent.click(screen.getByText("-- Tanpa Kategori --"))
    expect(handleChange).toHaveBeenCalledWith(null, "")
  })

  it("clears category via clear button without opening dropdown", () => {
    const handleChange = jest.fn()
    render(<CategoryCombobox value="cat-1" onChange={handleChange} allowClear={true} />)

    const clearBtn = screen.getByRole("button", { name: /hapus pilihan kategori/i })
    expect(clearBtn).toBeInTheDocument()

    fireEvent.click(clearBtn)
    expect(handleChange).toHaveBeenCalledWith(null, "")
  })

  it("supports keyboard navigation via ArrowDown and Enter", () => {
    const handleChange = jest.fn()
    render(<CategoryCombobox value={null} onChange={handleChange} allowClear={true} />)

    fireEvent.click(screen.getByRole("button", { name: /kategori produk/i }))
    const input = screen.getByLabelText("Cari kategori")

    // Arrow down once -> highlights '-- Tanpa Kategori --' (index 0)
    fireEvent.keyDown(input, { key: "ArrowDown" })
    // Arrow down second time -> highlights 'Makanan' (index 1)
    fireEvent.keyDown(input, { key: "ArrowDown" })
    // Press Enter to select
    fireEvent.keyDown(input, { key: "Enter" })

    expect(handleChange).toHaveBeenCalledWith("cat-1", "Makanan")
  })

  it("shows inline create button when user types a new category name", async () => {
    const handleChange = jest.fn()
    mockCreate.mockResolvedValue({ id: "cat-3", name: "Elektronik" })

    render(<CategoryCombobox value={null} onChange={handleChange} />)

    fireEvent.click(screen.getByRole("button", { name: /kategori produk/i }))
    const input = screen.getByLabelText("Cari kategori")
    fireEvent.change(input, { target: { value: "Elektronik" } })

    const createBtn = screen.getByRole("button", { name: /\+ Buat Kategori/i })
    expect(createBtn).toBeInTheDocument()

    fireEvent.click(createBtn)
    await waitFor(() => {
      expect(mockCreate).toHaveBeenCalledWith("Elektronik")
      expect(handleChange).toHaveBeenCalledWith("cat-3", "Elektronik")
    })
  })

  it("displays error message if category creation fails", async () => {
    mockCreate.mockRejectedValue(new Error("conflict: nama kategori sudah ada"))

    render(<CategoryCombobox value={null} onChange={jest.fn()} />)

    fireEvent.click(screen.getByRole("button", { name: /kategori produk/i }))
    const input = screen.getByLabelText("Cari kategori")
    fireEvent.change(input, { target: { value: "Makanan" } })

    // Simulate creation failure
    const createBtn = screen.queryByRole("button", { name: /\+ Buat Kategori/i })
    // Since "Makanan" exists, hasExactMatch is true. Let's type a non-existing one.
    fireEvent.change(input, { target: { value: "Bahan Mentah" } })
    const validCreateBtn = screen.getByRole("button", { name: /\+ Buat Kategori/i })
    fireEvent.click(validCreateBtn)

    await waitFor(() => {
      expect(screen.getByText("conflict: nama kategori sudah ada")).toBeInTheDocument()
    })
  })
})
