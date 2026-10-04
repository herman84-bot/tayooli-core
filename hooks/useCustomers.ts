import { useState, useEffect, useCallback } from 'react';
import { extractApiErrorMessage } from '@/lib/api/errors';

// Kontrak JSON backend Go (internal/domain/customer.go). Backend TIDAK
// mengirim field `status` — hanya id, name, email, phone (opsional),
// address (opsional), created_at, updated_at.
export interface Customer {
  id: string;
  name: string;
  email: string;
  phone?: string;
  address?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCustomerInput {
  name: string;
  email: string;
  phone?: string;
  address?: string;
}

export function useCustomers() {
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCustomers = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/v1/customers', { credentials: 'include' });
      if (!response.ok) {
        throw new Error(await extractApiErrorMessage(response, `Request failed: ${response.status}`));
      }
      const data = (await response.json()) as Customer[] | null;
      setCustomers(data ?? []);
    } catch (err) {
      console.error('Failed to fetch customers', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch customers');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCustomers();
  }, [fetchCustomers]);

  // Buat customer baru via proxy /api/customers. Melempar Error dengan
  // pesan yang bisa ditampilkan langsung ke user (error backend diteruskan).
  const createCustomer = useCallback(async (input: CreateCustomerInput) => {
    const response = await fetch('/api/v1/customers', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    });
    if (!response.ok) {
      throw new Error(await extractApiErrorMessage(response, `Request failed: ${response.status}`));
    }
    const created = (await response.json()) as Customer;
    setCustomers((prev) => [created, ...prev]);
    return created;
  }, []);

  return { customers, loading, error, createCustomer, refresh: fetchCustomers };
}
