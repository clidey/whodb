import type { CredentialProvider } from './auth.js';
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
export declare class FilesHandle {
    private readonly options;
    constructor(options: FilesOptions);
    /** Uploads a file (multipart) into the project's file storage. */
    upload(name: string, content: Blob | Uint8Array | string, folderId?: string): Promise<UploadedFile>;
    /** Downloads a project file's raw bytes. */
    download(fileId: string): Promise<Uint8Array>;
}
