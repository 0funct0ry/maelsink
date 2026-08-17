#!/usr/bin/env node
// Generates site/src/content/docs/docs/cli-reference/*.md from the real
// `maelsink <cmd> --help` output of the built binary.
//
// This is the M11.0 anti-drift generator: the CLI reference must never be
// hand-copied from SPEC.md, because flags change independently of docs.
// The fenced --help block in each generated page IS the source of truth;
// the prose above it is just orientation.
//
// Usage: node scripts/gen-cli-docs.mjs   (run from site/, or via npm script)

import { execFileSync, execSync } from "node:child_process";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const siteRoot = join(__dirname, "..");
const repoRoot = join(siteRoot, "..");
const binPath = join(repoRoot, "bin", "maelsink");
const outDir = join(siteRoot, "src", "content", "docs", "docs", "cli-reference");

function ensureBinary() {
  if (existsSync(binPath)) return;
  console.log("[gen-cli-docs] bin/maelsink not found, running `make build-go`...");
  execSync("make build-go", { cwd: repoRoot, stdio: "inherit" });
  if (!existsSync(binPath)) {
    throw new Error(`build-go completed but ${binPath} still does not exist`);
  }
}

// One entry per leaf command page. `words` is the argv passed to the
// binary (after the binary name) to get that command's --help text.
// `slug` is the output filename (without .md). Pure parent groupings
// (bare `auth`, bare `config`) are intentionally omitted here — they add
// no content beyond their subcommands' own --help output, and Starlight's
// sidebar autogenerate lists whatever files exist in this directory.
const commands = [
  { slug: "serve", words: ["serve"], title: "serve", summary: "Start the SMTP, Web UI, and REST API servers. Running `maelsink` with no subcommand is equivalent to `maelsink serve`." },
  { slug: "shell", words: ["shell"], title: "shell", summary: "Start an interactive maelsink shell — a REPL client of the REST API and SMTP port." },
  { slug: "compose", words: ["compose"], title: "compose", summary: "Start the maelsink compose browser-based playground." },
  { slug: "send", words: ["send"], title: "send", summary: "Compose and send a test message to a maelsink instance over SMTP." },
  { slug: "list", words: ["list"], title: "list", summary: "List messages via the REST API." },
  { slug: "get", words: ["get"], title: "get", summary: "Show full message detail via the REST API." },
  { slug: "delete", words: ["delete"], title: "delete", summary: "Delete one message via the REST API." },
  { slug: "clear", words: ["clear"], title: "clear", summary: "Delete all messages via the REST API." },
  { slug: "export", words: ["export"], title: "export", summary: "Download a message as a .eml file via the REST API." },
  { slug: "auth-adduser", words: ["auth", "adduser"], title: "auth adduser", summary: "Add (or update) a Web UI Basic Auth user in an htpasswd-style credential file." },
  { slug: "auth-removeuser", words: ["auth", "removeuser"], title: "auth removeuser", summary: "Remove a Web UI Basic Auth user from an htpasswd-style credential file." },
  { slug: "config-init", words: ["config", "init"], title: "config init", summary: "Write a maelsink.yaml scaffolded with the built-in defaults." },
  { slug: "config-show", words: ["config", "show"], title: "config show", summary: "Print the fully-resolved configuration (defaults + file + env + flags) that `serve` would use." },
  { slug: "config-validate", words: ["config", "validate"], title: "config validate", summary: "Validate a maelsink.yaml (and the layered config it produces) without starting any server." },
  { slug: "version", words: ["version"], title: "version", summary: "Print version information." },
];

function helpOutput(words) {
  const args = [...words, "--help"];
  try {
    return execFileSync(binPath, args, { encoding: "utf8" });
  } catch (err) {
    // Cobra --help exits 0 normally; if it didn't, surface stdout+stderr
    // captured on the error object so failures are debuggable.
    const out = (err.stdout || "") + (err.stderr || "");
    if (out.trim()) return out;
    throw err;
  }
}

function frontmatter(title, description) {
  const esc = (s) => s.replace(/"/g, '\\"');
  return `---\ntitle: "${esc(title)}"\ndescription: "${esc(description)}"\n---\n\n`;
}

function main() {
  ensureBinary();
  mkdirSync(outDir, { recursive: true });

  for (const cmd of commands) {
    const help = helpOutput(cmd.words).trimEnd();
    const invocation = `maelsink ${cmd.words.join(" ")}`;
    const body =
      frontmatter(cmd.title, cmd.summary) +
      `${cmd.summary}\n\n` +
      `This page is generated directly from \`${invocation} --help\`. The flag list below is always in sync with the binary and is never hand-maintained.\n\n` +
      "```\n" +
      help +
      "\n```\n";

    const outPath = join(outDir, `${cmd.slug}.md`);
    writeFileSync(outPath, body, "utf8");
    console.log(`[gen-cli-docs] wrote ${outPath}`);
  }
}

main();
