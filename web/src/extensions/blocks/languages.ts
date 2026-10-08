export const LANGUAGES: readonly (readonly [string, string])[] = [
  ["bash", "Bash"],
  ["c", "C"],
  ["cpp", "C++"],
  ["csharp", "C#"],
  ["css", "CSS"],
  ["diff", "Diff"],
  ["dockerfile", "Dockerfile"],
  ["go", "Go"],
  ["graphql", "GraphQL"],
  ["html", "HTML"],
  ["ini", "INI and TOML"],
  ["java", "Java"],
  ["javascript", "JavaScript"],
  ["json", "JSON"],
  ["kotlin", "Kotlin"],
  ["lua", "Lua"],
  ["makefile", "Makefile"],
  ["markdown", "Markdown"],
  ["mermaid", "Mermaid"],
  ["php", "PHP"],
  ["python", "Python"],
  ["ruby", "Ruby"],
  ["rust", "Rust"],
  ["scss", "SCSS"],
  ["shell", "Shell session"],
  ["sql", "SQL"],
  ["swift", "Swift"],
  ["typescript", "TypeScript"],
  ["xml", "XML"],
  ["yaml", "YAML"],
];

const ALIASES: Record<string, string> = {
  sh: "bash",
  zsh: "bash",
  js: "javascript",
  jsx: "javascript",
  ts: "typescript",
  tsx: "typescript",
  py: "python",
  rb: "ruby",
  rs: "rust",
  yml: "yaml",
  toml: "ini",
  md: "markdown",
  "c++": "cpp",
  cs: "csharp",
  golang: "go",
  docker: "dockerfile",
  console: "shell",
  svg: "xml",
};

export function normaliseLanguage(raw: string): string {
  const l = raw.trim().toLowerCase();
  return ALIASES[l] ?? l;
}

export function languageLabel(id: string): string {
  const n = normaliseLanguage(id);
  return LANGUAGES.find(([k]) => k === n)?.[1] ?? id;
}
