"use client"

import { useForm, useFieldArray } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Icon } from "@/components/ui/icon"
import { formatCurrency } from "@/lib/currency"
import type { CreateJournalEntryPayload } from "@/lib/api"

const journalLineSchema = z.object({
  account_id: z.string().min(1, "Account is required"),
  debit: z.coerce.number().min(0, "Must be >= 0"),
  credit: z.coerce.number().min(0, "Must be >= 0"),
})

const journalEntrySchema = z.object({
  date: z.string().min(1, "Date is required"),
  reference_id: z.string().min(1, "Reference is required"),
  description: z.string().min(1, "Description is required"),
  lines: z.array(journalLineSchema).min(2, "At least 2 line items required"),
})

type JournalEntryFormInput = z.input<typeof journalEntrySchema>
type JournalEntryFormData = z.output<typeof journalEntrySchema>

interface Account {
  id: string;
  code: string;
  name: string;
}

interface JEDrawerProps {
  onSubmit: (data: CreateJournalEntryPayload) => void
  isSubmitting?: boolean
  accounts?: Account[]
}

export function JEDrawer({ onSubmit, isSubmitting, accounts = [] }: JEDrawerProps) {
  const {
    register,
    control,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<JournalEntryFormInput, unknown, JournalEntryFormData>({
    resolver: zodResolver(journalEntrySchema),
    defaultValues: {
      date: new Date().toISOString().split("T")[0],
      reference_id: "",
      description: "",
      lines: [
        { account_id: "", debit: 0, credit: 0 },
        { account_id: "", debit: 0, credit: 0 },
      ],
    },
  })

  const { fields, append, remove } = useFieldArray({ control, name: "lines" })

  const watchedLines = watch("lines")
  const totalDebit = watchedLines.reduce((sum, line) => sum + (Number(line.debit) || 0), 0)
  const totalCredit = watchedLines.reduce((sum, line) => sum + (Number(line.credit) || 0), 0)
  const isBalanced = totalDebit === totalCredit && totalDebit > 0

  const handleFormSubmit = (data: JournalEntryFormData) => {
    onSubmit({
      date: data.date,
      reference_id: data.reference_id,
      description: data.description,
      lines: data.lines.map((l) => ({
        account_id: l.account_id,
        debit: Number(l.debit),
        credit: Number(l.credit),
      })),
    })
  }

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-4">
      {/* Header */}
      <div className="flex items-center gap-3 pb-4 border-b">
        <Icon name="BookOpen" size="lg" />
        <h2 className="text-lg font-semibold">Create Journal Entry</h2>
      </div>

      {/* Header fields */}
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <label htmlFor="date" className="text-sm font-medium">Date</label>
          <Input id="date" type="date" {...register("date")} />
          {errors.date && <p className="text-sm text-destructive">{errors.date.message}</p>}
        </div>
        <div className="space-y-1.5">
          <label htmlFor="reference_id" className="text-sm font-medium">Reference</label>
          <Input id="reference_id" {...register("reference_id")} placeholder="JE-001" />
          {errors.reference_id && <p className="text-sm text-destructive">{errors.reference_id.message}</p>}
        </div>
      </div>

      <div className="space-y-1.5">
        <label htmlFor="description" className="text-sm font-medium">Description</label>
        <Input id="description" {...register("description")} placeholder="Payment to vendor for Q3 services" />
        {errors.description && <p className="text-sm text-destructive">{errors.description.message}</p>}
      </div>

      {/* Line items */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium">Line Items</h3>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => append({ account_id: "", debit: 0, credit: 0 })}
          >
            <Icon name="Plus" size="sm" className="mr-1" />
            Add Line
          </Button>
        </div>

        {fields.map((field, index) => (
          <div key={field.id} className="grid grid-cols-[1fr_120px_120px_40px] gap-2 items-start">
            <div>
              <select
                {...register(`lines.${index}.account_id`)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option value="">Select account</option>
                {accounts.map((acc) => (
                  <option key={acc.id} value={acc.id}>
                    {acc.code} - {acc.name}
                  </option>
                ))}
              </select>
              {errors.lines?.[index]?.account_id && (
                <p className="text-xs text-destructive mt-1">{errors.lines[index]?.account_id?.message}</p>
              )}
            </div>
            <Input
              type="number"
              placeholder="Debit"
              min={0}
              step="0.01"
              {...register(`lines.${index}.debit`)}
            />
            <Input
              type="number"
              placeholder="Credit"
              min={0}
              step="0.01"
              {...register(`lines.${index}.credit`)}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => remove(index)}
              disabled={fields.length <= 2}
              className="mt-0"
            >
              <Icon name="Trash2" size="sm" className="text-destructive" />
            </Button>
          </div>
        ))}

        {errors.lines?.root && (
          <p className="text-sm text-destructive">{errors.lines.root.message}</p>
        )}
      </div>

      {/* Balance summary */}
      <div className={`flex items-center justify-between p-3 rounded-md text-sm ${isBalanced ? "bg-green-50 border border-green-200" : "bg-muted border"}`}>
        <div className="flex gap-6">
          <span className="tabular-nums">Debit: {formatCurrency(totalDebit)}</span>
          <span className="tabular-nums">Credit: {formatCurrency(totalCredit)}</span>
        </div>
        {!isBalanced ? (
          <span className="text-destructive text-xs font-medium">Not balanced</span>
        ) : (
          <span className="text-green-700 text-xs font-medium">Balanced</span>
        )}
      </div>

      {/* Footer */}
      <div className="pt-4 border-t flex justify-end gap-2">
        <Button type="submit" disabled={isSubmitting || !isBalanced}>
          {isSubmitting ? "Creating..." : "Create Entry"}
        </Button>
      </div>
    </form>
  )
}
