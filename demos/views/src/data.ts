import type { Property } from "../../../web/src/engine/types";

export const COL_MISSIONS = "col_missions";
export const COL_ASTRONAUTS = "col_astronauts";
export const COL_PROGRAMS = "col_programs";

export interface SeedRow {
  id: string;
  title: string;
  cover?: string;
  vals: Record<string, unknown>;
}

type Mission = {
  key: string;
  title: string;
  designation: string;
  program: string;
  vehicle: string;
  traj: string;
  launch: string;
  splash: string;
  time: string;
  orbits?: number;
  recovery: string;
  crew: string[];
  note: string;
};

const MERCURY = "Mercury";
const GEMINI = "Gemini";
const APOLLO = "Apollo";

const MISSIONS: Mission[] = [
  { key: "mr-3", title: "Freedom 7", designation: "MR-3", program: MERCURY, vehicle: "redstone", traj: "sub", launch: "1961-05-05", splash: "1961-05-05", time: "0:15:22", recovery: "atl", crew: ["Alan Shepard"], note: "First manned suborbital." },
  { key: "mr-4", title: "Liberty Bell 7", designation: "MR-4", program: MERCURY, vehicle: "redstone", traj: "sub", launch: "1961-07-21", splash: "1961-07-21", time: "0:15:37", recovery: "atl", crew: ["Gus Grissom"], note: "Second manned suborbital." },
  { key: "ma-6", title: "Friendship 7", designation: "MA-6", program: MERCURY, vehicle: "atlas", traj: "earth", launch: "1962-02-20", splash: "1962-02-20", time: "4:55:23", orbits: 3, recovery: "atl", crew: ["John Glenn"], note: "First orbital." },
  { key: "ma-7", title: "Aurora 7", designation: "MA-7", program: MERCURY, vehicle: "atlas", traj: "earth", launch: "1962-05-24", splash: "1962-05-24", time: "4:56:05", orbits: 3, recovery: "atl", crew: ["Scott Carpenter"], note: "Three orbits." },
  { key: "ma-8", title: "Sigma 7", designation: "MA-8", program: MERCURY, vehicle: "atlas", traj: "earth", launch: "1962-10-03", splash: "1962-10-03", time: "9:13:11", orbits: 6, recovery: "pac", crew: ["Wally Schirra"], note: "Six orbits." },
  { key: "ma-9", title: "Faith 7", designation: "MA-9", program: MERCURY, vehicle: "atlas", traj: "earth", launch: "1963-05-15", splash: "1963-05-16", time: "34:19:49", orbits: 22, recovery: "pac", crew: ["Gordon Cooper"], note: "Twenty-two orbits." },
  { key: "gt-3", title: "Gemini 3", designation: "GT-3", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1965-03-23", splash: "1965-03-23", time: "4:52:31", orbits: 3, recovery: "atl", crew: ["Gus Grissom", "John Young"], note: "First two-man orbital." },
  { key: "gt-4", title: "Gemini 4", designation: "GT-4", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1965-06-03", splash: "1965-06-07", time: "97:56:12", orbits: 62, recovery: "atl", crew: ["James McDivitt", "Ed White"], note: "First extravehicular activity." },
  { key: "gt-5", title: "Gemini 5", designation: "GT-5", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1965-08-21", splash: "1965-08-29", time: "190:55:14", orbits: 120, recovery: "atl", crew: ["Gordon Cooper", "Pete Conrad"], note: "Long-duration mission." },
  { key: "gt-7", title: "Gemini 7", designation: "GT-7", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1965-12-04", splash: "1965-12-18", time: "330:35:01", orbits: 206, recovery: "atl", crew: ["Frank Borman", "Jim Lovell"], note: "Long-duration, rendezvous." },
  { key: "gt-6a", title: "Gemini 6A", designation: "GT-6A", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1965-12-15", splash: "1965-12-16", time: "25:51:24", orbits: 16, recovery: "atl", crew: ["Wally Schirra", "Tom Stafford"], note: "Rendezvous." },
  { key: "gt-8", title: "Gemini 8", designation: "GT-8", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1966-03-16", splash: "1966-03-16", time: "10:41:26", orbits: 6.6, recovery: "pac", crew: ["Neil Armstrong", "David Scott"], note: "Rendezvous, first docking." },
  { key: "gt-9a", title: "Gemini 9A", designation: "GT-9A", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1966-06-03", splash: "1966-06-06", time: "72:20:50", orbits: 45, recovery: "atl", crew: ["Tom Stafford", "Gene Cernan"], note: "Rendezvous, extravehicular activity." },
  { key: "gt-10", title: "Gemini 10", designation: "GT-10", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1966-07-18", splash: "1966-07-21", time: "70:46:39", orbits: 43, recovery: "atl", crew: ["John Young", "Michael Collins"], note: "Rendezvous, dock, extravehicular activity, altitude record (475 mi.)." },
  { key: "gt-11", title: "Gemini 11", designation: "GT-11", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1966-09-12", splash: "1966-09-15", time: "71:17:08", orbits: 44, recovery: "atl", crew: ["Pete Conrad", "Richard Gordon"], note: "Rendezvous, dock, extravehicular activity, altitude record (853 mi.)." },
  { key: "gt-12", title: "Gemini 12", designation: "GT-12", program: GEMINI, vehicle: "titan", traj: "earth", launch: "1966-11-11", splash: "1966-11-15", time: "94:34:31", orbits: 59, recovery: "atl", crew: ["Jim Lovell", "Buzz Aldrin"], note: "Rendezvous, dock, extravehicular activity." },
  { key: "ap-7", title: "Apollo 7", designation: "AS-205", program: APOLLO, vehicle: "sib", traj: "earth", launch: "1968-10-11", splash: "1968-10-22", time: "260:09:03", orbits: 163, recovery: "atl", crew: ["Wally Schirra", "Donn Eisele", "Walter Cunningham"], note: "CSM manned flight demonstration." },
  { key: "ap-8", title: "Apollo 8", designation: "AS-503", program: APOLLO, vehicle: "sv", traj: "lunar", launch: "1968-12-21", splash: "1968-12-27", time: "147:00:42", recovery: "pac", crew: ["Frank Borman", "Jim Lovell", "William Anders"], note: "CSM manned flight demonstration." },
  { key: "ap-9", title: "Apollo 9", designation: "AS-504", program: APOLLO, vehicle: "sv", traj: "earth", launch: "1969-03-03", splash: "1969-03-13", time: "241:00:54", orbits: 151, recovery: "atl", crew: ["James McDivitt", "David Scott", "Russell Schweickart"], note: "Lunar module manned flight demonstration." },
  { key: "ap-10", title: "Apollo 10", designation: "AS-505", program: APOLLO, vehicle: "sv", traj: "lunar", launch: "1969-05-18", splash: "1969-05-26", time: "192:03:23", recovery: "pac", crew: ["Tom Stafford", "John Young", "Gene Cernan"], note: "Lunar module manned flight demonstration." },
  { key: "ap-11", title: "Apollo 11", designation: "AS-506", program: APOLLO, vehicle: "sv", traj: "land", launch: "1969-07-16", splash: "1969-07-24", time: "195:18:35", recovery: "pac", crew: ["Neil Armstrong", "Buzz Aldrin", "Michael Collins"], note: "Manned lunar landing demonstration." },
  { key: "ap-12", title: "Apollo 12", designation: "AS-507", program: APOLLO, vehicle: "sv", traj: "land", launch: "1969-11-14", splash: "1969-11-24", time: "244:36:25", recovery: "pac", crew: ["Pete Conrad", "Alan Bean", "Richard Gordon"], note: "Precision manned lunar landing demonstration and systematic lunar exploration." },
  { key: "ap-13", title: "Apollo 13", designation: "AS-508", program: APOLLO, vehicle: "sv", traj: "abort", launch: "1970-04-11", splash: "1970-04-17", time: "142:54:41", recovery: "pac", crew: ["Jim Lovell", "Fred Haise", "Jack Swigert"], note: "Landing aborted." },
  { key: "ap-14", title: "Apollo 14", designation: "AS-509", program: APOLLO, vehicle: "sv", traj: "land", launch: "1971-01-31", splash: "1971-02-09", time: "216:01:58", recovery: "pac", crew: ["Alan Shepard", "Edgar Mitchell", "Stuart Roosa"], note: "Precision manned lunar landing demonstration and systematic lunar exploration." },
  { key: "ap-15", title: "Apollo 15", designation: "AS-510", program: APOLLO, vehicle: "sv", traj: "land", launch: "1971-07-26", splash: "1971-08-07", time: "295:11:53", recovery: "pac", crew: ["David Scott", "James Irwin", "Alfred Worden"], note: "Extensive scientific investigation of the Moon on the surface and from lunar orbit." },
  { key: "ap-16", title: "Apollo 16", designation: "AS-511", program: APOLLO, vehicle: "sv", traj: "land", launch: "1972-04-16", splash: "1972-04-27", time: "265:51:05", recovery: "pac", crew: ["John Young", "Charles Duke", "Ken Mattingly"], note: "Extensive scientific investigation of the Moon on the surface and from lunar orbit." },
  { key: "ap-17", title: "Apollo 17", designation: "AS-512", program: APOLLO, vehicle: "sv", traj: "land", launch: "1972-12-07", splash: "1972-12-19", time: "301:51:59", recovery: "pac", crew: ["Gene Cernan", "Harrison Schmitt", "Ronald Evans"], note: "Extensive scientific investigation of the Moon on the surface and from lunar orbit." },
];

function hours(t: string): number {
  const [h, m, s] = t.split(":").map(Number) as [number, number, number];
  return Math.round((h + m / 60 + s / 3600) * 10000) / 10000;
}

const slug = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, "_");

export const PROGRAM_NOTES: Record<string, string> = {
  [MERCURY]: "One astronaut per capsule: the first American crewed flights, two suborbital and four orbital.",
  [GEMINI]: "Two astronauts per spacecraft: long-duration flights, rendezvous, docking and spacewalks.",
  [APOLLO]: "Three astronauts per spacecraft: the crewed flights that tested the Moon ships and landed on the Moon.",
};

export function astronautIds(): { id: string; name: string }[] {
  const seen = new Map<string, string>();
  for (const m of MISSIONS) for (const c of m.crew) seen.set(c, `a_${slug(c)}`);
  return [...seen].map(([name, id]) => ({ id, name }));
}

export function missionRows(): SeedRow[] {
  return MISSIONS.map((m) => ({
    id: `m_${m.key}`,
    title: m.title,
    vals: {
      designation: m.designation,
      program: [`p_${slug(m.program)}`],
      crew: m.crew.map((c) => `a_${slug(c)}`),
      flight: m.launch === m.splash ? { start: m.launch } : { start: m.launch, end: m.splash },
      hours: hours(m.time),
      ...(m.orbits ? { orbits: m.orbits } : {}),
      traj: m.traj,
      vehicle: m.vehicle,
      recovery: m.recovery,
      highlight: m.note,
      photo: [{ id: m.key, name: `${m.key}.webp`, mime: "image/webp" }],
    },
  }));
}

export function astronautRows(): SeedRow[] {
  return astronautIds().map((a) => ({
    id: a.id,
    title: a.name,
    vals: {
      flights: MISSIONS.filter((m) => m.crew.includes(a.name)).map((m) => `m_${m.key}`),
    },
  }));
}

export function programRows(): SeedRow[] {
  return [MERCURY, GEMINI, APOLLO].map((p) => ({
    id: `p_${slug(p)}`,
    title: p,
    vals: {
      about: PROGRAM_NOTES[p],
      missions: MISSIONS.filter((m) => m.program === p).map((m) => `m_${m.key}`),
    },
  }));
}

export const MISSION_SCHEMA: Property[] = [
  { id: "title", name: "Mission", type: "title" },
  { id: "designation", name: "Designation", type: "text" },
  { id: "program", name: "Program", type: "relation", collection: COL_PROGRAMS, pair: "missions", limit: 1 },
  { id: "flight", name: "Flight dates", type: "date" },
  { id: "traj", name: "Trajectory", type: "select", options: [
    { id: "sub", name: "Suborbital", color: "gray" },
    { id: "earth", name: "Earth orbit", color: "blue" },
    { id: "lunar", name: "Lunar orbit", color: "purple" },
    { id: "land", name: "Lunar landing", color: "green" },
    { id: "abort", name: "Landing aborted", color: "red" },
  ] },
  { id: "crew", name: "Crew", type: "relation", collection: COL_ASTRONAUTS, pair: "flights" },
  { id: "size", name: "Crew size", type: "rollup", relation: "crew", target: "title", fn: "count_all" },
  { id: "hours", name: "Flight time (h)", type: "number", precision: 2 },
  { id: "days", name: "Days", type: "formula", expr: 'round(prop("Flight time (h)") / 24, 2)', precision: 2 },
  { id: "crewhours", name: "Crew-hours", type: "formula", expr: 'round(prop("Flight time (h)") * prop("Crew size"), 1)', display: "bar", max: 900, color: "blue" },
  { id: "length", name: "Length", type: "formula", expr: 'ifs(prop("Flight time (h)") < 1, "Minutes", prop("Flight time (h)") < 24, "Hours", prop("Flight time (h)") < 168, "Days", "A week or more")' },
  { id: "ago", name: "Years ago", type: "formula", expr: 'dateBetween(now(), dateStart(prop("Flight dates")), "years")' },
  { id: "orbits", name: "Orbits", type: "number" },
  { id: "vehicle", name: "Launch vehicle", type: "select", options: [
    { id: "redstone", name: "Redstone", color: "orange" },
    { id: "atlas", name: "Atlas", color: "yellow" },
    { id: "titan", name: "Titan II", color: "pink" },
    { id: "sib", name: "Saturn IB", color: "brown" },
    { id: "sv", name: "Saturn V", color: "red" },
  ] },
  { id: "recovery", name: "Recovery", type: "select", options: [
    { id: "atl", name: "Atlantic", color: "blue" },
    { id: "pac", name: "Pacific", color: "green" },
  ] },
  { id: "highlight", name: "Significant event", type: "text" },
  { id: "photo", name: "Photo", type: "files" },
];

export const ASTRONAUT_SCHEMA: Property[] = [
  { id: "title", name: "Astronaut", type: "title" },
  { id: "flights", name: "Flights", type: "relation", collection: COL_MISSIONS, pair: "crew" },
  { id: "count", name: "Missions", type: "rollup", relation: "flights", target: "title", fn: "count_all" },
  { id: "total", name: "Hours flown", type: "rollup", relation: "flights", target: "hours", fn: "sum", precision: 1, display: "bar", max: 700, color: "purple" },
  { id: "first", name: "First launch", type: "rollup", relation: "flights", target: "flight", fn: "earliest" },
  { id: "last", name: "Last splashdown", type: "rollup", relation: "flights", target: "flight", fn: "latest" },
  { id: "span", name: "Career span (years)", type: "formula", expr: 'dateBetween(prop("Last splashdown"), prop("First launch"), "years")' },
  { id: "veteran", name: "Experience", type: "formula", expr: 'ifs(prop("Missions") >= 3, "Veteran", prop("Missions") == 2, "Repeat flyer", "First-timer")' },
];

export const PROGRAM_SCHEMA: Property[] = [
  { id: "title", name: "Program", type: "title" },
  { id: "about", name: "About", type: "text" },
  { id: "missions", name: "Missions", type: "relation", collection: COL_MISSIONS, pair: "program" },
  { id: "count", name: "Flights", type: "rollup", relation: "missions", target: "title", fn: "count_all" },
  { id: "hours", name: "Flight hours", type: "rollup", relation: "missions", target: "hours", fn: "sum", precision: 1, display: "bar", max: 2600, color: "green" },
  { id: "longest", name: "Longest flight (h)", type: "rollup", relation: "missions", target: "hours", fn: "max", precision: 1 },
  { id: "seats", name: "Crew seats", type: "rollup", relation: "missions", target: "size", fn: "sum" },
  { id: "first", name: "First launch", type: "rollup", relation: "missions", target: "flight", fn: "earliest" },
  { id: "last", name: "Last splashdown", type: "rollup", relation: "missions", target: "flight", fn: "latest" },
  { id: "avg", name: "Hours per flight", type: "formula", expr: 'round(prop("Flight hours") / prop("Flights"), 1)', precision: 1 },
  { id: "digest", name: "Summary", type: "formula", expr: 'concat(prop("Program"), ": ", format(prop("Flights")), " flights, ", format(round(prop("Flight hours"))), " hours")' },
];

export const MISSION_IDS = MISSIONS.map((m) => ({ key: m.key, id: `m_${m.key}`, title: m.title }));
