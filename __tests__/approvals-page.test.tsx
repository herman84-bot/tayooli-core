import { render, screen, fireEvent } from "@testing-library/react"
import "@testing-library/jest-dom"
import ApprovalsPage from "@/app/(app)/approvals/page"
import { useApprovals, useApprovalActions } from "@/hooks/useApprovals"
import type { Approval } from "@/lib/api"

jest.mock("@/hooks/useApprovals", () => ({
  useApprovals: jest.fn(),
  useApprovalActions: jest.fn(),
}))

const mockUseApprovals = useApprovals as jest.Mock
const mockUseApprovalActions = useApprovalActions as jest.Mock

const approvals: Approval[] = [
  {
    id: "appr-1",
    tenant_id: "t1",
    workflow_id: "11111111-2222-3333-4444-555555555555",
    target_type: "invoice",
    target_id: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    status: "pending",
    current_step_index: 0,
    requested_by: "user-1",
    created_at: "2026-08-01T10:00:00Z",
    updated_at: "2026-08-01T10:00:00Z",
  },
  {
    id: "appr-2",
    tenant_id: "t1",
    workflow_id: "99999999-8888-7777-6666-555555555555",
    target_type: "purchase_order",
    target_id: "bbbbbbbb-aaaa-cccc-dddd-eeeeeeeeeeee",
    status: "approved",
    current_step_index: 1,
    requested_by: "user-2",
    approved_by: "boss-1",
    created_at: "2026-07-30T09:00:00Z",
    updated_at: "2026-07-31T09:00:00Z",
  },
]

describe("ApprovalsPage", () => {
  const approveMutate = jest.fn()
  const rejectMutate = jest.fn()

  beforeEach(() => {
    approveMutate.mockClear()
    rejectMutate.mockClear()
    mockUseApprovals.mockReturnValue({ data: approvals, isLoading: false })
    mockUseApprovalActions.mockReturnValue({
      approve: { mutate: approveMutate },
      reject: { mutate: rejectMutate },
    })
  })

  it("renders title, pending count, and table rows", () => {
    render(<ApprovalsPage />)

    expect(screen.getByRole("heading", { name: /Approvals/ })).toBeInTheDocument()
    expect(screen.getByText("1 pending")).toBeInTheDocument()

    // Table headers
    expect(screen.getByText("Type")).toBeInTheDocument()
    expect(screen.getByText("Target")).toBeInTheDocument()
    expect(screen.getByText("Workflow")).toBeInTheDocument()
    expect(screen.getByText("Status")).toBeInTheDocument()

    // Row cells — target type and truncated UUIDs
    expect(screen.getByText("invoice")).toBeInTheDocument()
    expect(screen.getByText("purchase_order")).toBeInTheDocument()
    expect(screen.getAllByText(/^[a-z0-9]{8}\.\.\.$/).length).toBeGreaterThan(0)
  })

  it("shows the loading state while fetching", () => {
    mockUseApprovals.mockReturnValue({ data: undefined, isLoading: true })
    render(<ApprovalsPage />)

    expect(screen.getByText("Loading...")).toBeInTheDocument()
  })

  it("opens the drawer with details when a row is clicked", () => {
    render(<ApprovalsPage />)

    fireEvent.click(screen.getByText("invoice"))

    expect(screen.getByText("Approval Detail")).toBeInTheDocument()
    expect(screen.getByText(/invoice - aaaaaaaa/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Approve/ })).toBeInTheDocument()
  })

  it("calls approve.mutate with the id when Approve is clicked in the drawer", () => {
    render(<ApprovalsPage />)

    fireEvent.click(screen.getByText("invoice"))
    fireEvent.click(screen.getByRole("button", { name: /Approve/ }))

    expect(approveMutate).toHaveBeenCalledTimes(1)
    expect(approveMutate).toHaveBeenCalledWith("appr-1")
    // Drawer closes after the action
    expect(screen.queryByText("Approval Detail")).not.toBeInTheDocument()
  })

  it("calls reject.mutate with id and reason from the drawer", () => {
    render(<ApprovalsPage />)

    fireEvent.click(screen.getByText("invoice"))
    fireEvent.click(screen.getByRole("button", { name: /Reject/ }))
    fireEvent.change(screen.getByPlaceholderText("Why are you rejecting this?"), {
      target: { value: "Duplicate invoice" },
    })
    fireEvent.click(screen.getByRole("button", { name: /Confirm Rejection/ }))

    expect(rejectMutate).toHaveBeenCalledTimes(1)
    expect(rejectMutate).toHaveBeenCalledWith({ id: "appr-1", reason: "Duplicate invoice" })
    expect(screen.queryByText("Approval Detail")).not.toBeInTheDocument()
  })
})
