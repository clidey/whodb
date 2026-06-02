import { Dialog, DialogContent } from '@/components/ui/dialog'
import { ModalForm } from '@/components/ui/ModalForm'
import { Textarea } from '@/components/ui/Textarea'

export interface DocumentEditorDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  submitLabel: string
  description?: string
  placeholder?: string
  content: string
  onContentChange: (content: string) => void
  onSave: () => Promise<void>
}

/** Shared dialog shell for document add/edit modals. Owns layout, not behavior. */
export function DocumentEditorDialog({
  open,
  onOpenChange,
  title,
  submitLabel,
  description,
  placeholder,
  content,
  onContentChange,
  onSave,
}: DocumentEditorDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
        <ModalForm.Provider onSubmit={onSave} meta={{ title, description }}>
          <ModalForm.Header />
          <div className="flex-1 overflow-hidden">
            <Textarea
              className="h-full min-h-[300px] p-4 font-mono resize-none"
              value={content}
              onChange={(e) => onContentChange(e.target.value)}
              placeholder={placeholder}
            />
          </div>
          <ModalForm.Alert />
          <ModalForm.Footer>
            <ModalForm.CancelButton />
            <ModalForm.SubmitButton label={submitLabel} />
          </ModalForm.Footer>
        </ModalForm.Provider>
      </DialogContent>
    </Dialog>
  )
}
