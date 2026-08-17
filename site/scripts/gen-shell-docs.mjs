#!/usr/bin/env node
// Generates:
//   site/src/content/docs/docs/shell-builtin-reference.md
//   site/src/content/docs/docs/shell-functions-reference.md
//
// The shell builtin registry and template FuncMap registry are Go data
// (internal/shell/builtin, internal/shell/tmpl) and aren't introspectable
// from Node, so this script shells out to `go run ./tools/docgen` (repo
// root) which prints the same registries as JSON, and renders that JSON
// into Starlight-frontmattered Markdown. Like gen-cli-docs.mjs, this is
// the M11.0 anti-drift path: nothing here is hand-copied from SPEC.md.

import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const siteRoot = join(__dirname, "..");
const repoRoot = join(siteRoot, "..");
const docsDir = join(siteRoot, "src", "content", "docs", "docs");

// Prose context for each builtin: what it maps to (REST endpoint or local
// action), taken from the inventory (internal/shell/builtin/*.go via
// registry.go's All()). Kept short and deliberately separate from the
// generated flag data above it, so a renamed/added flag never requires
// touching this file — only new/renamed *builtins* do.
const builtinContext = {
  list: "GET /api/v1/messages — list/filter stored messages.",
  show: "GET /api/v1/messages/:id — show one message's detail/body/headers.",
  delete: "DELETE /api/v1/messages/:id, or clear's bulk path with --all.",
  clear: "DELETE /api/v1/messages?confirm=true — delete every stored message.",
  export: "GET /api/v1/messages/:id/export, or a bulk zip export with --all.",
  attachment: "GET /api/v1/messages/:id/attachments/:attId — download an attachment.",
  send: "Sends directly over SMTP via the shell's SMTP client (cliclient.SendTLS) — not a REST call.",
  randmsg: "Generates and sends a random message over SMTP (same SMTP override flags as send).",
  intmsg: "Generates and sends an \"interesting\" message over SMTP (same SMTP override flags as send).",
  weirdmsg: "Generates and sends a deliberately malformed message over SMTP (same SMTP override flags as send).",
  blast: "Sends a burst of generated messages over SMTP (same SMTP override flags as send).",
  deluge: "Sends a sustained stream of generated messages over SMTP (same SMTP override flags as send).",
  echo: "Local-only — prints text back to the shell.",
  example: "Local-only — emits a canned example message (eml or json).",
  prompt: "Local-only — inspects or resets the shell prompt state.",
  functions: "Local-only — lists the template FuncMap registry (see the functions reference below).",
  stats: "GET /api/v1/stats — server stats (message count, storage size, uptime).",
  health: "GET /api/v1/health — health check.",
  version: "GET /api/v1/version, or local build info with --local.",
  config: "Local-only — get/set/list/save shell session config.",
  set: "Local-only — set a shell variable.",
  unset: "Local-only — unset a shell variable.",
  vars: "Local-only — list shell variables.",
  alias: "Local-only — define a command alias.",
  unalias: "Local-only — remove a command alias.",
  abbr: "Local-only — define an abbreviation, expanded on a trigger key.",
  unabbr: "Local-only — remove an abbreviation.",
  template: "Local-only — inspect template settings/registry (--funcs).",
  history: "Local-only — inspect or clear shell command history.",
  edit: "Local-only — open the last command (or a script) in $VISUAL/$EDITOR.",
  sh: "Local-only — run a raw shell command.",
  source: "Local-only — execute a script file in the current session.",
  help: "Local-only — show help for a builtin.",
  exit: "Local-only — exit the shell.",
};

function runDocgen() {
  const stdout = execFileSync("go", ["run", "./tools/docgen"], {
    cwd: repoRoot,
    encoding: "utf8",
    maxBuffer: 16 * 1024 * 1024,
  });
  return JSON.parse(stdout);
}

function fmtFlag(f) {
  const short = f.shorthand ? `-${f.shorthand}, ` : "";
  const def = f.defValue && f.defValue !== "" && f.defValue !== "[]" ? ` (default \`${f.defValue}\`)` : "";
  return `- \`${short}--${f.name}\`${def} — ${f.usage}`;
}

function renderBuiltins(data) {
  let out = `---\ntitle: "Shell Builtin Reference"\ndescription: "Every builtin command in the maelsink interactive shell, generated from the builtin registry."\n---\n\n`;
  out += "This page is generated from `internal/shell/builtin`'s registry (`go run ./tools/docgen`) — the flag list for each builtin is always in sync with the shell binary; it is never hand-maintained.\n\n";

  for (const b of data.builtins) {
    out += `## ${b.name}\n\n`;
    if (b.aliases.length > 0) {
      out += `Aliases: ${b.aliases.map((a) => `\`${a}\``).join(", ")}\n\n`;
    }
    const ctx = builtinContext[b.name];
    if (ctx) out += `${ctx}\n\n`;
    if (b.flags.length > 0) {
      out += b.flags.map(fmtFlag).join("\n") + "\n\n";
    } else {
      out += "_No flags._\n\n";
    }
  }
  return out;
}

function renderFunctions(data) {
  let out = `---\ntitle: "Shell Functions Reference"\ndescription: "Every template function available to maelsink shell templates, generated from the template function registry."\n---\n\n`;
  out += "This page is generated from `internal/shell/tmpl`'s registry (`go run ./tools/docgen`) — descriptions and signatures are always in sync with the engine; they are never hand-maintained.\n\n";
  out += ":::caution\n";
  out += "Functions marked **unsafe** below (`env`, `expandenv`, `getHostByName`) are gated behind the shell's `--template-unsafe-funcs` / `-Z` flag (or the `shell.template_unsafe_funcs` config key). They are disabled by default because they can read host environment variables or perform DNS lookups from inside a template.\n";
  out += ":::\n\n";

  const byCategory = new Map();
  for (const f of data.functions) {
    if (!byCategory.has(f.category)) byCategory.set(f.category, []);
    byCategory.get(f.category).push(f);
  }

  const categoryOrder = ["identifiers", "generate", "string", "date", "encoding", "email", "files", "ansi"];
  const orderedCategories = [
    ...categoryOrder.filter((c) => byCategory.has(c)),
    ...[...byCategory.keys()].filter((c) => !categoryOrder.includes(c)),
  ];

  for (const cat of orderedCategories) {
    out += `## ${cat}\n\n`;
    for (const f of byCategory.get(cat)) {
      const badge = f.unsafe ? " **(unsafe)**" : "";
      out += `### \`${f.name}\`${badge}\n\n`;
      if (f.args || f.returns) {
        out += `\`${f.name}(${f.args || ""})\`${f.returns ? ` -> ${f.returns}` : ""}\n\n`;
      }
      if (f.description) out += `${f.description}\n\n`;
      if (f.unsafe) {
        out += "Requires `--template-unsafe-funcs` / `-Z`.\n\n";
      }
    }
  }
  return out;
}

function main() {
  const data = runDocgen();
  mkdirSync(docsDir, { recursive: true });

  const builtinsPath = join(docsDir, "shell-builtin-reference.md");
  writeFileSync(builtinsPath, renderBuiltins(data), "utf8");
  console.log(`[gen-shell-docs] wrote ${builtinsPath}`);

  const functionsPath = join(docsDir, "shell-functions-reference.md");
  writeFileSync(functionsPath, renderFunctions(data), "utf8");
  console.log(`[gen-shell-docs] wrote ${functionsPath}`);
}

main();
