import { AlertCircle, X } from "lucide-react";
import { useId } from "react";
import { Button, IconButton, Modal } from "../ui";

type ConfirmDialogProps = {
  title: string;
  message: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onClose: () => void;
};

export function ConfirmDialog({
  title,
  message,
  confirmLabel = "删除",
  onConfirm,
  onClose,
}: ConfirmDialogProps) {
  const titleId = useId();
  const descriptionId = useId();

  return (
    <Modal
      className="confirm-modal"
      role="alertdialog"
      labelledBy={titleId}
      describedBy={descriptionId}
      onClose={onClose}
    >
      <header className="modal-head">
        <div className="confirm-title">
          <span className="confirm-icon">
            <AlertCircle size={18} />
          </span>
          <div>
            <h2 id={titleId}>{title}</h2>
            <p id={descriptionId}>{message}</p>
          </div>
        </div>
        <IconButton label="关闭" onClick={onClose}>
          <X size={18} />
        </IconButton>
      </header>
      <footer className="modal-actions">
        <span className="spacer" />
        <Button onClick={onClose}>取消</Button>
        <Button variant="danger" onClick={onConfirm}>
          {confirmLabel}
        </Button>
      </footer>
    </Modal>
  );
}
