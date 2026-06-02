import { Dialog, DialogContent } from '@/components/ui/dialog'
import { ModalForm } from '@/components/ui/ModalForm'
import { Textarea } from '@/components/ui/Textarea'
import { useI18n } from '@/i18n/useI18n'

interface FieldJsonEditorDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fieldName: string
  content: string
  onContentChange: (content: string) => void
  onSave: () => Promise<void>
}

/** Dialog for editing one MongoDB document field as a JSON value. */
export function FieldJsonEditorDialog({
  open,
  onOpenChange,
  fieldName,
  content,
  onContentChange,
  onSave,
}: FieldJsonEditorDialogProps) {
  const { t } = useI18n()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-xl"
        data-testid="mongodb.collection.field-json-dialog"
        data-qa-module="mongodb"
        data-qa-object="field-json-editor"
        data-qa-state={open ? 'open' : 'closed'}
        data-qa-field={fieldName}
      >
        <ModalForm.Provider
          onSubmit={onSave}
          meta={{
            title: t('mongodb.fieldJson.editTitle', { field: fieldName }),
            description: t('mongodb.fieldJson.description'),
          }}
        >
          <ModalForm.Header />
          <Textarea
            className="min-h-[320px] resize-none p-4 font-mono"
            value={content}
            onChange={(event) => onContentChange(event.target.value)}
            data-testid="mongodb.collection.field-json-textarea"
            data-qa-module="mongodb"
            data-qa-object="field-json-editor"
            data-qa-action="edit"
            data-qa-field={fieldName}
          />
          <ModalForm.Alert />
          <ModalForm.Footer>
            <ModalForm.CancelButton />
            <ModalForm.SubmitButton label={t('mongodb.document.saveChanges')} />
          </ModalForm.Footer>
        </ModalForm.Provider>
      </DialogContent>
    </Dialog>
  )
}
