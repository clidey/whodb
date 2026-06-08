const DOT_NOTATION_COLLECTION_PATTERN = /^[A-Za-z_][\w.]*$/

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

/** Reads top-level key order from a strict JSON object document. */
export function readMongoDocumentFieldOrder(content: string): string[] {
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

/** Builds a complete document field order from a preferred order and appended new fields. */
export function buildMongoDocumentFieldOrder(
  document: Record<string, unknown>,
  preferredFieldOrder: string[] = [],
): string[] {
  const fields = new Set<string>()
  const fieldOrder: string[] = []

  const addField = (field: string) => {
    if (fields.has(field)) return
    if (!Object.prototype.hasOwnProperty.call(document, field)) return
    fields.add(field)
    fieldOrder.push(field)
  }

  preferredFieldOrder.forEach(addField)
  Object.keys(document).forEach(addField)

  return fieldOrder
}

/** Builds replacement order from authored JSON order, keeping MongoDB `_id` first. */
export function buildMongoEditedDocumentFieldOrder(
  document: Record<string, unknown>,
  editedFieldOrder: string[] = [],
): string[] {
  const preferredOrder = Object.prototype.hasOwnProperty.call(document, '_id')
    ? ['_id', ...editedFieldOrder.filter((field) => field !== '_id')]
    : editedFieldOrder
  return buildMongoDocumentFieldOrder(document, preferredOrder)
}

function stringifyMongoValue(value: unknown, spaces: number): string {
  return JSON.stringify(value, null, spaces) as string
}

function indentMongoValue(value: string, indent: string): string {
  return value.replace(/\n/g, `\n${indent}`)
}

/** Stringifies a MongoDB document using an explicit top-level field order. */
export function stringifyMongoDocument(
  document: Record<string, unknown>,
  fieldOrder: string[] = Object.keys(document),
  spaces = 2,
  excludedFields: string[] = [],
): string {
  const excluded = new Set(excludedFields)
  const keys = buildMongoDocumentFieldOrder(document, fieldOrder).filter((key) => !excluded.has(key))
  if (keys.length === 0) return '{}'

  if (spaces <= 0) {
    const entries = keys.map((key) => `${JSON.stringify(key)}:${JSON.stringify(document[key])}`)
    return `{${entries.join(',')}}`
  }

  const indent = ' '.repeat(spaces)
  const entries = keys.map((key) => {
    const value = indentMongoValue(stringifyMongoValue(document[key], spaces), indent)
    return `${indent}${JSON.stringify(key)}: ${value}`
  })

  return `{\n${entries.join(',\n')}\n}`
}

/** Build a MongoDB shell collection accessor, falling back to getCollection for unsafe names. */
export function buildMongoCollectionAccessor(collectionName: string): string {
  if (DOT_NOTATION_COLLECTION_PATTERN.test(collectionName)) {
    return `db.${collectionName}`
  }

  return `db.getCollection(${JSON.stringify(collectionName)})`
}

/** Build a MongoDB shell command against a collection. */
export function buildMongoCollectionCommand(
  collectionName: string,
  method: string,
  rawArgs = '',
): string {
  return `${buildMongoCollectionAccessor(collectionName)}.${method}(${rawArgs})`
}

/** Parse strict JSON document input from the UI and reject non-object payloads. */
export function parseMongoDocumentInput(content: string): Record<string, unknown> {
  const parsed = JSON.parse(content)

  if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('MongoDB document input must be a JSON object')
  }

  return parsed as Record<string, unknown>
}

/** Parse strict JSON document input and return its top-level field order. */
export function parseMongoDocumentInputWithOrder(content: string): {
  document: Record<string, unknown>
  fieldOrder: string[]
} {
  const document = parseMongoDocumentInput(content)
  return {
    document,
    fieldOrder: buildMongoDocumentFieldOrder(document, readMongoDocumentFieldOrder(content)),
  }
}

/** Build an insertOne command for a parsed MongoDB document payload. */
export function buildMongoInsertOneCommand(
  collectionName: string,
  document: Record<string, unknown>,
  fieldOrder: string[] = Object.keys(document),
): string {
  return buildMongoCollectionCommand(collectionName, 'insertOne', stringifyMongoDocument(document, fieldOrder, 0))
}

/** Build a MongoDB shell command that drops the current database. */
export function buildMongoDropDatabaseCommand(): string {
  return 'db.dropDatabase()'
}
