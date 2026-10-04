import { useState, useEffect, useCallback } from 'react';
import { extractApiErrorMessage } from '@/lib/api/errors';

export interface SalesOrder {
  id: string;
  customer_id: string;
  customer_name?: string;
  order_number: string;
  total_amount: number;
  status: string;
  created_at: string;
}

export interface CreateSalesOrderInput {
  customer_id: string;
  total_amount: number;
  order_number?: string;
}

export function useSalesOrders() {
  const [orders, setOrders] = useState<SalesOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchOrders = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/v1/sales-orders', { credentials: 'include' });
      if (!response.ok) {
        throw new Error(await extractApiErrorMessage(response, `Request failed: ${response.status}`));
      }
      const data = await response.json();
      setOrders(data ?? []);
    } catch (err) {
      console.error('Failed to fetch sales orders', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch sales orders');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOrders();
  }, [fetchOrders]);

  const createSalesOrder = useCallback(
    async (input: CreateSalesOrderInput) => {
    const response = await fetch('/api/v1/sales-orders', {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    });
      if (!response.ok) {
        throw new Error(await extractApiErrorMessage(response, `Request failed: ${response.status}`));
      }
      const created = (await response.json()) as SalesOrder;
      setOrders((prev) => [created, ...prev]);
      return created;
    },
    []
  );

  return { orders, loading, error, createSalesOrder, refresh: fetchOrders };
}
