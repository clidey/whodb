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

export interface ParsedMongoDocumentRow {
  document: Record<string, unknown>
  fieldOrder: string[]
}

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

function skipJsonWhitespace(content: string, startIndex: number): number {
  let index = startIndex
  while (index < content.length && /\s/.test(content[index])) index += 1
  return index
}

function readJsonStringLiteral(content: string, startIndex: number): { value: string; nextIndex: number } {
  for (let index = startIndex + 1; index < content.length; index += 1) {
    if (content[index] === '\\') {
      index += 1
      continue
    }

    if (content[index] === '"') {
      return {
        value: JSON.parse(content.slice(startIndex, index + 1)) as string,
        nextIndex: index + 1,
      }
    }
  }

  return { value: '', nextIndex: content.length }
}

function skipJsonValue(content: string, startIndex: number): number {
  let depth = 0
  let inString = false
  let escaped = false

  for (let index = skipJsonWhitespace(content, startIndex); index < content.length; index += 1) {
    const character = content[index]

    if (inString) {
      if (escaped) {
        escaped = false
        continue
      }
      if (character === '\\') {
        escaped = true
        continue
      }
      if (character === '"') inString = false
      continue
    }

    if (character === '"') {
      inString = true
      continue
    }

    if (character === '{' || character === '[') {
      depth += 1
      continue
    }

    if (character === '}' || character === ']') {
      if (depth === 0) return index
      depth -= 1
      continue
    }

    if (character === ',' && depth === 0) return index
  }

  return content.length
}

function readTopLevelJsonObjectKeys(content: string): string[] {
  const keys: string[] = []
  let index = skipJsonWhitespace(content, 0)
  if (content[index] !== '{') return keys

  index += 1
  while (index < content.length) {
    index = skipJsonWhitespace(content, index)
    if (content[index] === '}') return keys

    const key = readJsonStringLiteral(content, index)
    keys.push(key.value)

    index = skipJsonWhitespace(content, key.nextIndex)
    if (content[index] === ':') index += 1

    index = skipJsonWhitespace(content, skipJsonValue(content, index))
    if (content[index] === ',') {
      index += 1
      continue
    }
    if (content[index] === '}') return keys
  }

  return keys
}

/** Parses one MongoDB document row and preserves its top-level field order. */
export function parseMongoDocumentRow(content: string): ParsedMongoDocumentRow {
  return {
    document: JSON.parse(content) as Record<string, unknown>,
    fieldOrder: readTopLevelJsonObjectKeys(content),
  }
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

/** Builds the table column order from current documents and pending changes. */
export function buildMongoTableColumns({
  documents,
  documentFieldOrders,
  changes,
}: {
  documents: Record<string, unknown>[]
  documentFieldOrders: string[][]
  changes: Map<DocumentChangesetRowKey, DocumentChange>
}): string[] {
  const columns = ['_id']
  const fields = new Set<string>(columns)

  const addField = (field: string) => {
    if (fields.has(field)) return
    fields.add(field)
    columns.push(field)
  }

  for (const [index, doc] of documents.entries()) {
    const fieldOrder = documentFieldOrders[index] ?? []
    fieldOrder.forEach((field) => {
      if (hasDocumentField(doc, field)) addField(field)
    })
    Object.keys(doc).forEach(addField)
  }

  for (const change of changes.values()) {
    Object.keys(change.document).forEach(addField)
  }

  return columns
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
