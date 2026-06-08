import {
  buildMongoCollectionAccessor,
  buildMongoDocumentFieldOrder,
  buildMongoEditedDocumentFieldOrder,
  stringifyMongoDocument,
} from '@/utils/mongodb-shell'
import type { DocumentChange, DocumentChangesetRowKey } from './types'

export interface ChangesetSummary {
  updates: number
  inserts: number
  deletes: number
}

export function summarizeChanges(changes: Map<DocumentChangesetRowKey, DocumentChange>): ChangesetSummary {
  let updates = 0
  let inserts = 0
  let deletes = 0

  for (const change of changes.values()) {
    if (change.type === 'update') updates += 1
    if (change.type === 'insert') inserts += 1
    if (change.type === 'delete') deletes += 1
  }

  return { updates, inserts, deletes }
}

export function buildPreviewCommands(
  collectionName: string,
  changes: Map<DocumentChangesetRowKey, DocumentChange>,
): string[] {
  const accessor = buildMongoCollectionAccessor(collectionName)

  return [...changes.values()].map((change) => {
    if (change.type !== 'update') {
      if (change.type === 'insert') {
        return `${accessor}.insertOne(${stringifyMongoDocument(change.document, change.fieldOrder, 2)});`
      }
      return `${accessor}.deleteOne({ _id: ${JSON.stringify(change.originalDocument._id)} });`
    }

    if (change.saveMode === 'replace') {
      const replacementDocument = { ...change.document, _id: change.originalDocument._id }
      const replacementFieldOrder = buildMongoEditedDocumentFieldOrder(replacementDocument, change.fieldOrder)
      return `${accessor}.replaceOne(\n  { _id: ${JSON.stringify(change.originalDocument._id)} },\n  ${stringifyMongoDocument(replacementDocument, replacementFieldOrder, 2)}\n);`
    }

    const updateFields = { ...change.document }
    delete updateFields._id
    const updateFieldOrder = buildMongoDocumentFieldOrder(
      updateFields,
      change.fieldOrder.filter((field) => field !== '_id'),
    )
    return `${accessor}.updateOne(\n  { _id: ${JSON.stringify(change.originalDocument._id)} },\n  { $set: ${stringifyMongoDocument(updateFields, updateFieldOrder, 2)} }\n);`
  })
}
