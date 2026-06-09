interface GraphQLErrorPayload {
  message?: unknown
}

interface GraphQLServerErrorLike extends Error {
  result?: unknown
  statusCode?: unknown
}

/** Formats Apollo GraphQL errors with server-provided response details when available. */
export function formatGraphQLErrorMessage(error: unknown): string {
  if (!(error instanceof Error)) return String(error)

  const serverDetails = formatServerResult((error as GraphQLServerErrorLike).result)
  if (!serverDetails || serverDetails === error.message) return error.message

  return `${error.message}\n${serverDetails}`
}

function formatServerResult(result: unknown): string | null {
  if (!result) return null
  if (typeof result === 'string') return result

  if (typeof result !== 'object') return null

  const errors = (result as { errors?: unknown }).errors
  if (Array.isArray(errors)) {
    const messages = errors
      .map(formatGraphQLErrorPayload)
      .filter((message): message is string => Boolean(message))

    if (messages.length > 0) return messages.join('\n')
  }

  try {
    return JSON.stringify(result)
  } catch {
    return null
  }
}

function formatGraphQLErrorPayload(error: unknown): string | null {
  if (!error || typeof error !== 'object') return null

  const message = (error as GraphQLErrorPayload).message
  if (typeof message !== 'string') return null

  return message
}
