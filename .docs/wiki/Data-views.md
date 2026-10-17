# Data views

Every database can be seen as a table, board, list, gallery, calendar, timeline, form, chart, dashboard, heatmap, graph, map or feed. This page covers the visual ones added in v0.12.0: the chart kinds, the heatmap, the graph, and the dashboard and board upgrades. All of them follow the theme (dark by default), are drawn without a chart library, load with the database views on demand, and keep a fixed size while they load so nothing on the page moves.

## Layout

| Path | What lives there |
|---|---|
| `internal/collection/chart.go` | The chart endpoint: buckets, measures, series, dot points and picks |
| `internal/collection/query.go` | Date groupings, including `weekday` and `hour` |
| `web/src/extensions/database/views/chart.ts` | The chart view, its settings panel and mark clicks |
| `web/src/extensions/database/views/chartdraw.ts` | Bar, line and donut drawing and the shared SVG helpers |
| `web/src/extensions/database/views/chartkinds.ts` | Area, radial, gauge, histogram, dot plot, comparison, matrix and calendar drawing |
| `web/src/extensions/database/views/heatmap.ts` | The heatmap view |
| `web/src/extensions/database/views/graph.ts` | The graph view |
| `web/src/extensions/database/views/graphsim.ts` | The force layout, run in a worker |
| `web/src/extensions/database/views/kpi.ts` | KPI cards on dashboards |
| `web/src/extensions/database/views/picklist.ts` | The list of rows behind a clicked mark |
| `web/src/styles/viz.css` | Styles for all of the above |

## Charts

A chart view (or a dashboard widget) picks a kind, a property for the x axis, an optional property to split into series, and a measure (count, sum, average, median, min, max, range, counts and percentages). The kinds:

| Kind | What it draws | Notes |
|---|---|---|
| Vertical and horizontal bar | Bars per group, grouped or stacked by series | |
| Line | A line per series, optionally smooth and cumulative | |
| Area | Filled areas, optionally stacked, smooth or cumulative | |
| Donut, radial bar | One ring per group | No series |
| Number, gauge | One value; the gauge fills toward a target and turns green once it is met | |
| Histogram | Bars over a number property in ranges of the chosen size, with a cumulative share line on a second axis | X must be a number |
| Dot plot | One dot per row placed by a number property, one band per group; with negative and positive values the dots are coloured by sign (a diverging beeswarm) | Up to 2,000 dots |
| Compare periods | The last N buckets of a date axis against the N before them, with the totals and the change | N is "Buckets per period", half the buckets by default |
| Heatmap | A grid of x groups by series groups, coloured by value; with a date by "Day of the week" against the same date by "Hour of the day" it is the hours-worked chart | Linear or quantile colour scale |
| Calendar heatmap | A year of days | Linear or quantile colour scale |

Dates group by day, week, month, year, relative ranges, day of the week or hour of the day.

**Clicking a mark** (a bar, slice, dot on a line, ring or heatmap cell) lists the rows behind it in a small panel; each opens for editing. Clicking a dot in a dot plot opens its row. The same works in dashboards.

## Heatmap view

A heatmap view shows one date property over time:

| Layout | Cells |
|---|---|
| Year, by day | A GitHub-style year of days (one to three years) |
| Months, by day | One block per month |
| Weeks, by hour | One block per week, 7 days by 24 hours |
| Days, by hour | One row per day, 24 hours across |

The value is a count of rows or the sum, average or max of a number property. The settings choose the date property, the layout and how many years, months, weeks or days, the value, a colour (accent, green, teal, amber, rose, violet), a linear or quantile scale and whether to print values in the cells. The arrows move back and forward a whole range; Today returns. Clicking a cell lists that day's (or hour's) rows and, for a date property you can edit, offers **Add a row here**, which creates a row on that date and hour and opens it. The heights are fixed per layout.

## Graph view

Rows are nodes and relation properties are links. Settings: colour by a select or status property, size by the number of links or by a number property, cluster by colour, labels (when there is room, always or never), which relation properties draw links, and whether rows linked from other databases appear (as outlined nodes, up to 600). Up to 2,000 rows are drawn.

- The layout is a Barnes-Hut force simulation that runs in a worker, so the page stays responsive; the view fits the graph while it settles until you zoom or pan.
- Scroll to zoom, drag empty space to pan, the buttons zoom and fit. Hovering a node highlights it and its neighbours. Find a row types a filter; Enter centres the first match.
- **Move** mode drags nodes. **Link** mode (or Shift in Move mode) drags from one node to another to relate the two rows through the relation property that points at the second row's database; with several such properties a menu asks which.
- Clicking a link offers Remove link. Double-clicking empty space adds a row there and puts the cursor in its title. Double-clicking a node opens its row.
- Clicking a node opens a side panel with its title, its colour property and its links (each removable); Open goes to the row page.
- For screen readers and keyboards, the graph carries a list of its rows after the canvas, each reading like "Alpha: linked to Beta, Gamma" (five names, then "and N more"). It is hidden until focus enters it: Tab from the canvas shows it as a panel in the corner, and Enter on a row selects it and centres the graph on it, as Find a row does.

## Dashboards

Widgets sit on a four-column grid and are small (one column), medium (two) or full width; on narrow screens they stack. Drag a widget by its grip to move it, or focus the grip and press the left and right (or up and down) arrow keys; focus stays on the grip and a screen reader hears the new position. Move earlier and Move later are also in each widget's menu. **Add a KPI** adds a number widget with a trend: pick a date property and day, week or month, and the card shows the value, a sparkline of the last 24 buckets and the change of the last bucket against the one before.

## Boards

The board's layout menu adds a **column total** (a count, or the sum or average of a number property) and a **progress bar** (the share of rows with a checkbox ticked, or the average of a number property against its maximum or 100). Each column's menu can collapse it to a narrow strip, which stays collapsed for everyone using the view, and set a card limit; a column over its limit shows "count / limit" in red. The pencil on a card edits its title in place; while editing, clicking a property shown on the card edits it in place too. The plus in a column header adds a card at the top, New at the bottom.

## Speed at scale

Measured on v0.19.0 with a 10,000-row database of a title, a status, a number, a text and a date property, on one development machine:

| What | Result |
|---|---|
| Query for the first 100 rows, sorted by a number, from the end (offset 9,900), and a title search | 7 to 14 ms each on the server |
| Opening the database in Chrome with the CPU slowed 4x and a 500 kbps, 400 ms network | first rows visible after 2.9 s |
| Row elements in the page while showing the table | 27 to 42 at any scroll position, out of 10,000 |
| Jumping to the middle and to the end of the table | painted in 25 and 34 ms; the last row appears once its window loads |
| Longest main-thread task seen with the CPU slowed 4x | 281 ms, during the first load |
| Adding rows through MCP `add_rows` | about 1.2 s per call of 100 rows at v0.19.0; 5000 rows in 18.6 s at v0.27.0 |
| Changing rows through MCP `update_rows` on a 5000-row database | 3000 rows in 53.8 s at v0.24.0, 19.1 s at v0.25.0, 8.6 s at v0.27.0 (the later figures measured with a second server running beside it) |

The table renders only the rows near the viewport and asks the server for each window as it scrolls, so the cost of a view does not grow with the row count; the server keeps each database's rows in memory after the first read, which is why sorting and searching stay in milliseconds.

## The API

`POST /api/collections/{id}/chart` takes `{"chart": {...}, "view_id": "...", "search": "...", "pick": {"x": "...", "series": "..."}}`. The chart object has `kind`, `x` and `series` groupings (`prop`, `by`, `step`), `fn`, `prop`, `target`, `window`, `scale`, `filter`, `limit` and the style flags. A `pick` answers with `picked`, the ids of up to 500 rows behind that mark, instead of values. A `dots` chart answers with `points` (`id`, `title`, `group`, `value`). Calendar charts return up to 1,100 days.

## The contract

1. A chart kind that the server aggregates differently goes into `chartKinds` (and, if it has one series, `singleSeries`) in `internal/collection/chart.go`; one that only draws differently needs only a drawing in `chartkinds.ts` and an entry in `EXTRA_KINDS`.
2. Every mark that stands for rows carries `data-x` (and `data-s` for a series) with the bucket keys the server returns, so a pick finds its rows; a mark for one row carries `data-row`.
3. A new view type is added to `viewTypes` in `internal/collection/service.go`, to the `collection_views.type` check by a migration that keeps `form_links` and `calendar_feeds`, to `ViewType` and `VIEW_INFO` on the client and to `renderers` in `views/index.ts`.

## Related

[[API-and-agents]] · [[Offline]]
