import { describe, expect, it } from "vitest";
import { formatGhanaPhone, isValidGhanaPhone, normalizeGhanaPhone } from "./phone";

describe("formatGhanaPhone", () => {
  it("groups a local number as it is typed", () => {
    expect(formatGhanaPhone("0")).toBe("0");
    expect(formatGhanaPhone("024")).toBe("024");
    expect(formatGhanaPhone("0241234")).toBe("024 123 4");
    expect(formatGhanaPhone("0241234567")).toBe("024 123 4567");
  });

  it("strips stray characters while grouping", () => {
    expect(formatGhanaPhone("024-123-4567")).toBe("024 123 4567");
    expect(formatGhanaPhone("(024) 123 4567")).toBe("024 123 4567");
  });

  it("formats the international form", () => {
    expect(formatGhanaPhone("+233")).toBe("+233");
    expect(formatGhanaPhone("+233241234567")).toBe("+233 24 123 4567");
    expect(formatGhanaPhone("233241234567")).toBe("+233 24 123 4567");
  });

  it("caps the digit count so extra input is ignored", () => {
    expect(formatGhanaPhone("024123456789")).toBe("024 123 4567");
    expect(formatGhanaPhone("+233241234567890")).toBe("+233 24 123 4567");
  });
});

describe("isValidGhanaPhone", () => {
  it("accepts valid local and international numbers", () => {
    expect(isValidGhanaPhone("024 123 4567")).toBe(true);
    expect(isValidGhanaPhone("0541234567")).toBe(true);
    expect(isValidGhanaPhone("+233 24 123 4567")).toBe(true);
    expect(isValidGhanaPhone("233241234567")).toBe(true);
  });

  it("rejects wrong length or implausible prefixes", () => {
    expect(isValidGhanaPhone("024 123 456")).toBe(false); // too short
    expect(isValidGhanaPhone("0141234567")).toBe(false); // bad prefix
    expect(isValidGhanaPhone("+233 14 123 4567")).toBe(false);
    expect(isValidGhanaPhone("")).toBe(false);
  });
});

describe("normalizeGhanaPhone", () => {
  it("canonicalises valid numbers to +233", () => {
    expect(normalizeGhanaPhone("024 123 4567")).toBe("+233241234567");
    expect(normalizeGhanaPhone("+233 24 123 4567")).toBe("+233241234567");
  });

  it("returns trimmed input unchanged when invalid", () => {
    expect(normalizeGhanaPhone("  not a phone ")).toBe("not a phone");
  });
});
