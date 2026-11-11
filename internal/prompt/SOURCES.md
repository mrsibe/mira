# Prompt sources

Mira's chat prompt is assembled from public descriptions of ChatGPT's chat
instructions. The source snapshots are research material and are not shipped
verbatim because most of them describe tools and product surfaces Mira does not
have.

Retrieved on 2026-11-11 from
`https://github.com/asgeirtj/system_prompts_leaks/tree/main/OpenAI`:

| Source | SHA-256 | Used for |
| --- | --- | --- |
| `gpt-5.5-instant.md` | `217d1fbb453ec9740994286a3ed76ed462f35c0cd5628b9a03e157d8642881b9` | Core request priority, context use, source-of-truth rules, factuality, verbal-tic guidance |
| `gpt-5.5-thinking.md` | `060e5ee038f65f08ef3bf9eae5dfbe5fbe903ebe6ec564d251829b230c306016` | Writing style and `oververbosity=4` |
| `chatgpt-personality-instructions.md` | `65d3fb8ed7b0e3003de2b5523f629bd9b45683608e6a1be5e259a5e36a7749b2` | The six selectable personalities and their shared additional instruction |
| `tool-advanced-memory.md` | `25a970aa6fca76be97fe2a3d010e1934f8a7bddec7c4c035b20d63ba45195d71` | Reference only for the four automatic-memory sections; no tool instructions copied |
| `gpt-5.6-sol.md` | `5f4ca63d00730919bf0046a627a6eef28bf9fc755b1d8c87ccc058aaac6ace89` | Reviewed but excluded from the default chat prompt |

## Deliberate changes

- Replace the ChatGPT/OpenAI identity with Mira and use a runtime-generated
  current date.
- Refer only to application context and tools that Mira actually exposes.
- Remove Writing Blocks, ads, content references, private tool namespaces,
  product UI instructions, channel markers, and unrelated user-profile data.
- Keep source wording intact wherever it describes general assistant behavior.
  Mira-specific tool policy, Intelligent UI instructions, expression sliders,
  and custom instructions are appended as separate modules.

These leaked/community-extracted prompts are not official production prompts.
The source, date, hashes, and exclusions are recorded here so the provenance is
auditable without committing the raw snapshots.
