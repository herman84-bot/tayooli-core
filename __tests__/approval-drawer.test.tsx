import { render, screen, fireEvent } from "@testing-library/react"
import "@testing-library/jest-dom"
import { ApprovalDrawer } from "@/components/approvals/ApprovalDrawer"
import type { Approval } from "@/lib/api"

const baseApproval: Approval = {
  id: "appr-1",
  tenant_id: "t1",
  workflow_id: "wf-123",
  target_type: "invoice",
  target_id: "inv-456",
  status: "pending",
  current_step_index: 0,
  requested_by: "user-1",
  created_at: "2026-08-01T10:00:00Z",
  updated_at: "2026-08-01T10:00:00Z",
}

describe("ApprovalDrawer", () => {
  it("renders status, target, and detail rows for a pending approval", () => {
    render(<ApprovalDrawer approval={baseApproval} onApprove={jest.fn()} onReject={jest.fn()} />)

    expect(screen.getByText("Pending")).toBeInTheDocument()
    expect(screen.getByText("invoice - inv-456")).toBeInTheDocument()
    expect(screen.getByText("Workflow")).toBeInTheDocument()
    expect(screen.getByText("wf-123")).toBeInTheDocument()
    expect(screen.getByText("Requested by")).toBeInTheDocument()
    expect(screen.getByText("user-1")).toBeInTheDocument()
    expect(screen.getByText("Created")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Approve/ })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Reject/ })).toBeInTheDocument()
  })

  it("calls onApprove with the approval id", () => {
    const onApprove = jest.fn()
    render(<ApprovalDrawer approval={baseApproval} onApprove={onApprove} onReject={jest.fn()} />)

    fireEvent.click(screen.getByRole("button", { name: /Approve/ }))

    expect(onApprove).toHaveBeenCalledTimes(1)
    expect(onApprove).toHaveBeenCalledWith("appr-1")
  })

  it("reject flow requires a reason before confirming", () => {
    const onReject = jest.fn()
    render(<ApprovalDrawer approval={baseApproval} onApprove={jest.fn()} onReject={onReject} />)

    fireEvent.click(screen.getByRole("button", { name: /Reject/ }))

    // Reject button is replaced by the confirm/cancel pair
    expect(screen.getByText("Rejection reason")).toBeInTheDocument()
    const confirm = screen.getByRole("button", { name: /Confirm Rejection/ })
    expect(confirm).toBeDisabled()

    fireEvent.change(screen.getByPlaceholderText("Why are you rejecting this?"), {
      target: { value: "Duplicate invoice" },
    })
    expect(confirm).toBeEnabled()

    fireEvent.click(confirm)
    expect(onReject).toHaveBeenCalledTimes(1)
    expect(onReject).toHaveBeenCalledWith({ id: "appr-1", reason: "Duplicate invoice" })
  })

  it("cancel resets the reject flow", () => {
    render(<ApprovalDrawer approval={baseApproval} onApprove={jest.fn()} onReject={jest.fn()} />)

    fireEvent.click(screen.getByRole("button", { name: /Reject/ }))
    fireEvent.click(screen.getByRole("button", { name: /Cancel/ }))

    // Back to the Approve/Reject pair
    expect(screen.getByRole("button", { name: /Approve/ })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Reject/ })).toBeInTheDocument()
  })

  it("approved approval shows handled by and no action buttons", () => {
    const approved: Approval = {
      ...baseApproval,
      status: "approved",
      approved_by: "boss-1",
      approved_at: "2026-08-02T09:00:00Z",
    }
    render(<ApprovalDrawer approval={approved} onApprove={jest.fn()} onReject={jest.fn()} />)

    expect(screen.getByText("Approved")).toBeInTheDocument()
    expect(screen.getByText("Handled by")).toBeInTheDocument()
    expect(screen.getByText("boss-1")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /Approve/ })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /Reject/ })).not.toBeInTheDocument()
  })

  it("rejected approval shows the rejection reason row", () => {
    const rejected: Approval = {
      ...baseApproval,
      status: "rejected",
      rejected_by: "boss-1",
      rejection_reason: "Missing PO reference",
    }
    render(<ApprovalDrawer approval={rejected} onApprove={jest.fn()} onReject={jest.fn()} />)

    expect(screen.getByText("Rejected")).toBeInTheDocument()
    expect(screen.getByText("Reason")).toBeInTheDocument()
    expect(screen.getByText("Missing PO reference")).toBeInTheDocument()
  })
})
