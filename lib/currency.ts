/**
 * Centralized currency formatting.
 *
 * Default currency is IDR (Indonesian Rupiah).
 * Override at any call site by passing a different ISO 4217 code.
 *
 * Usage:
 *   formatCurrency(1500000)               // Rp 1.500.000
 *   formatCurrency(299000, { currency: 'USD' })  // $299.00
 *   formatCurrency(1500000, { compact: true })   // Rp1,5Jt
 */

const DEFAULT_CURRENCY = "IDR";
const DEFAULT_LOCALE = "id-ID";

interface CurrencyOptions {
  /** ISO 4217 currency code. Defaults to IDR. */
  currency?: string;
  /** Locale for number formatting. Defaults to "id-ID". */
  locale?: string;
  /** Maximum fraction digits. Defaults to 0 for IDR, 2 for others. */
  maximumFractionDigits?: number;
  /** Compact notation (Rp1,5Jt, $1.5M). */
  compact?: boolean;
}

export function formatCurrency(
  value: number | string | null | undefined,
  options?: CurrencyOptions
): string {
  const num = typeof value === "string" ? parseFloat(value) : (value ?? 0);
  if (!Number.isFinite(num)) return "—";

  const currency = options?.currency ?? DEFAULT_CURRENCY;
  const locale = options?.locale ?? DEFAULT_LOCALE;
  const maxFraction =
    options?.maximumFractionDigits ?? (currency === "IDR" ? 0 : 2);

  if (options?.compact) {
    return compactCurrency(num, currency, locale);
  }

  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
    maximumFractionDigits: maxFraction,
  }).format(num);
}

function compactCurrency(
  num: number,
  currency: string,
  locale: string
): string {
  const abs = Math.abs(num);
  const sign = num < 0 ? "-" : "";

  // Use locale-appropriate compact suffixes
  const isIDR = currency === "IDR";
  const suffix = isIDR ? "jt" : "M";

  if (abs >= 1_000_000_000) {
    const val = abs / 1_000_000_000;
    return `${sign}${formatCompact(val, locale)}${isIDR ? "M" : "B"}`;
  }
  if (abs >= 1_000_000) {
    const val = abs / 1_000_000;
    return `${sign}${formatCompact(val, locale)}${suffix}`;
  }
  if (abs >= 1_000) {
    const val = abs / 1_000;
    return `${sign}${formatCompact(val, locale)}${isIDR ? "rb" : "K"}`;
  }

  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
    maximumFractionDigits: 0,
  }).format(num);
}

function formatCompact(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, {
    maximumFractionDigits: 1,
  }).format(value);
}

/**
 * Short label for a currency code (for UI display).
 */
export function currencyLabel(currency: string = DEFAULT_CURRENCY): string {
  const labels: Record<string, string> = {
    IDR: "Rupiah",
    USD: "US Dollar",
    EUR: "Euro",
    MYR: "Ringgit",
    THB: "Baht",
    PHP: "Peso",
    SGD: "Singapore Dollar",
    JPY: "Yen",
    CNY: "Yuan",
    AUD: "Australian Dollar",
  };
  return labels[currency] ?? currency;
}

const ONES = [
  "",
  "Satu",
  "Dua",
  "Tiga",
  "Empat",
  "Lima",
  "Enam",
  "Tujuh",
  "Delapan",
  "Sembilan",
  "Sepuluh",
  "Sebelas",
];

function toWords(num: number): string {
  if (num === 0) return "";
  if (num < 12) return ONES[num];
  if (num < 20) return `${toWords(num - 10)} Belas`;
  if (num < 100) return `${toWords(Math.floor(num / 10))} Puluh ${toWords(num % 10)}`.trim();
  if (num < 200) return `Seratus ${toWords(num - 100)}`.trim();
  if (num < 1000) return `${toWords(Math.floor(num / 100))} Ratus ${toWords(num % 100)}`.trim();
  if (num < 2000) return `Seribu ${toWords(num - 1000)}`.trim();
  if (num < 1_000_000) return `${toWords(Math.floor(num / 1000))} Ribu ${toWords(num % 1000)}`.trim();
  if (num < 1_000_000_000) return `${toWords(Math.floor(num / 1_000_000))} Juta ${toWords(num % 1_000_000)}`.trim();
  if (num < 1_000_000_000_000) return `${toWords(Math.floor(num / 1_000_000_000))} Miliar ${toWords(num % 1_000_000_000)}`.trim();
  return `${toWords(Math.floor(num / 1_000_000_000_000))} Triliun ${toWords(num % 1_000_000_000_000)}`.trim();
}

/**
 * Converts a numeric amount to formal Indonesian words with "Rupiah" suffix.
 * Example: 1250000 -> "Satu Juta Dua Ratus Lima Puluh Ribu Rupiah"
 */
export function terbilang(value: number | string | null | undefined): string {
  const num = typeof value === "string" ? parseFloat(value) : (value ?? 0);
  if (!Number.isFinite(num)) return "Nol Rupiah";
  const floored = Math.floor(Math.abs(num));
  if (floored === 0) return "Nol Rupiah";

  const words = toWords(floored).replace(/\s+/g, " ").trim();
  const prefix = num < 0 ? "Minus " : "";
  return `${prefix}${words} Rupiah`;
}
