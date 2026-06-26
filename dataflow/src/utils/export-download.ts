import { addAuthHeader } from '@/config/auth-headers'
import { authFetch } from '@/config/graphql-client'

interface ExportDownloadPayload {
  DownloadURL: string
  Filename: string
}

/** Fetches a server-prepared export file with the active WhoDB auth session. */
export async function fetchExportDownloadBlob(
  download: ExportDownloadPayload,
  databaseName: string,
): Promise<Blob> {
  const response = await authFetch(download.DownloadURL, {
    credentials: 'include',
    headers: addAuthHeader({}, databaseName),
  })

  if (!response.ok) {
    throw new Error(`${response.status} ${response.statusText}`)
  }

  return response.blob()
}
