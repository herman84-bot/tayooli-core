"use client";

import React, { useState } from 'react';
import { z } from 'zod';
import { useCustomers } from '@/hooks/useCustomers';
import styles from '@/app/(app)/o2c.module.css';

const customerSchema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(255, 'Name must be 255 characters or fewer'),
  email: z.string().trim().email('Enter a valid email address').max(255, 'Email must be 255 characters or fewer'),
  phone: z.string().trim().max(50, 'Phone must be 50 characters or fewer').optional().or(z.literal('').transform(() => undefined)),
  address: z.string().trim().max(1000, 'Address must be 1000 characters or fewer').optional().or(z.literal('').transform(() => undefined)),
});
type CustomerInput = z.infer<typeof customerSchema>;

interface FieldErrors {
  name?: string;
  email?: string;
  phone?: string;
  address?: string;
}

export default function CustomersPage() {
  const { customers, loading, error, createCustomer, refresh } = useCustomers();

  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [address, setAddress] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [submitting, setSubmitting] = useState(false);

  const openForm = () => {
    setFormError(null);
    setFieldErrors({});
    setShowForm(true);
  };

  const closeForm = () => {
    setShowForm(false);
    setName('');
    setEmail('');
    setPhone('');
    setAddress('');
    setFormError(null);
    setFieldErrors({});
  };

  function clearFieldError(field: keyof FieldErrors) {
    if (fieldErrors[field]) {
      setFieldErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setFieldErrors({});

    const result = customerSchema.safeParse({ name, email, phone, address });
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

    const data = result.data as CustomerInput;
    setSubmitting(true);
    try {
      await createCustomer({
        name: data.name,
        email: data.email,
        phone: data.phone ?? '',
        address: data.address ?? '',
      });
      closeForm();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create customer');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.title}>Customers</h1>
        <button className={styles.addButton} onClick={openForm}>+ New Customer</button>
      </div>

      {showForm && (
        <form className={styles.formCard} onSubmit={handleSubmit} noValidate>
          <h2 className={styles.formTitle}>New Customer</h2>

          <label className={styles.formLabel}>
            Name <span className={styles.formRequired}>*</span>
            <input
              className={styles.formInput}
              value={name}
              onChange={(e) => { setName(e.target.value); clearFieldError('name'); }}
              placeholder="PT Maju Jaya"
            />
            {fieldErrors.name && <p className={styles.formError}>{fieldErrors.name}</p>}
          </label>

          <label className={styles.formLabel}>
            Email <span className={styles.formRequired}>*</span>
            <input
              type="email"
              className={styles.formInput}
              value={email}
              onChange={(e) => { setEmail(e.target.value); clearFieldError('email'); }}
              placeholder="contact@majujaya.com"
            />
            {fieldErrors.email && <p className={styles.formError}>{fieldErrors.email}</p>}
          </label>

          <label className={styles.formLabel}>
            Phone
            <input
              className={styles.formInput}
              value={phone}
              onChange={(e) => { setPhone(e.target.value); clearFieldError('phone'); }}
              placeholder="+62 812 3456 7890"
            />
            {fieldErrors.phone && <p className={styles.formError}>{fieldErrors.phone}</p>}
          </label>

          <label className={styles.formLabel}>
            Address
            <input
              className={styles.formInput}
              value={address}
              onChange={(e) => { setAddress(e.target.value); clearFieldError('address'); }}
              placeholder="Jl. Sudirman No. 1, Jakarta"
            />
            {fieldErrors.address && <p className={styles.formError}>{fieldErrors.address}</p>}
          </label>

          {formError && <p className={styles.formError}>{formError}</p>}

          <div className={styles.formActions}>
            <button type="button" className={styles.formCancel} onClick={closeForm} disabled={submitting}>
              Cancel
            </button>
            <button type="submit" className={styles.formSubmit} disabled={submitting}>
              {submitting ? 'Creating...' : 'Create Customer'}
            </button>
          </div>
        </form>
      )}

      <div className={styles.tableContainer}>
        {loading ? (
          <div className={styles.loadingState}>
            <div className={styles.spinner}></div>
            <p>Loading customers...</p>
          </div>
        ) : error ? (
          <div className={styles.errorState} role="alert">
            Failed to load customers: {error}
            <button className={styles.retryButton} onClick={refresh}>Retry</button>
          </div>
        ) : customers.length === 0 ? (
          <div className={styles.emptyState}>No customers found.</div>
        ) : (
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.th}>ID</th>
                <th className={styles.th}>Name</th>
                <th className={styles.th}>Email</th>
                <th className={styles.th}>Phone</th>
                <th className={styles.th}>Address</th>
              </tr>
            </thead>
            <tbody>
              {customers.map((customer) => (
                <tr key={customer.id} className={styles.tr}>
                  <td className={styles.td}><strong>{customer.id}</strong></td>
                  <td className={styles.td}>{customer.name}</td>
                  <td className={styles.td}>{customer.email}</td>
                  <td className={styles.td}>{customer.phone ?? '—'}</td>
                  <td className={styles.td}>{customer.address ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
