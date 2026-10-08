type Views = typeof import("./views-entry");

export type DatabaseViews = Views;

let job: Promise<Views> | null = null;
let loaded: Views | null = null;

export function loadDatabase(): Promise<Views> {
  if (!job) {
    job = import("./views-entry").then(
      (m) => (loaded = m),
      (e: unknown) => {
        job = null;
        throw e;
      },
    );
  }
  return job;
}

export function databaseViews(): Views | null {
  return loaded;
}
