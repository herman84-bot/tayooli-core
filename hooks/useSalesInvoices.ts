import { useState, useEffect, useCallback } from 'react';
import { extractApiErrorMessage } from '@/lib/api/errors';

// Bentuk response nyata dari backend Go (domain.SalesInvoice):
// id, tenant_id, sales_order_id, invoice_number, amount (string decimal),
// status (UNPAID | PARTIAL | PAID | CANCELLED), due_date, created_at, updated_at.
interface SalesInvoiceDTO {
  id: string;
  tenant_id: string;
  sales_order_id: string | null;
  invoice_number: string;
  amount: string;
  status: string;
  due_date: string;
  created_at: string;
  updated_at: string;
}

export type InvoiceStatus = 'UNPAID' | 'PARTIAL' | 'PAID' | 'CANCELLED';

export interface SalesInvoice {
  id: string;
  invoiceNumber: string;
  orderId: string | null;
  amount: number;
  status: InvoiceStatus;
  dueDate: string;
  createdAt: string;
}

export interface CreateSalesInvoiceInput {
  sales_order_id?: string;
  invoice_number: string;
  amount: number;
  due_date: string;
}

function toDisplayInvoice(dto: SalesInvoiceDTO): SalesInvoice {
  return {
    id: dto.id,
    invoiceNumber: dto.invoice_number,
    orderId: dto.sales_order_id,
    amount: Number(dto.amount),
    status: dto.status as InvoiceStatus,
    dueDate: dto.due_date,
    createdAt: dto.created_at,
  };
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(await extractApiErrorMessage(response, `Request failed: ${response.status}`));
  }
  return response.json() as Promise<T>;
}

export function useSalesInvoices() {
  const [invoices, setInvoices] = useState<SalesInvoice[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await fetch('/api/v1/sales-invoices', { credentials: 'include' }).then((res) =>
        handleResponse<SalesInvoiceDTO[] | null>(res)
      );
      setInvoices((data ?? []).map(toDisplayInvoice));
      setError(null);
    } catch (err) {
      console.error('Failed to fetch sales invoices', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch sales invoices');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const createSalesInvoice = useCallback(
    async (input: CreateSalesInvoiceInput) => {
    const dto = await fetch('/api/v1/sales-invoices', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    }).then((res) => handleResponse<SalesInvoiceDTO>(res));
      await refresh();
      return toDisplayInvoice(dto);
    },
    [refresh]
  );

  return { invoices, loading, error, refresh, createSalesInvoice };
}
