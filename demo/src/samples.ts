const ZONES = ["North Ridge", "Lakeside", "Quarry", "Old Mill", "Meadow"];
const STEMS = ["Cedar", "Fern", "Heron", "Alder", "Basalt", "Wren", "Marsh", "Birch", "Copper", "Lantern", "Willow", "Granite"];
const ENDS = ["Loop", "Hollow", "Steps", "Spur", "Run", "Traverse", "Climb", "Path"];
const STATUSES = ["Open", "Open", "Open", "In review", "Needs work", "Closed"];
const CONDITIONS = ["Good", "Good", "Fair", "Fair", "Poor"];
const PEOPLE = ["maya", "tomas", "ines", "kofi", "ada"];

function lcg(seed: number): () => number {
  let s = seed;
  return () => {
    s = (s * 1664525 + 1013904223) % 4294967296;
    return s / 4294967296;
  };
}

export function trailCsv(): string {
  const rnd = lcg(48);
  const pick = <T>(xs: T[]): T => xs[Math.floor(rnd() * xs.length)] as T;
  const used = new Set<string>();
  const lines = ["Trail,Zone,Status,Condition,Length km,Elevation gain m,Surveyed,Volunteers,Reviewed,Contact"];
  while (lines.length <= 26) {
    const name = `${pick(STEMS)} ${pick(ENDS)}`;
    if (used.has(name)) continue;
    used.add(name);
    const month = 1 + Math.floor(rnd() * 9);
    const day = 1 + Math.floor(rnd() * 27);
    const date = `2026-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
    const length = (1 + rnd() * 11).toFixed(1);
    const gain = Math.round(30 + rnd() * 640);
    const crew = 1 + Math.floor(rnd() * 8);
    const reviewed = rnd() > 0.45 ? "yes" : "no";
    lines.push([name, pick(ZONES), pick(STATUSES), pick(CONDITIONS), length, String(gain), date, String(crew), reviewed, `${pick(PEOPLE)}@example.org`].join(","));
  }
  return `${lines.join("\n")}\n`;
}
