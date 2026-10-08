import type { ViewType } from "../../../engine/types";
import type { ViewFactory } from "../surface";
import { boardView } from "./board";
import { calendarView } from "./calendar";
import { formView } from "./form";
import { galleryView, listView } from "./list";
import { tableView } from "./table";
import { timelineView } from "./timeline";

export const renderers: Partial<Record<ViewType, ViewFactory>> = {
  table: tableView,
  board: boardView,
  list: listView,
  gallery: galleryView,
  calendar: calendarView,
  timeline: timelineView,
  form: formView,
};
