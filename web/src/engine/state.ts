export type Unsubscribe = () => void;

type Watcher<V> = (value: V, previous: V) => void;

export class State<T extends object> {
  private values: T;
  private watchers = new Map<keyof T, Set<Watcher<never>>>();

  constructor(initial: T) {
    this.values = { ...initial };
  }

  get<K extends keyof T>(key: K): T[K] {
    return this.values[key];
  }

  set<K extends keyof T>(key: K, value: T[K]): void {
    const previous = this.values[key];
    if (Object.is(previous, value)) return;
    this.values[key] = value;
    const set = this.watchers.get(key);
    if (!set) return;
    for (const fn of [...set]) (fn as Watcher<T[K]>)(value, previous);
  }

  watch<K extends keyof T>(key: K, fn: Watcher<T[K]>, options: { immediate?: boolean } = {}): Unsubscribe {
    let set = this.watchers.get(key);
    if (!set) {
      set = new Set();
      this.watchers.set(key, set);
    }
    set.add(fn as Watcher<never>);
    if (options.immediate) fn(this.values[key], this.values[key]);
    return () => {
      set.delete(fn as Watcher<never>);
    };
  }
}
