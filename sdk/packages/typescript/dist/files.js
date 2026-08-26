import { PlatformError } from './errors.js';
import { SDK_VERSION } from './version.js';
/**
 * FilesHandle wraps the platform's file HTTP endpoints — bulk payloads are
 * deliberately not squeezed through GraphQL (SDK_DESIGN.md §4).
 */
export class FilesHandle {
    options;
    constructor(options) {
        this.options = options;
    }
    /** Uploads a file (multipart) into the project's file storage. */
    async upload(name, content, folderId) {
        const token = await this.options.credentials.token();
        const form = new FormData();
        const blob = content instanceof Blob ? content : new Blob([content]);
        form.set('file', blob, name);
        if (folderId)
            form.set('folderId', folderId);
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
        return (await response.json());
    }
    /** Downloads a project file's raw bytes. */
    async download(fileId) {
        const token = await this.options.credentials.token();
        const projectId = await this.options.projectId();
        const response = await fetch(`${this.options.host.replace(/\/$/, '')}/api/projects/${encodeURIComponent(projectId)}/files/${encodeURIComponent(fileId)}/download`, {
            headers: {
                Authorization: `Bearer ${token}`,
                'User-Agent': `clidey-whodb-ts/${SDK_VERSION}`,
                'X-Whodb-Org-Id': await this.options.orgId(),
                'X-Whodb-Project-Id': projectId,
            },
        });
        if (!response.ok) {
            throw new PlatformError(`file download failed with HTTP ${response.status}`, `HTTP_${response.status}`);
        }
        return new Uint8Array(await response.arrayBuffer());
    }
}
