import type { View } from "../mygo";

export function shouldAutoOpenModelWizard(settings: View, dismissed: boolean) {
  return settings.models.length === 0 && !dismissed;
}
