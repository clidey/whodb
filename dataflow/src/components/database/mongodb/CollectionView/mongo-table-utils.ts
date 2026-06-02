import type {
  DocumentChange,
  DocumentChangesetRowKey,
  RenderedMongoDocument,
} from './types'
import { buildExistingRowKey } from './useDocumentChangesetManager'

export type MongoScalarValue = string | number | boolean | null
export type MongoComplexValue = Record<string, unknown> | unknown[]
export type MongoCellValue = MongoScalarValue | MongoComplexValue
export type MongoCellCoercionError = 'invalid-number' | 'invalid-boolean' | 'complex-value'

export type MongoCellCoercionResult =
  | { ok: true; value: MongoCellValue }
  | { ok: false; error: MongoCellCoercionError }

export type MongoFieldJsonParseResult =
  | { ok: true; value: unknown }
  | { ok: false; error: string }

/** Returns whether a value can be edited directly in a MongoDB table cell. */
export function isMongoScalarValue(value: unknown): value is MongoScalarValue {
  return value === null || typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
}

function parseMongoJsonObjectOrArrayDraft(draftValue: string): MongoComplexValue | null {
  const trimmedDraft = draftValue.trim()
  if (!trimmedDraft) return null

  try {
    const value = JSON.parse(trimmedDraft) as unknown
    if (Array.isArray(value)) return value
    if (value !== null && typeof value === 'object') return value as Record<string, unknown>
  } catch {
    return null
  }

  return null
}

function parseMongoQuotedComplexLiteralDraft(draftValue: string): string | null {
  const trimmedDraft = draftValue.trim()
  if (!trimmedDraft) return null

  try {
    const value = JSON.parse(trimmedDraft) as unknown
    if (typeof value !== 'string') return null
    return parseMongoJsonObjectOrArrayDraft(value) ? value : null
  } catch {
    return null
  }
}

/** Returns whether a document owns the requested top-level field. */
export function hasDocumentField(doc: Record<string, unknown>, field: string): boolean {
  return Object.prototype.hasOwnProperty.call(doc, field)
}

/** Builds the document rows that should be rendered after applying pending changes. */
export function buildRenderedMongoDocuments({
  documents,
  changes,
  newRowOrder,
  pageOffset,
}: {
  documents: Record<string, unknown>[]
  changes: Map<DocumentChangesetRowKey, DocumentChange>
  newRowOrder: DocumentChangesetRowKey[]
  pageOffset: number
}): RenderedMongoDocument[] {
  const inserted = newRowOrder
    .map((rowKey): RenderedMongoDocument | null => {
      const change = changes.get(rowKey)
      if (!change) return null

      return {
        rowKey,
        doc: change.document,
        originalDocument: change.originalDocument,
        changeType: change.type,
        isDeleted: false,
        isInserted: true,
        rowNumber: null,
      }
    })
    .filter((item): item is RenderedMongoDocument => item !== null)

  const existing = documents.map((doc, idx): RenderedMongoDocument => {
    const rowKey = buildExistingRowKey(pageOffset, idx)
    const change = changes.get(rowKey)
    return {
      rowKey,
      doc: change?.document ?? doc,
      originalDocument: change?.originalDocument ?? doc,
      changeType: change?.type ?? null,
      isDeleted: change?.type === 'delete',
      isInserted: false,
      rowNumber: pageOffset + idx + 1,
    }
  })

  return [...inserted, ...existing]
}

/** Builds the table column order from sampled fields, current documents, and pending changes. */
export function buildMongoTableColumns({
  sampledFields,
  documents,
  changes,
}: {
  sampledFields: string[]
  documents: Record<string, unknown>[]
  changes: Map<DocumentChangesetRowKey, DocumentChange>
}): string[] {
  const fields = new Set<string>(['_id', ...sampledFields])

  for (const doc of documents) {
    Object.keys(doc).forEach((field) => fields.add(field))
  }

  for (const change of changes.values()) {
    Object.keys(change.document).forEach((field) => fields.add(field))
  }

  const sorted = [...fields].filter((field) => field !== '_id').sort((left, right) => left.localeCompare(right))
  return ['_id', ...sorted]
}

/** Returns whether a rendered table cell differs from its original document value. */
export function isMongoCellChanged(
  originalDocument: Record<string, unknown>,
  document: Record<string, unknown>,
  field: string,
): boolean {
  const originalHasField = hasDocumentField(originalDocument, field)
  const nextHasField = hasDocumentField(document, field)
  if (originalHasField !== nextHasField) return true
  return !Object.is(originalDocument[field], document[field])
}

/** Converts an edited cell draft back to the correct MongoDB scalar value. */
export function coerceMongoCellDraft(
  existingValue: unknown,
  draftValue: string,
  fieldExists: boolean,
): MongoCellCoercionResult {
  const complexValue = parseMongoJsonObjectOrArrayDraft(draftValue)
  if (complexValue !== null) return { ok: true, value: complexValue }

  if (!fieldExists || existingValue === null) {
    const complexLiteral = parseMongoQuotedComplexLiteralDraft(draftValue)
    if (complexLiteral !== null) return { ok: true, value: complexLiteral }
    return { ok: true, value: draftValue }
  }

  if (typeof existingValue === 'string') {
    const complexLiteral = parseMongoQuotedComplexLiteralDraft(draftValue)
    if (complexLiteral !== null) return { ok: true, value: complexLiteral }
    return { ok: true, value: draftValue }
  }

  if (typeof existingValue === 'number') {
    const nextValue = Number(draftValue)
    if (Number.isNaN(nextValue)) return { ok: false, error: 'invalid-number' }
    return { ok: true, value: nextValue }
  }

  if (typeof existingValue === 'boolean') {
    const normalized = draftValue.trim().toLowerCase()
    if (normalized === 'true') return { ok: true, value: true }
    if (normalized === 'false') return { ok: true, value: false }
    return { ok: false, error: 'invalid-boolean' }
  }

  return { ok: false, error: 'complex-value' }
}

/** Parses a field-level JSON editor draft into a MongoDB field value. */
export function parseMongoFieldJsonDraft(content: string): MongoFieldJsonParseResult {
  try {
    return { ok: true, value: JSON.parse(content) }
  } catch (error) {
    return { ok: false, error: error instanceof Error ? error.message : String(error) }
  }
}
