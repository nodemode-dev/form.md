const FENCE = "```";

export const WELCOME_MD = [
  "This is a live form.md page. Everything you type is edited as blocks by the same ProseMirror schema the app uses, and nothing leaves this tab.",
  "",
  "> [!TIP]",
  "> Markdown shortcuts work as you type: start a line with `#`, `-`, `1.`, `[]` or `>` and the block changes.",
  "",
  "## Try these",
  "",
  "- [x] Open this page in the editor",
  "- [ ] Select text and press Ctrl or Cmd with B, I or K for bold, italic and a link",
  "- [ ] Use Outline to jump between headings",
  "- [ ] Use Find and replace on this page",
  "- [ ] Export the page as Markdown",
  "- [ ] Open the Trail survey database and switch to the board",
  "- [ ] Drop your own .md or .csv file onto the window",
  "",
  "## Pipe tables become databases",
  "",
  "Pages can hold plain tables. The Table to database tool turns the first table on a page into a database you can sort, filter and group.",
  "",
  "| Station | Elevation m | Open |",
  "| --- | --- | --- |",
  "| Trailhead | 410 | yes |",
  "| Fern Gully | 520 | yes |",
  "| Summit hut | 905 | no |",
  "| Saddle | 760 | yes |",
  "",
  "## Code and quotes",
  "",
  `${FENCE}ts`,
  "const rows = table.filter((r) => r.open).map((r) => r.station);",
  FENCE,
  "",
  "> The page you are reading is stored as a Yjs document in this tab and written back to Markdown by the real form.md Markdown writer.",
  "",
].join("\n");

export const TRAIL_MD = [
  "Survey date: first week of the dry season. Crew of six, two groups.",
  "",
  "> [!NOTE]",
  "> The upper switchbacks hold water after rain. Bring the long-handled drain tools.",
  "",
  "## Findings",
  "",
  "1. Cedar boardwalk: three planks soft, one missing near the creek.",
  "2. Signage: the junction post leans and the arrow has faded.",
  "3. Tread: the last 400 m before the saddle is braided and widening.",
  "4. Drainage: two culverts silted shut.",
  "",
  "## Work plan",
  "",
  "- Replace four planks and re-seat the handrail",
  "  - Order treated timber before the next crew day",
  "- Re-cut the sign arrow and set a new post",
  "- Close the braided tread with brush and rock",
  "- Clear both culverts and photograph before and after",
  "",
  "## Hazard log",
  "",
  "| Where | Hazard | Severity |",
  "| --- | --- | --- |",
  "| Creek crossing | Slick planks | High |",
  "| Saddle | Loose scree | Medium |",
  "| Junction | Leaning post | Low |",
  "",
  "See the **Trail survey** database for lengths, conditions and who checked each trail.",
  "",
].join("\n");

export const MEETING_MD = [
  "Attendees: Maya, Tomas, Ines, Kofi.",
  "",
  "## Agenda",
  "",
  "1. Autumn crew calendar",
  "2. Tool inventory",
  "3. Volunteer training day",
  "",
  "## Decisions",
  "",
  "- Crew days move to the second Saturday of each month.",
  "- Tool library gets a sign-out sheet.",
  "- Training day is open to new volunteers with no prior experience.",
  "",
  "## Actions",
  "",
  "- [ ] Maya: publish the calendar",
  "- [ ] Tomas: count the pulaskis and rock bars",
  "- [x] Ines: book the hall for the training day",
  "- [ ] Kofi: draft the safety briefing",
  "",
].join("\n");

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
