"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Icon } from "@/components/ui/icon"
import type { Approval } from "@/lib/api"

interface ApprovalDrawerProps {
  approval: Approval
  onApprove: (id: string) => void
  onReject: (args: { id: string; reason: string }) => void
}

const statusConfig: Record<string, { label: string; color: string; bg: string }> = {
  pending:  { label: "Pending",  color: "text-yellow-700",  bg: "bg-yellow-100" },
  approved: { label: "Approved", color: "text-green-700",   bg: "bg-green-100"  },
  rejected: { label: "Rejected", color: "text-red-700",     bg: "bg-red-100"    },
}

export function ApprovalDrawer({ approval, onApprove, onReject }: ApprovalDrawerProps) {
  const [rejectReason, setRejectReason] = useState("")
  const [showReject, setShowReject] = useState(false)
  const status = statusConfig[approval.status]

  return (
    <div className="space-y-6">
      {/* Status badge */}
      <div className="flex items-center gap-3">
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${status.bg} ${status.color}`}>
          {status.label}
        </span>
        <span className="text-sm text-muted-foreground">
          {approval.target_type} - {approval.target_id}
        </span>
      </div>

      {/* Detail rows */}
      <div className="space-y-3">
        <DetailRow label="Target type" value={approval.target_type} />
        <DetailRow label="Target ID" value={approval.target_id} />
        {approval.workflow_id && <DetailRow label="Workflow" value={approval.workflow_id} />}
        <DetailRow label="Requested by" value={approval.requested_by} />
        {approval.approved_by && <DetailRow label="Handled by" value={approval.approved_by} />}
        <DetailRow label="Created" value={new Date(approval.created_at).toLocaleString()} />
        {approval.rejection_reason && <DetailRow label="Reason" value={approval.rejection_reason} />}
      </div>

      {/* Reject reason input */}
      {showReject && (
        <div className="space-y-2">
          <label className="text-sm font-medium">Rejection reason</label>
          <Input
            placeholder="Why are you rejecting this?"
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
          />
        </div>
      )}

      {/* Actions — only show for pending */}
      {approval.status === "pending" && (
        <div className="flex items-center gap-2 pt-2">
          {!showReject ? (
            <>
              <Button
                variant="default"
                onClick={() => onApprove(approval.id)}
              >
                <Icon name="Check" size="sm" className="mr-1" />
                Approve
              </Button>
              <Button
                variant="destructive"
                onClick={() => setShowReject(true)}
              >
                <Icon name="X" size="sm" className="mr-1" />
                Reject
              </Button>
            </>
          ) : (
            <>
              <Button
                variant="destructive"
                disabled={!rejectReason.trim()}
                onClick={() => onReject({ id: approval.id, reason: rejectReason })}
              >
                Confirm Rejection
              </Button>
              <Button
                variant="ghost"
                onClick={() => { setShowReject(false); setRejectReason("") }}
              >
                Cancel
              </Button>
            </>
          )}
        </div>
      )}
    </div>
  )
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="text-sm font-medium">{value}</span>
    </div>
  )
}
