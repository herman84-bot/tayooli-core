"use client";

import React, { useState } from 'react';
import { z } from 'zod';
import { Printer } from 'lucide-react';
import { useSalesInvoices } from '@/hooks/useSalesInvoices';
import type { InvoiceStatus, SalesInvoice } from '@/hooks/useSalesInvoices';
import styles from '@/app/(app)/o2c.module.css';
import { formatCurrency } from '@/lib/currency';
import { PrintSalesInvoice } from '@/components/sales/PrintSalesInvoice';

const STATUS_LABELS: Record<InvoiceStatus, string> = {
  UNPAID: 'Unpaid',
  PARTIAL: 'Partial',
  PAID: 'Paid',
  CANCELLED: 'Cancelled',
};

const salesInvoiceSchema = z.object({
  invoiceNumber: z.string().trim().min(1, 'Invoice number is required.'),
  salesOrderId: z
    .string()
    .trim()
    .refine((value) => value === '' || /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/.test(value), 'Sales Order ID must be a valid UUID.')
    .optional()
    .or(z.literal('').transform(() => undefined)),
  amount: z
    .string()
    .min(1, 'Amount is required.')
    .refine((value) => Number.isFinite(Number(value)), 'Amount must be a valid number.'),
  dueDate: z
    .string()
    .min(1, 'Due date is required.')
    .refine((value) => !Number.isNaN(new Date(value).getTime()), 'Due date must be a valid date.'),
});
type SalesInvoiceInput = z.infer<typeof salesInvoiceSchema>;

interface FieldErrors {
  invoiceNumber?: string;
  salesOrderId?: string;
  amount?: string;
  dueDate?: string;
}

function formatDate(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleDateString('en-CA');
}

export default function SalesInvoicesPage() {
  const { invoices, loading, error, refresh, createSalesInvoice } = useSalesInvoices();
  const [selectedInvoiceForPrint, setSelectedInvoiceForPrint] = useState<SalesInvoice | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [invoiceNumber, setInvoiceNumber] = useState('');
  const [salesOrderId, setSalesOrderId] = useState('');
  const [amount, setAmount] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const openForm = () => {
    setFormError(null);
    setFieldErrors({});
    setFormOpen(true);
  };

  const closeForm = () => {
    setFormOpen(false);
    setFormError(null);
    setFieldErrors({});
  };

  const clearFieldError = (field: keyof FieldErrors) => {
    if (fieldErrors[field]) {
      setFieldErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);
    setFieldErrors({});

    const result = salesInvoiceSchema.safeParse({ invoiceNumber, salesOrderId, amount, dueDate });
    if (!result.success) {
      const errors: FieldErrors = {};
      for (const issue of result.error.issues) {
        const field = issue.path[0] as keyof FieldErrors;
        if (field && !errors[field]) {
          errors[field] = issue.message;
        }
      }
      setFieldErrors(errors);
      return;
    }

    const data = result.data as SalesInvoiceInput;
    setSubmitting(true);
    try {
      await createSalesInvoice({
        invoice_number: data.invoiceNumber,
        sales_order_id: data.salesOrderId ?? undefined,
        amount: Number(data.amount),
        due_date: new Date(data.dueDate).toISOString(),
      });
      setInvoiceNumber('');
      setSalesOrderId('');
      setAmount('');
      setDueDate('');
      closeForm();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create invoice.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.title}>Sales Invoices</h1>
        <button className={styles.addButton} onClick={openForm}>
          + New Invoice
        </button>
      </div>

      {formOpen && (
        <form className={styles.formCard} onSubmit={handleSubmit} noValidate>
          <h2 className={styles.formTitle}>New Sales Invoice</h2>
          <label className={styles.formLabel}>
            Invoice Number *
            <input
              className={styles.formInput}
              type="text"
              value={invoiceNumber}
              onChange={(e) => { setInvoiceNumber(e.target.value); clearFieldError('invoiceNumber'); }}
              placeholder="e.g. SI-2026-0001"
            />
            {fieldErrors.invoiceNumber && <p className={styles.formError}>{fieldErrors.invoiceNumber}</p>}
          </label>
          <label className={styles.formLabel}>
            Sales Order ID
            <input
              className={styles.formInput}
              type="text"
              value={salesOrderId}
              onChange={(e) => { setSalesOrderId(e.target.value); clearFieldError('salesOrderId'); }}
              placeholder="Optional UUID"
            />
            {fieldErrors.salesOrderId && <p className={styles.formError}>{fieldErrors.salesOrderId}</p>}
          </label>
          <label className={styles.formLabel}>
            Amount *
            <input
              className={styles.formInput}
              type="number"
              min="0"
              step="0.01"
              value={amount}
              onChange={(e) => { setAmount(e.target.value); clearFieldError('amount'); }}
              placeholder="0.00"
            />
            {fieldErrors.amount && <p className={styles.formError}>{fieldErrors.amount}</p>}
          </label>
          <label className={styles.formLabel}>
            Due Date *
            <input
              className={styles.formInput}
              type="date"
              value={dueDate}
              onChange={(e) => { setDueDate(e.target.value); clearFieldError('dueDate'); }}
            />
            {fieldErrors.dueDate && <p className={styles.formError}>{fieldErrors.dueDate}</p>}
          </label>
          {formError && <p className={styles.formError}>{formError}</p>}
          <div className={styles.formActions}>
            <button
              type="button"
              className={styles.formCancel}
              onClick={closeForm}
              disabled={submitting}
            >
              Cancel
            </button>
            <button
              type="submit"
              className={styles.formSubmit}
              disabled={submitting}
            >
              {submitting ? 'Creating...' : 'Create Invoice'}
            </button>
          </div>
        </form>
      )}

      <div className={styles.tableContainer}>
        {loading ? (
          <div className={styles.loadingState}>
            <div className={styles.spinner}></div>
            <p>Loading sales invoices...</p>
          </div>
        ) : error ? (
          <div className={styles.emptyState}>
            <p style={{ marginBottom: '0.75rem', fontWeight: 500, color: '#334155' }}>
              {error === 'backend unreachable'
                ? 'Tidak dapat terhubung ke server backend.'
                : error.startsWith('{')
                ? 'Terjadi kendala saat memuat faktur penjualan.'
                : error}
            </p>
            <button
              onClick={() => refresh()}
              className={styles.createButton}
              style={{ fontSize: '0.875rem', padding: '0.5rem 1rem' }}
            >
              Coba Lagi
            </button>
          </div>
        ) : invoices.length === 0 ? (
          <div className={styles.emptyState}>No sales invoices found.</div>
        ) : (
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.th}>Invoice ID</th>
                <th className={styles.th}>Order ID</th>
                <th className={styles.th}>Due Date</th>
                <th className={styles.th}>Amount</th>
                <th className={styles.th}>Status</th>
                <th className={styles.th}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {invoices.map((invoice) => {
                let statusClass = styles.statusUnpaid;
                if (invoice.status === 'PAID') statusClass = styles.statusPaid;
                if (invoice.status === 'CANCELLED') statusClass = styles.statusOverdue;

                return (
                  <tr key={invoice.id} className={styles.tr}>
                    <td className={styles.td}>
                      <strong>{invoice.invoiceNumber}</strong>
                    </td>
                    <td className={styles.td}>
                      {invoice.orderId ? invoice.orderId.slice(0, 8) : '—'}
                    </td>
                    <td className={styles.td}>{formatDate(invoice.dueDate)}</td>
                    <td className={styles.td}>
                      {formatCurrency(invoice.amount)}
                    </td>
                    <td className={styles.td}>
                      <span className={`${styles.statusBadge} ${statusClass}`}>
                        {STATUS_LABELS[invoice.status]}
                      </span>
                    </td>
                    <td className={styles.td}>
                      <button
                        type="button"
                        onClick={() => setSelectedInvoiceForPrint(invoice)}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg border border-slate-200 bg-white text-slate-700 hover:bg-blue-50 hover:text-blue-600 hover:border-blue-200 transition shadow-sm"
                        title="Cetak Faktur Penjualan (A4)"
                      >
                        <Printer className="h-3.5 w-3.5 text-blue-600" />
                        Cetak Faktur
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* MODAL PRATINJAU CETAK FAKTUR PENJUALAN (A4) */}
      {selectedInvoiceForPrint && (
        <PrintSalesInvoice
          invoice={selectedInvoiceForPrint}
          onClose={() => setSelectedInvoiceForPrint(null)}
        />
      )}
    </div>
  );
}
