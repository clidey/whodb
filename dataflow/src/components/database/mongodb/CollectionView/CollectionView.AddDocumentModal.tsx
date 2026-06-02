import { DocumentEditorDialog, type DocumentEditorDialogProps } from './CollectionView.DocumentEditorDialog'
import { useI18n } from '@/i18n/useI18n'

/** Dialog for adding a new MongoDB document with JSON textarea input. */
export function AddDocumentModal(
  props: Omit<DocumentEditorDialogProps, 'title' | 'submitLabel' | 'description' | 'placeholder'>,
) {
  const { t } = useI18n()

  return (
    <DocumentEditorDialog
      title={t('mongodb.document.addTitle')}
      submitLabel={t('mongodb.document.add')}
      description={t('mongodb.document.addDescription')}
      placeholder={t('mongodb.document.placeholder')}
      {...props}
    />
  )
}
