#!/usr/bin/env node
// ドキュメントと GitHub Actions(yml)の、簡単な検査。依存なし(Node 標準のみ)。make verify から呼ばれる。
// ルールは、実際に起きた失敗に絞る。複雑にしない(正規表現で見られる範囲だけ)。
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = process.env.VERIFY_ROOT ?? resolve(dirname(fileURLToPath(import.meta.url)), "..");
const read = (p) => readFileSync(join(ROOT, p), "utf8");
const rel = (p) => relative(ROOT, p);

let failures = 0;
function check(name, problems) {
  if (problems.length === 0) return console.log(`  ok   ${name}`);
  failures++;
  console.log(`  NG   ${name}`);
  for (const p of problems) console.log(`         ${p}`);
}

function walk(dir, ext) {
  if (!existsSync(dir)) return [];
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    if (name === ".terraform" || name === "node_modules") return [];
    return statSync(p).isDirectory() ? walk(p, ext) : p.endsWith(ext) ? [p] : [];
  });
}

// ---------------------------------------------------------------- ドキュメント
console.log("\n== ドキュメント");

const OPENQ = "docs/engineering/00-openquestion.md";
const mdFiles = [join(ROOT, "README.md"), join(ROOT, "CLAUDE.md"), ...walk(join(ROOT, "docs"), ".md"), ...walk(join(ROOT, ".claude"), ".md")].filter(existsSync);
// この検証のスキルは、検出する失敗の例として、削除済みの番号や廃止した用語を載せている
const isExample = (f) => rel(f).includes("verify-professional");
// コードブロックとインラインコードは、例として書かれたものなので、検査から外す
const stripCode = (t) => t.replace(/```[\s\S]*?```/g, "").replace(/`[^`\n]*`/g, "");

check(
  "Markdown の相対リンクが、存在するファイルを指す",
  mdFiles.flatMap((f) =>
    [...stripCode(readFileSync(f, "utf8")).matchAll(/\[[^\]]*\]\(([^)\s]+)\)/g)]
      .map((m) => m[1])
      .filter((t) => !/^(https?:|mailto:|#)/.test(t))
      .filter((t) => !existsSync(resolve(dirname(f), t.split("#")[0])))
      .map((t) => `${rel(f)}: リンク切れ → ${t}`),
  ),
);

if (existsSync(join(ROOT, OPENQ))) {
  const defined = new Set([...read(OPENQ).matchAll(/^\| ([QD]\d+) \|/gm)].map((m) => m[1]));
  check(
    "参照している Q / D の番号が、00-openquestion.md にある",
    mdFiles
      .filter((f) => rel(f) !== OPENQ && !isExample(f))
      .flatMap((f) =>
        [...new Set(readFileSync(f, "utf8").match(/\b[QD]\d+\b/g) ?? [])]
          .filter((id) => !defined.has(id))
          .map((id) => `${rel(f)}: ${id} が 00-openquestion.md にない(解決して削除した未決か、番号の誤り)`),
      ),
  );
}

// 廃止した用語。決定の経緯を書く 00-openquestion.md は対象外
const RETIRED = [/\bTASK\b/, /LINE_CHANNEL_ACCESS_TOKEN/, /line-channel-access-token/, /\bfunctions\//, /bacnend/];
check(
  "廃止した用語が残っていない",
  mdFiles
    .filter((f) => rel(f) !== OPENQ && !isExample(f))
    .flatMap((f) => RETIRED.filter((re) => re.test(readFileSync(f, "utf8"))).map((re) => `${rel(f)}: ${re.source}`)),
);

const docsReadme = existsSync(join(ROOT, "docs/README.md")) ? read("docs/README.md").split("\n").length : 0;
check(`docs/README.md が 100 行以内(${docsReadme} 行)`, docsReadme > 100 ? ["100 行を超えている"] : []);

// ---------------------------------------------------------------- GitHub Actions(yml)
console.log("\n== GitHub Actions(yml。ローカルでは動かせないため、簡単な静的検査のみ)");

const wfFiles = walk(join(ROOT, ".github/workflows"), ".yml");
const workflows = Object.fromEntries(wfFiles.map((f) => [f.split("/").pop(), readFileSync(f, "utf8")]));

// ファイル直下のキー(permissions: や defaults:)の中身。次のインデントなしの行までを、その範囲とする
const topBlock = (text, key) => text.match(new RegExp(`^${key}:[^\\n]*\\n((?:[ \\t]+.*\\n?|\\n)*)`, "m"))?.[1] ?? "";

// jobs: の下を、ジョブごとのブロックに分ける({ id: ブロックのテキスト })
function splitJobs(text) {
  const lines = text.split("\n");
  const start = lines.findIndex((l) => /^jobs:\s*$/.test(l));
  if (start < 0) return {};
  const jobs = {};
  let current = null;
  for (const line of lines.slice(start + 1)) {
    const m = line.match(/^  ([A-Za-z0-9_-]+):\s*$/);
    if (m) {
      current = m[1];
      jobs[current] = "";
    } else if (current) jobs[current] += line + "\n";
  }
  return jobs;
}
const allJobs = Object.entries(workflows).flatMap(([file, text]) => Object.entries(splitJobs(text)).map(([id, block]) => ({ file, id, block, text })));

check(
  "タブ文字がない(YAML では使えない)",
  Object.entries(workflows).filter(([, t]) => t.includes("\t")).map(([f]) => f),
);

check(
  "呼び出す reusable workflow が存在する",
  allJobs.flatMap(({ file, id, block }) => {
    const m = block.match(/^\s+uses:\s*\.\/\.github\/workflows\/(\S+)/m);
    return m && !workflows[m[1]] ? [`${file}: job ${id} → ${m[1]} が存在しない`] : [];
  }),
);

// 呼び出し先のジョブが id-token: write を要求するのに、呼び出し側が許可していないと、実行前にエラーになる
check(
  "呼び出し先が id-token: write を要求するとき、呼び出し側も許可している",
  allJobs.flatMap(({ file, id, block, text }) => {
    const m = block.match(/^\s+uses:\s*\.\/\.github\/workflows\/(\S+)/m);
    if (!m || !workflows[m[1]]?.includes("id-token: write")) return [];
    const allowed = /id-token:\s*write/.test(block) || /id-token:\s*write/.test(topBlock(text, "permissions"));
    return allowed ? [] : [`${file}: job ${id} は ${m[1]} を呼ぶが、permissions に id-token: write がない(実行しないジョブでも、エラーになる)`];
  }),
);

check(
  "needs が、存在するジョブを指す",
  allJobs.flatMap(({ file, id, block, text }) => {
    const ids = Object.keys(splitJobs(text));
    const m = block.match(/^\s+needs:\s*(.+)$/m);
    if (!m) return [];
    return m[1]
      .replace(/[\[\]]/g, "")
      .split(",")
      .map((s) => s.trim())
      .filter((n) => n && !ids.includes(n))
      .map((n) => `${file}: job ${id} の needs ${n} が存在しない`);
  }),
);

// working-directory を指定したジョブで、checkout より前に run すると、ディレクトリが存在せず失敗する
check(
  "working-directory のあるジョブで、checkout より前に run していない",
  allJobs.flatMap(({ file, id, block, text }) => {
    if (!/working-directory:/.test(block) && !/working-directory:/.test(topBlock(text, "defaults"))) return [];
    const steps = block.slice(block.indexOf("steps:"));
    const run = steps.search(/^\s*-?\s*run:/m);
    const checkout = steps.indexOf("actions/checkout");
    return run >= 0 && (checkout < 0 || run < checkout) ? [`${file}: job ${id} は、checkout より前に run がある`] : [];
  }),
);

// Environment の変数名は、02-infra.md の表を正本にする
const infraDoc = existsSync(join(ROOT, "docs/engineering/02-infra.md")) ? read("docs/engineering/02-infra.md") : "";
const documentedVars = new Set([...infraDoc.matchAll(/^\| `([A-Z][A-Z0-9_]*)` \|/gm)].map((m) => m[1]));
check(
  "ワークフローが使う vars.* が、02-infra.md の表にある",
  Object.entries(workflows).flatMap(([file, text]) =>
    [...new Set([...text.matchAll(/\bvars\.([A-Z][A-Z0-9_]*)/g)].map((m) => m[1]))]
      .filter((v) => !documentedVars.has(v))
      .map((v) => `${file}: vars.${v} が 02-infra.md の表にない`),
  ),
);

// terraform apply するワークフローは、必須の変数(default なし)を TF_VAR_* で渡す
const tfEnvs = walk(join(ROOT, "infra/terraform/env"), "variables.tf");
const tfVars = (f) =>
  [...readFileSync(f, "utf8").matchAll(/variable "(\w+)" \{([\s\S]*?)\n\}/g)].map((m) => ({ name: m[1], required: !/^\s*default\s*=/m.test(m[2]) }));
check(
  "terraform apply するワークフローが、必須の変数を TF_VAR_* で渡す",
  Object.entries(workflows)
    .filter(([, t]) => t.includes("terraform apply"))
    .flatMap(([file, text]) => {
      const passed = new Set([...text.matchAll(/TF_VAR_(\w+):/g)].map((m) => m[1]));
      return tfEnvs.flatMap((f) => {
        const vars = tfVars(f);
        return [
          ...vars.filter((v) => v.required && !passed.has(v.name)).map((v) => `${file}: ${rel(f)} の必須の変数 ${v.name} を TF_VAR_${v.name} で渡していない`),
          ...[...passed].filter((p) => !vars.some((v) => v.name === p)).map((p) => `${file}: TF_VAR_${p} は ${rel(f)} にない変数`),
        ];
      });
    }),
);

check(
  "terraform apply を、cancel-in-progress: true で中断しない",
  Object.entries(workflows)
    .filter(([, t]) => t.includes("terraform apply") && /cancel-in-progress:\s*true/.test(t))
    .map(([f]) => `${f}: apply が途中で中断されうる`),
);

console.log(failures ? `\n失敗 ${failures} 件` : "\nすべて通った");
process.exit(failures ? 1 : 0);
