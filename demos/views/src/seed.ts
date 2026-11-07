import type { CollectionView, Property, ViewConfig } from "../../../web/src/engine/types";
import { ASTRONAUT_SCHEMA, astronautRows, COL_ASTRONAUTS, COL_MISSIONS, COL_PROGRAMS, MISSION_SCHEMA, missionRows, PROGRAM_SCHEMA, programRows, type SeedRow } from "./data";

type SeedView = Omit<CollectionView, "collection_id" | "created_at" | "updated_at" | "sort_key">;

export interface SeedCollection {
  id: string;
  title: string;
  icon: string;
  schema: Property[];
  views: SeedView[];
  rows: SeedRow[];
}

const v = (id: string, name: string, type: CollectionView["type"], config: ViewConfig): SeedView => ({ id, name, type, config });

const show = (all: Property[], visible: string[]): { id: string; hidden: boolean }[] => {
  const order = [...visible, ...all.map((p) => p.id).filter((id) => !visible.includes(id))];
  return order.map((id) => ({ id, hidden: !visible.includes(id) }));
};

const missionViews = (): SeedView[] => [
  v("v_m_table", "All flights", "table", {
    props: show(MISSION_SCHEMA, ["title", "program", "flight", "traj", "vehicle", "crew", "size", "hours", "days"]),
    sorts: [{ prop: "flight", dir: "asc" }],
    calcs: { hours: "sum", size: "sum" },
  }),
  v("v_m_board", "By trajectory", "board", {
    props: show(MISSION_SCHEMA, ["program", "flight", "hours"]),
    group: { prop: "traj" },
    sorts: [{ prop: "flight", dir: "asc" }],
    layout: { cover: "photo", size: "small", total: { prop: "hours", fn: "sum" } },
  }),
  v("v_m_gallery", "Gallery", "gallery", {
    props: show(MISSION_SCHEMA, ["program", "flight", "crew"]),
    sorts: [{ prop: "flight", dir: "asc" }],
    layout: { cover: "photo", size: "medium", fit: "cover" },
  }),
  v("v_m_list", "Crewed by program", "list", {
    props: show(MISSION_SCHEMA, ["flight", "crew", "highlight"]),
    group: { prop: "program" },
    sorts: [{ prop: "flight", dir: "asc" }],
    layout: { cover: "photo", size: "small" },
  }),
  v("v_m_cal", "Calendar", "calendar", {
    props: show(MISSION_SCHEMA, ["program"]),
    layout: { date: "flight", mode: "month", weekends: true },
  }),
  v("v_m_time", "Timeline", "timeline", {
    props: show(MISSION_SCHEMA, ["program", "traj", "hours"]),
    sorts: [{ prop: "flight", dir: "desc" }],
    layout: { date: "flight", zoom: "month", table: true },
  }),
  v("v_m_chart", "Hours by program", "chart", {
    layout: { chart: { kind: "bar", x: { prop: "program" }, series: { prop: "traj" }, fn: "sum", prop: "hours", stacked: true, labels: true, legend: true } },
  }),
  v("v_m_heat", "Launch heatmap", "heatmap", {
    layout: { heatmap: { prop: "flight", domain: "year", count: 1, color: "blue" } },
  }),
];

const astronautViews = (): SeedView[] => [
  v("v_a_table", "Crew roster", "table", {
    props: show(ASTRONAUT_SCHEMA, ["title", "flights", "count", "total", "first", "last", "span", "veteran"]),
    sorts: [{ prop: "total", dir: "desc" }],
    calcs: { total: "sum", count: "sum" },
  }),
  v("v_a_board", "By experience", "board", {
    props: show(ASTRONAUT_SCHEMA, ["count", "total"]),
    group: { prop: "veteran" },
    sorts: [{ prop: "total", dir: "desc" }],
    layout: { cover: "none", size: "small", total: { prop: "total", fn: "sum" } },
  }),
  v("v_a_chart", "Hours flown", "chart", {
    sorts: [{ prop: "total", dir: "desc" }],
    layout: { chart: { kind: "hbar", x: { prop: "title" }, fn: "sum", prop: "total", sort: "value_desc", limit: 12, labels: true } },
  }),
];

const programViews = (): SeedView[] => [
  v("v_p_table", "Programs", "table", {
    props: show(PROGRAM_SCHEMA, ["title", "missions", "count", "hours", "longest", "seats", "first", "last", "avg"]),
    calcs: { count: "sum", hours: "sum", seats: "sum" },
  }),
  v("v_p_chart", "Flight hours", "chart", {
    layout: { chart: { kind: "donut", x: { prop: "title" }, fn: "sum", prop: "hours", labels: true, legend: true } },
  }),
];

export function seedCollections(): SeedCollection[] {
  return [
    { id: COL_MISSIONS, title: "Crewed missions", icon: "i:rocket", schema: MISSION_SCHEMA, views: missionViews(), rows: missionRows() },
    { id: COL_ASTRONAUTS, title: "Astronauts", icon: "i:users", schema: ASTRONAUT_SCHEMA, views: astronautViews(), rows: astronautRows() },
    { id: COL_PROGRAMS, title: "Programs", icon: "i:layers", schema: PROGRAM_SCHEMA, views: programViews(), rows: programRows() },
  ];
}
