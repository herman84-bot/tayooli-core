"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Icon } from "@/components/ui/icon"

const coaSchema = z.object({
  code: z.string().min(3, "Min 3 characters"),
  name: z.string().min(1, "Name is required"),
  type: z.enum(["Asset", "Liability", "Equity", "Revenue", "Expense"]),
  parent_id: z.string().optional(),
})

type CoAFormData = z.infer<typeof coaSchema>

interface CoADrawerProps {
  onSubmit: (data: CoAFormData) => void
  isSubmitting?: boolean
}

const accountTypes = ["Asset", "Liability", "Equity", "Revenue", "Expense"] as const

export function CoADrawer({ onSubmit, isSubmitting }: CoADrawerProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<CoAFormData>({
    resolver: zodResolver(coaSchema),
    defaultValues: { type: "Asset" },
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="flex items-center gap-3 pb-4 border-b">
        <Icon name="Calculator" size="lg" />
        <h2 className="text-lg font-semibold">Create Account</h2>
      </div>

      <div className="space-y-4">
        <div className="space-y-1.5">
          <label htmlFor="code" className="text-sm font-medium">
            Account Code
          </label>
          <Input
            id="code"
            {...register("code")}
            placeholder="1001"
          />
          {errors.code && (
            <p className="text-sm text-destructive">{errors.code.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="name" className="text-sm font-medium">
            Account Name
          </label>
          <Input
            id="name"
            {...register("name")}
            placeholder="Cash and Cash Equivalents"
          />
          {errors.name && (
            <p className="text-sm text-destructive">{errors.name.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="type" className="text-sm font-medium">
            Account Type
          </label>
          <select
            id="type"
            {...register("type")}
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            {accountTypes.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
        </div>

        <div className="space-y-1.5">
          <label htmlFor="parent_id" className="text-sm font-medium">
            Parent Account ID <span className="text-muted-foreground">(optional)</span>
          </label>
          <Input
            id="parent_id"
            {...register("parent_id")}
            placeholder="Leave empty for top-level account"
          />
        </div>
      </div>

      <div className="pt-4 border-t flex justify-end gap-2">
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? "Creating..." : "Create Account"}
        </Button>
      </div>
    </form>
  )
}
