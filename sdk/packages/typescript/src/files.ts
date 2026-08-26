import type { CredentialProvider } from './auth.js';
import { PlatformError } from './errors.js';
import { SDK_VERSION } from './version.js';

/** Options for the files facade. */
export interface FilesOptions {
  host: string;
  credentials: CredentialProvider;
  orgId: () => Promise<string>;
  projectId: () => Promise<string>;
}

/** Metadata returned after uploading a file. */
export interface UploadedFile {
  id: string;
  name: string;
}

/**
 * FilesHandle wraps the platform's file HTTP endpoints — bulk payloads are
 * deliberately not squeezed through GraphQL (SDK_DESIGN.md §4).
 */
export class FilesHandle {
  private readonly options: FilesOptions;

  constructor(options: FilesOptions) {
    this.options = options;
  }

  /** Uploads a file (multipart) into the project's file storage. */
  async upload(name: string, content: Blob | Uint8Array | string, folderId?: string): Promise<UploadedFile> {
    const token = await this.options.credentials.token();
    const form = new FormData();
    const blob = content instanceof Blob ? content : new Blob([content as BlobPart]);
    form.set('file', blob, name);
    if (folderId) form.set('folderId', folderId);
    form.set('projectId', await this.options.projectId());
    const response = await fetch(`${this.options.host.replace(/\/$/, '')}/api/files/upload`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'User-Agent': `clidey-whodb-ts/${SDK_VERSION}`,
        'X-Whodb-Org-Id': await this.options.orgId(),
        'X-Whodb-Project-Id': await this.options.projectId(),
      },
      body: form,
    });
    if (!response.ok) {
      throw new PlatformError(`file upload failed with HTTP ${response.status}`, `HTTP_${response.status}`);
    }
    return (await response.json()) as UploadedFile;
  }

  /** Downloads a project file's raw bytes. */
  async download(fileId: string): Promise<Uint8Array> {
    const token = await this.options.credentials.token();
    const projectId = await this.options.projectId();
    const response = await fetch(
      `${this.options.host.replace(/\/$/, '')}/api/projects/${encodeURIComponent(projectId)}/files/${encodeURIComponent(fileId)}/download`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
          'User-Agent': `clidey-whodb-ts/${SDK_VERSION}`,
          'X-Whodb-Org-Id': await this.options.orgId(),
          'X-Whodb-Project-Id': projectId,
        },
      },
    );
    if (!response.ok) {
      throw new PlatformError(`file download failed with HTTP ${response.status}`, `HTTP_${response.status}`);
    }
    return new Uint8Array(await response.arrayBuffer());
  }
}
