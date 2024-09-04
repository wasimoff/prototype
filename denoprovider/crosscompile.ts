#!/usr/bin/env -S deno run --allow-run

// possible compilation targets
const targets = [
  "x86_64-unknown-linux-gnu",
  "aarch64-unknown-linux-gnu",
  "x86_64-pc-windows-msvc",
  "x86_64-apple-darwin",
  "aarch64-apple-darwin",
] as const;
type Target = typeof targets[number];

// func to compile for a single target
async function compile(target: Target = Deno.build.target as Target) {
  const cmd = new Deno.Command("deno", {
    args: [
      "compile", "--allow-env", "--allow-net",
      "--allow-read=./,../webprovider/",
      "--target", target,
      "--output", `wasimoff-provider-${target}${target.includes("windows") ? ".exe" : ""}`,
      "main.ts"
    ],
  });
  // execute it, inheriting stdio
  return await cmd.spawn().status;
};

// compile all known targets
for (const target of targets) {
  console.log(`--> compile ${target}`);
  await compile(target);
};
