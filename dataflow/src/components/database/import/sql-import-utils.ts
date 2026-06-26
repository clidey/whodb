import type { ImportSqlInput } from '@graphql'

export type SqlScriptSource =
  | {
      kind: 'file'
      file: File
      filename: string
      preview: string
    }
  | {
      kind: 'text'
      script: string
    }

/** Returns whether a selected file can be used as a SQL script source. */
export function isAcceptedSqlFile(file: File): boolean {
  return file.name.toLowerCase().endsWith('.sql')
}

/** Returns whether the active SQL script source contains executable content. */
export function hasSqlScriptSource(source: SqlScriptSource | null): boolean {
  if (!source) return false
  if (source.kind === 'file') return source.preview.trim().length > 0
  return source.script.trim().length > 0
}

/** Builds the GraphQL input for the active SQL script source. */
export function buildImportSqlInput(source: SqlScriptSource): ImportSqlInput {
  if (source.kind === 'file') {
    return {
      File: source.file,
      Filename: source.filename,
    }
  }

  return {
    Script: source.script,
  }
}
