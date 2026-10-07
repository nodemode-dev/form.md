const DIGITS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
const BASE = DIGITS.length;

function value(s: string, i: number): number {
  return i >= s.length ? 0 : DIGITS.indexOf(s.charAt(i));
}

function midpoint(a: string, b: string): string {
  if (b === "") {
    const d = value(a, 0);
    if (a === "" || d < BASE - 1) return DIGITS.charAt(Math.floor((d + BASE + 1) / 2));
    return a.charAt(0) + midpoint(a.slice(1), "");
  }
  let i = 0;
  while (value(a, i) === value(b, i)) i++;
  const prefix = b.slice(0, i);
  const lo = value(a, i);
  const hi = value(b, i);
  if (hi - lo > 1) return prefix + DIGITS.charAt(Math.floor((lo + hi) / 2));
  if (b.length > i + 1) return b.slice(0, i + 1);
  return prefix + DIGITS.charAt(lo) + midpoint(a.length > i + 1 ? a.slice(i + 1) : "", "");
}

export function keyBetween(a: string, b: string): string {
  if (b !== "" && a >= b) return a + "V";
  return midpoint(a, b);
}

export function compareKeys(a: string, b: string): number {
  return a < b ? -1 : a > b ? 1 : 0;
}
