import { Field, SegmentedControl } from "../../ui";
import type { SetDraftField, SettingsDraft } from "./types";

type PreferenceKey = "responseStyle" | "warmth" | "enthusiasm" | "emoji";

const preferences: Array<{
  key: PreferenceKey;
  title: string;
  options: ReadonlyArray<{ value: string; label: string }>;
}> = [
  {
    key: "responseStyle",
    title: "回答风格",
    options: [
      { value: "concise", label: "简洁" },
      { value: "balanced", label: "平衡" },
      { value: "detailed", label: "详细" },
    ],
  },
  {
    key: "warmth",
    title: "亲切程度",
    options: [
      { value: "reserved", label: "克制" },
      { value: "balanced", label: "自然" },
      { value: "friendly", label: "亲切" },
    ],
  },
  {
    key: "enthusiasm",
    title: "热情程度",
    options: [
      { value: "calm", label: "冷静" },
      { value: "balanced", label: "适中" },
      { value: "enthusiastic", label: "热情" },
    ],
  },
  {
    key: "emoji",
    title: "Emoji",
    options: [
      { value: "none", label: "不用" },
      { value: "sparing", label: "少量" },
      { value: "frequent", label: "适度" },
    ],
  },
];

type PersonalizationSettingsProps = {
  draft: SettingsDraft;
  setField: SetDraftField;
};

export function PersonalizationSettings({ draft, setField }: PersonalizationSettingsProps) {
  return (
    <>
      <section className="settings-group">
        <Field label="称呼">
          {(id) => (
            <input
              id={id}
              value={draft.userName}
              maxLength={32}
              onChange={(event) => setField("userName", event.target.value)}
              placeholder="怎么称呼你"
            />
          )}
        </Field>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">表达方式</h3>
        <div className="preference-list">
          {preferences.map((preference) => (
            <div key={preference.key} className="preference-row">
              <div>
                <strong>{preference.title}</strong>
              </div>
              <SegmentedControl
                label={preference.title}
                value={draft[preference.key]}
                options={preference.options}
                className="choice-grid"
                optionClassName="choice-option"
                onChange={(value) => setField(preference.key, value)}
              />
            </div>
          ))}
        </div>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">自定义指令</h3>
        <p className="settings-description">应用到所有对话。</p>
        <textarea
          className="settings-textarea"
          value={draft.customInstructions}
          maxLength={5000}
          rows={7}
          onChange={(event) => setField("customInstructions", event.target.value)}
          placeholder="例如：解释技术概念时，先给一个类比，再给具体细节。"
        />
        <span className="field-meta">{draft.customInstructions.length} / 5000</span>
      </section>
    </>
  );
}
