import { useMessages } from "../../i18n";
import { Field, Input, SegmentedControl, Switch } from "../../ui";
import type { SetDraftField, SettingsDraft } from "./types";

type PreferenceKey = "responseStyle" | "warmth" | "enthusiasm" | "emoji";

type PersonalizationSettingsProps = {
  draft: SettingsDraft;
  setField: SetDraftField;
};

export function PersonalizationSettings({ draft, setField }: PersonalizationSettingsProps) {
  const t = useMessages();
  const personalities = [
    { value: "professional", label: t.personalityProfessional() },
    { value: "friendly", label: t.personalityFriendly() },
    { value: "candid", label: t.personalityCandid() },
    { value: "quirky", label: t.personalityQuirky() },
    { value: "efficient", label: t.personalityEfficient() },
    { value: "cynical", label: t.personalityCynical() },
  ] as const;
  const preferences: Array<{
    key: PreferenceKey;
    title: string;
    options: ReadonlyArray<{ value: string; label: string }>;
  }> = [
    {
      key: "responseStyle",
      title: t.responseStyleTitle(),
      options: [
        { value: "concise", label: t.responseStyleConcise() },
        { value: "balanced", label: t.responseStyleBalanced() },
        { value: "detailed", label: t.responseStyleDetailed() },
      ],
    },
    {
      key: "warmth",
      title: t.warmthTitle(),
      options: [
        { value: "reserved", label: t.warmthReserved() },
        { value: "balanced", label: t.warmthBalanced() },
        { value: "friendly", label: t.warmthFriendly() },
      ],
    },
    {
      key: "enthusiasm",
      title: t.enthusiasmTitle(),
      options: [
        { value: "calm", label: t.enthusiasmCalm() },
        { value: "balanced", label: t.enthusiasmBalanced() },
        { value: "enthusiastic", label: t.enthusiasmEnthusiastic() },
      ],
    },
    {
      key: "emoji",
      title: t.emojiTitle(),
      options: [
        { value: "none", label: t.emojiNone() },
        { value: "sparing", label: t.emojiSparing() },
        { value: "frequent", label: t.emojiFrequent() },
      ],
    },
  ];
  return (
    <>
      <section className="settings-group">
        <Field label={t.personalizationName()}>
          {(id) => (
            <Input
              id={id}
              value={draft.userName}
              maxLength={32}
              onChange={(event) => setField("userName", event.target.value)}
              placeholder={t.personalizationNamePlaceholder()}
            />
          )}
        </Field>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">{t.personalizationPersonality()}</h3>
        <p className="settings-description">{t.personalizationPersonalityDescription()}</p>
        <SegmentedControl
          label={t.personalizationPersonality()}
          value={draft.personality}
          options={personalities}
          className="personality-grid"
          optionClassName="personality-option"
          onChange={(value) => setField("personality", value)}
        />
      </section>

      <section className="settings-group">
        <Switch
          checked={draft.intelligentUIEnabled}
          title={t.personalizationIntelligentUI()}
          description={t.personalizationIntelligentUIDescription()}
          onChange={() => setField("intelligentUIEnabled", !draft.intelligentUIEnabled)}
        />
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">{t.personalizationExpression()}</h3>
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
        <h3 className="settings-group-title">{t.personalizationCustomInstructions()}</h3>
        <p className="settings-description">{t.personalizationCustomInstructionsDescription()}</p>
        <textarea
          className="settings-textarea"
          value={draft.customInstructions}
          maxLength={5000}
          rows={7}
          onChange={(event) => setField("customInstructions", event.target.value)}
          placeholder={t.personalizationCustomInstructionsPlaceholder()}
        />
        <span className="field-meta">{draft.customInstructions.length} / 5000</span>
      </section>
    </>
  );
}
