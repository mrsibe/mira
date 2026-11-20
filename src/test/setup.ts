import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

Object.defineProperty(window.navigator, "language", {
  configurable: true,
  value: "zh-Hans",
});
Object.defineProperty(window.navigator, "languages", {
  configurable: true,
  value: ["zh-Hans"],
});

afterEach(cleanup);
