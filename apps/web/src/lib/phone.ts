/**
 * Ghana phone-number helpers. Customers reach the atelier on WhatsApp, so the
 * field accepts the two shapes people actually type — local `0XX XXX XXXX` and
 * international `+233 XX XXX XXXX` — formats them as they type, and validates
 * the digit count and a plausible mobile/landline prefix.
 */

/** Local mobile/landline starts 02/03/05; the national form (no 0) starts 2/3/5. */
const LOCAL_PATTERN = /^0[235]\d{8}$/;
const NATIONAL_PATTERN = /^[235]\d{8}$/;

function isInternational(input: string): boolean {
  return input.trimStart().startsWith("+") || input.replace(/\D/g, "").startsWith("233");
}

/**
 * Formats a Ghana phone number for display as the user types. Groups the digits
 * (`024 123 4567` / `+233 24 123 4567`) and never drops characters the user is
 * mid-way through entering.
 */
export function formatGhanaPhone(input: string): string {
  const nums = input.replace(/\D/g, "");

  if (isInternational(input)) {
    const national = nums.replace(/^233/, "").slice(0, 9);
    const groups = [national.slice(0, 2), national.slice(2, 5), national.slice(5, 9)].filter(Boolean);
    return groups.length > 0 ? `+233 ${groups.join(" ")}` : "+233";
  }

  const local = nums.slice(0, 10);
  return [local.slice(0, 3), local.slice(3, 6), local.slice(6, 10)].filter(Boolean).join(" ");
}

/** True when the input is a plausible Ghanaian phone number. */
export function isValidGhanaPhone(input: string): boolean {
  const nums = input.replace(/\D/g, "");

  if (isInternational(input)) {
    return NATIONAL_PATTERN.test(nums.replace(/^233/, ""));
  }

  return LOCAL_PATTERN.test(nums);
}

/**
 * Canonical international form (`+233241234567`) for storage/sending, or the
 * trimmed input unchanged when it isn't a recognised Ghana number.
 */
export function normalizeGhanaPhone(input: string): string {
  if (!isValidGhanaPhone(input)) return input.trim();

  const nums = input.replace(/\D/g, "");
  const national = isInternational(input) ? nums.replace(/^233/, "") : nums.replace(/^0/, "");
  return `+233${national}`;
}
