import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import "@testing-library/jest-dom"
import { JEDrawer } from "@/components/accounting/JEDrawer"

const accounts = [
  { id: "acc-1", code: "1000", name: "Cash" },
  { id: "acc-2", code: "5000", name: "Office Expense" },
]

describe("JEDrawer", () => {
  it("renders header fields and two default line items", () => {
    render(<JEDrawer onSubmit={jest.fn()} accounts={accounts} />)

    expect(screen.getByText("Create Journal Entry")).toBeInTheDocument()
    expect(screen.getByLabelText("Date")).toBeInTheDocument()
    expect(screen.getByLabelText("Reference")).toBeInTheDocument()
    expect(screen.getByLabelText("Description")).toBeInTheDocument()
    expect(screen.getByText("Line Items")).toBeInTheDocument()

    // Two default line rows (each with Debit and Credit inputs)
    expect(screen.getAllByPlaceholderText("Debit")).toHaveLength(2)
    expect(screen.getAllByPlaceholderText("Credit")).toHaveLength(2)
    // Each row has an account select (both renders all options)
    expect(screen.getAllByRole("combobox")).toHaveLength(2)
    expect(screen.getAllByRole("option", { name: "1000 - Cash" })).toHaveLength(2)
    expect(screen.getAllByRole("option", { name: "5000 - Office Expense" })).toHaveLength(2)
  })

  it("shows Not balanced and disables submit until debits equal credits", () => {
    render(<JEDrawer onSubmit={jest.fn()} accounts={accounts} />)

    // Default state: all zeros → not balanced → disabled
    expect(screen.getByText("Not balanced")).toBeInTheDocument()
    const submit = screen.getByRole("button", { name: "Create Entry" })
    expect(submit).toBeDisabled()

    // Fill line 0: debit 100, line 1: credit 100 → balanced
    const debitInputs = screen.getAllByPlaceholderText("Debit")
    const creditInputs = screen.getAllByPlaceholderText("Credit")
    fireEvent.change(debitInputs[0], { target: { value: "100" } })
    fireEvent.change(creditInputs[1], { target: { value: "100" } })

    expect(screen.getByText("Balanced")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Create Entry" })).toBeEnabled()
  })

  it("submits the form with numeric debit/credit values", async () => {
    const onSubmit = jest.fn()
    render(<JEDrawer onSubmit={onSubmit} accounts={accounts} />)

    fireEvent.change(screen.getByLabelText("Date"), { target: { value: "2026-08-01" } })
    fireEvent.change(screen.getByLabelText("Reference"), { target: { value: "JE-001" } })
    fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Payment to vendor" } })

    const selects = screen.getAllByRole("combobox") as HTMLSelectElement[]
    fireEvent.change(selects[0], { target: { value: "acc-1" } })
    fireEvent.change(selects[1], { target: { value: "acc-2" } })
    expect(selects[0].value).toBe("acc-1")
    expect(selects[1].value).toBe("acc-2")

    const debitInputs = screen.getAllByPlaceholderText("Debit")
    const creditInputs = screen.getAllByPlaceholderText("Credit")
    fireEvent.change(debitInputs[0], { target: { value: "100" } })
    fireEvent.change(creditInputs[1], { target: { value: "100" } })

    // Balance must be reached and the submit button enabled before submitting
    expect(screen.getByText("Balanced")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Create Entry" })).toBeEnabled()
    fireEvent.submit(document.querySelector("form")!)

    // handleSubmit validates asynchronously (await resolver), so wait for it
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1))
    expect(onSubmit).toHaveBeenCalledWith({
      date: "2026-08-01",
      reference_id: "JE-001",
      description: "Payment to vendor",
      lines: [
        { account_id: "acc-1", debit: 100, credit: 0 },
        { account_id: "acc-2", debit: 0, credit: 100 },
      ],
    })
  })

  it("Add Line appends a new line item row", () => {
    render(<JEDrawer onSubmit={jest.fn()} accounts={accounts} />)

    expect(screen.getAllByPlaceholderText("Debit")).toHaveLength(2)
    fireEvent.click(screen.getByRole("button", { name: /Add Line/ }))
    expect(screen.getAllByPlaceholderText("Debit")).toHaveLength(3)
    expect(screen.getAllByRole("combobox")).toHaveLength(3)
  })
})
