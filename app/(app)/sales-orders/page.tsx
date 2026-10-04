"use client";

import React, { useState, FormEvent } from 'react';
import { z } from 'zod';
import { useSalesOrders, type SalesOrder } from '@/hooks/useSalesOrders';
import { useCustomers } from '@/hooks/useCustomers';
import styles from '@/app/(app)/o2c.module.css';
import { formatCurrency } from '@/lib/currency';

const salesOrderSchema = z.object({
  customerId: z.string().min(1, 'Please select a customer.'),
  amount: z
    .string()
    .min(1, 'Please enter a total amount.')
    .refine((value) => {
      const parsed = Number(value);
      return Number.isFinite(parsed) && parsed > 0;
    }, 'Please enter a valid total amount greater than zero.'),
});
type SalesOrderInput = z.infer<typeof salesOrderSchema>;

interface FieldErrors {
  customerId?: string;
  amount?: string;
}

function statusClass(status: string, styles: Record<string, string>): string {
  const normalized = status.toLowerCase();
  if (['delivered', 'paid', 'active'].includes(normalized)) return styles.statusDelivered;
  if (normalized === 'shipped') return styles.statusShipped;
  return styles.statusPending;
}

function displayStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-CA');
}

function formatMoney(amount: number): string {
  return formatCurrency(amount);
}

export default function SalesOrdersPage() {
  const { orders, loading, error, createSalesOrder, refresh } = useSalesOrders();
  const { customers, loading: customersLoading } = useCustomers();

  const [showForm, setShowForm] = useState(false);
  const [customerId, setCustomerId] = useState('');
  const [amount, setAmount] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const clearFieldError = (field: keyof FieldErrors) => {
    if (fieldErrors[field]) {
      setFieldErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setFieldErrors({});

    const result = salesOrderSchema.safeParse({ customerId, amount });
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

    const data = result.data as SalesOrderInput;
    setSubmitting(true);
    try {
      await createSalesOrder({ customer_id: data.customerId, total_amount: Number(data.amount) });
      setCustomerId('');
      setAmount('');
      setShowForm(false);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create sales order.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.title}>Sales Orders</h1>
        <button
          className={styles.addButton}
          onClick={() => setShowForm((prev) => !prev)}
          disabled={submitting}
        >
          + New Order
        </button>
      </div>

      {showForm && (
        <div className={styles.formCard}>
          <form onSubmit={handleSubmit} noValidate>
            <label className={styles.label} htmlFor="customer">Customer</label>
            <select
              id="customer"
              className={styles.input}
              value={customerId}
              onChange={(e) => { setCustomerId(e.target.value); clearFieldError('customerId'); }}
              disabled={customersLoading}
            >
              <option value="">
                {customersLoading ? 'Loading customers...' : 'Select a customer'}
              </option>
              {customers.map((customer) => (
                <option key={customer.id} value={customer.id}>
                  {customer.name}
                </option>
              ))}
            </select>
            {fieldErrors.customerId && <p className={styles.formError}>{fieldErrors.customerId}</p>}

            <label className={styles.label} htmlFor="amount">Total Amount</label>
            <input
              id="amount"
              className={styles.input}
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
              value={amount}
              onChange={(e) => { setAmount(e.target.value); clearFieldError('amount'); }}
            />
            {fieldErrors.amount && <p className={styles.formError}>{fieldErrors.amount}</p>}

            {formError && <p className={styles.formError}>{formError}</p>}

            <div className={styles.formActions}>
              <button
                type="button"
                className={styles.cancelButton}
                onClick={() => setShowForm(false)}
                disabled={submitting}
              >
                Cancel
              </button>
              <button type="submit" className={styles.submitButton} disabled={submitting}>
                {submitting ? 'Creating...' : 'Create Order'}
              </button>
            </div>
          </form>
        </div>
      )}

      <div className={styles.tableContainer}>
        {loading ? (
          <div className={styles.loadingState}>
            <div className={styles.spinner}></div>
            <p>Loading sales orders...</p>
          </div>
        ) : error ? (
          <div className={styles.emptyState}>{error}</div>
        ) : orders.length === 0 ? (
          <div className={styles.emptyState}>No sales orders found.</div>
        ) : (
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.th}>Order ID</th>
                <th className={styles.th}>Customer</th>
                <th className={styles.th}>Date</th>
                <th className={styles.th}>Total</th>
                <th className={styles.th}>Status</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((order: SalesOrder) => (
                <tr key={order.id} className={styles.tr}>
                  <td className={styles.td}>
                    <strong>{order.order_number || order.id}</strong>
                  </td>
                  <td className={styles.td}>{order.customer_name ?? order.customer_id}</td>
                  <td className={styles.td}>{formatDate(order.created_at)}</td>
                  <td className={styles.td}>{formatMoney(order.total_amount)}</td>
                  <td className={styles.td}>
                    <span className={`${styles.statusBadge} ${statusClass(order.status, styles)}`}>
                      {displayStatus(order.status)}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
