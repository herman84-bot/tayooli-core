import { terbilang as terbilangFromCurrency, formatCurrency } from "@/lib/currency";
import { terbilang as terbilangFromLib } from "@/lib/terbilang";

describe("Indonesian Terbilang Accuracy Verification", () => {
  const testCases: [number | string, string][] = [
    [0, "Nol Rupiah"],
    [11, "Sebelas Rupiah"],
    [100, "Seratus Rupiah"],
    [1000, "Seribu Rupiah"],
    [1250000, "Satu Juta Dua Ratus Lima Puluh Ribu Rupiah"],
    [
      1234567890123,
      "Satu Triliun Dua Ratus Tiga Puluh Empat Miliar Lima Ratus Enam Puluh Tujuh Juta Delapan Ratus Sembilan Puluh Ribu Seratus Dua Puluh Tiga Rupiah",
    ],
  ];

  describe("lib/currency.ts - terbilang()", () => {
    testCases.forEach(([input, expected]) => {
      it(`correctly converts ${input} -> "${expected}"`, () => {
        expect(terbilangFromCurrency(input)).toBe(expected);
      });
    });
  });

  describe("lib/terbilang.ts re-export", () => {
    testCases.forEach(([input, expected]) => {
      it(`correctly re-exports and converts ${input} -> "${expected}"`, () => {
        expect(terbilangFromLib(input)).toBe(expected);
      });
    });
  });

  describe("Edge cases & boundaries", () => {
    it("handles null and undefined gracefully", () => {
      expect(terbilangFromCurrency(null)).toBe("Nol Rupiah");
      expect(terbilangFromCurrency(undefined)).toBe("Nol Rupiah");
    });

    it("handles negative numbers with Minus prefix", () => {
      expect(terbilangFromCurrency(-1250000)).toBe("Minus Satu Juta Dua Ratus Lima Puluh Ribu Rupiah");
    });

    it("handles string numeric inputs", () => {
      expect(terbilangFromCurrency("1000")).toBe("Seribu Rupiah");
    });
  });
});
