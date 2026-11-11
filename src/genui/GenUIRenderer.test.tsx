import type { Spec } from "@json-render/core";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { GenUIRenderer } from "./GenUIRenderer";

const spec: Spec = {
  root: "stack-1",
  elements: {
    "stack-1": {
      type: "Stack",
      props: { direction: "column", gap: "md", align: "stretch" },
      children: ["text-1"],
    },
    "text-1": {
      type: "Text",
      props: { content: "界面渲染正常", variant: "body" },
      children: [],
    },
  },
};

describe("GenUIRenderer", () => {
  it("renders a validated catalog spec", () => {
    render(<GenUIRenderer spec={spec} />);

    expect(screen.getByText("界面渲染正常")).toBeTruthy();
  });
});
