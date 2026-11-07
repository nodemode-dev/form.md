const ALPHABET = "abcdefghijklmnopqrstuvwxyz234567";

export function pid(name: string): string {
  let a = 2166136261;
  for (let i = 0; i < name.length; i++) a = Math.imul(a ^ name.charCodeAt(i), 16777619) >>> 0;
  let out = "";
  for (let i = 0; i < 26; i++) {
    a ^= a << 13;
    a >>>= 0;
    a ^= a >>> 17;
    a ^= a << 5;
    a >>>= 0;
    out += ALPHABET[a & 31];
  }
  return out;
}

export function fresh(prefix: string): string {
  return pid(`${prefix}:${Date.now()}:${Math.random()}`);
}
