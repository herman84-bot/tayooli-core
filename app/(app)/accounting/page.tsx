'use client';

import { useState, useEffect } from 'react';
import { formatCurrency } from '@/lib/currency';

type Account = {
  id: string;
  code: string;
  name: string;
  type: string;
  balance: number;
};

export default function AccountingPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Fetch accounts from API
    const fetchAccounts = async () => {
      try {
        const res = await fetch('/api/v1/accounts');
        if (res.ok) {
          const data = await res.json();
          const list = Array.isArray(data) ? data : data?.data ?? [];
          setAccounts(list);
        }
      } catch (error) {
        console.error('Failed to fetch accounts', error);
      } finally {
        setLoading(false);
      }
    };
    fetchAccounts();
  }, []);

  return (
    <div className="p-8 font-sans">
      <h1 className="text-3xl font-bold mb-6 text-gray-800">Chart of Accounts</h1>
      <div className="overflow-x-auto rounded-lg shadow-md">
        <table className="min-w-full bg-white">
          <thead className="bg-gray-100 border-b">
            <tr>
              <th className="py-3 px-6 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Code</th>
              <th className="py-3 px-6 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
              <th className="py-3 px-6 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
              <th className="py-3 px-6 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Balance</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {loading ? (
              <tr>
                <td colSpan={4} className="py-4 text-center text-gray-500">Loading...</td>
              </tr>
            ) : accounts.length === 0 ? (
              <tr>
                <td colSpan={4} className="py-4 text-center text-gray-500">No accounts found.</td>
              </tr>
            ) : (
              accounts.map((account) => (
                <tr key={account.id} className="hover:bg-gray-50 transition-colors">
                  <td className="py-4 px-6 whitespace-nowrap text-sm font-medium text-gray-900">{account.code}</td>
                  <td className="py-4 px-6 whitespace-nowrap text-sm text-gray-700">{account.name}</td>
                  <td className="py-4 px-6 whitespace-nowrap text-sm text-gray-700">
                    <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-blue-100 text-blue-800">
                      {account.type}
                    </span>
                  </td>
                  <td className="py-4 px-6 whitespace-nowrap text-sm text-right text-gray-700">
                    {formatCurrency(account.balance)}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
