import type { Conversation } from "../mygo";

export const conversationDragMime = "application/x-mira-conversation";

export type ConversationDrag = {
  conversationID: string;
  sourceProjectID: string;
};

export function conversationProjectID(conversation: Pick<Conversation, "projectId">): string {
  return conversation.projectId ?? "";
}

export function canDropConversation(
  drag: ConversationDrag | null,
  targetProjectID: string,
): drag is ConversationDrag {
  return drag !== null && drag.sourceProjectID !== targetProjectID;
}
