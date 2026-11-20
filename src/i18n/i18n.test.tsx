import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { I18nProvider, useMessages } from "./index";

function Probe() {
  const t = useMessages();
  return <p>{t.commonCancel()}</p>;
}

describe("I18nProvider", () => {
  it("renders English messages", () => {
    render(
      <I18nProvider initialLocale="en">
        <Probe />
      </I18nProvider>,
    );
    expect(screen.getByText("Cancel")).toBeTruthy();
  });

  it("renders Simplified Chinese messages", () => {
    render(
      <I18nProvider initialLocale="zh-Hans">
        <Probe />
      </I18nProvider>,
    );
    expect(screen.getByText("取消")).toBeTruthy();
  });
});
