#!/usr/bin/env node

import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";

const packageJson = JSON.parse(
  readFileSync(new URL("../package.json", import.meta.url), "utf8").replace(/^\uFEFF/, ""),
);
const version = process.env.npm_package_version || packageJson.version;
const env = {
  ...process.env,
  CGO_ENABLED: "0",
  GOARCH: "amd64",
  GOOS: "windows",
};

execFileSync(
  "go",
  [
    "build",
    "-trimpath",
    "-ldflags",
    `-s -w -X main.version=${version}`,
    "-o",
    "rainhush.exe",
    ".",
  ],
  {
    env,
    stdio: "inherit",
  },
);
