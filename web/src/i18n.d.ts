declare module "virtual:i18n" {
  export const loaders: Record<string, () => Promise<{ default: Record<string, string> }>>;
}
