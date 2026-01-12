/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Alert, AlertDescription, Button, Input, Label, TextArea, cn } from '@clidey/ux';
import { FC, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useGetSslModesLazyQuery, SslModeOption } from '@graphql';
import { SearchSelect } from './ux';
import { DocumentTextIcon, ExclamationCircleIcon, FolderIcon } from './heroicons';
import { useTranslation } from '@/hooks/use-translation';

// SSL configuration keys that match the backend constants.
// Note: Path-based keys are intentionally not supported to prevent path traversal attacks.
// Frontend reads certificate files client-side and sends Content directly.
export const SSL_KEYS = {
  MODE: 'SSL Mode',
  CA_CONTENT: 'SSL CA Content',
  CLIENT_CERT_CONTENT: 'SSL Client Cert Content',
  CLIENT_KEY_CONTENT: 'SSL Client Key Content',
  SERVER_NAME: 'SSL Server Name',
} as const;

// Modes that require CA certificate
const MODES_REQUIRING_CA = ['verify-ca', 'verify-identity', 'enabled'];

// Modes that require client certificate for mutual TLS (optional for most)
const MODES_SUPPORTING_CLIENT_CERT = ['verify-ca', 'verify-identity', 'enabled'];

// Databases that only support system CAs (driver limitation - can't inject custom CA content)
const SYSTEM_CA_ONLY_DATABASES = ['MSSQL', 'Oracle'];

export interface SSLConfigProps {
  /** Database type to fetch SSL modes for */
  databaseType: string;
  /** Current advanced form values */
  advancedForm: Record<string, string>;
  /** Handler to update advanced form values */
  onAdvancedFormChange: (key: string, value: string) => void;
  /** Optional className for the container */
  className?: string;
}

/**
 * Detects if the current page is served over an insecure connection (HTTP).
 */
function isInsecureConnection(): boolean {
  return typeof window !== 'undefined' && window.location.protocol === 'http:' && window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1';
}

/**
 * SSL Configuration component that provides:
 * - Mode dropdown with database-specific options
 * - Certificate inputs with file picker or paste PEM
 * - HTTP security warning for private keys
 */
export const SSLConfig: FC<SSLConfigProps> = ({
  databaseType,
  advancedForm,
  onAdvancedFormChange,
  className,
}) => {
  const { t } = useTranslation('components/ssl-config');
  const [getSSLModes, { data: sslModesData, loading }] = useGetSslModesLazyQuery();
  const [inputModes, setInputModes] = useState<Record<string, 'file' | 'paste'>>({
    ca: 'file',
    clientCert: 'file',
    clientKey: 'file',
  });
  const [showHttpWarning, setShowHttpWarning] = useState(false);

  // Fetch SSL modes when database type changes
  useEffect(() => {
    if (databaseType) {
      getSSLModes({ variables: { type: databaseType } });
    }
  }, [databaseType, getSSLModes]);

  // Check for insecure connection when component mounts
  useEffect(() => {
    setShowHttpWarning(isInsecureConnection());
  }, []);

  const sslModes = useMemo(() => sslModesData?.SSLModes ?? [], [sslModesData]);
  const rawMode = advancedForm[SSL_KEYS.MODE] || 'disabled';

  // Normalize the current mode - if it matches an alias, convert to canonical value.
  // This handles cases where profiles use database-native names like PostgreSQL's "require"
  // instead of our canonical "required".
  const currentMode = useMemo(() => {
    // First check if the raw mode is a canonical value
    const exactMatch = sslModes.find((m: SslModeOption) => m.Value === rawMode);
    if (exactMatch) return rawMode;

    // Check if the raw mode is an alias for any canonical mode
    const aliasMatch = sslModes.find((m: SslModeOption) =>
      m.Aliases?.includes(rawMode)
    );
    if (aliasMatch) return aliasMatch.Value;

    // Fallback to the raw mode (might be invalid, will show as no selection)
    return rawMode;
  }, [rawMode, sslModes]);

  // Check if the current mode requires certificates
  const requiresCA = useMemo(
    () => MODES_REQUIRING_CA.includes(currentMode),
    [currentMode]
  );
  const supportsClientCert = useMemo(
    () => MODES_SUPPORTING_CLIENT_CERT.includes(currentMode),
    [currentMode]
  );

  // Check if database supports custom CA certificates (some drivers only support system CAs)
  const supportsCustomCA = useMemo(
    () => !SYSTEM_CA_ONLY_DATABASES.includes(databaseType),
    [databaseType]
  );

  // Handle mode change
  const handleModeChange = useCallback(
    (mode: string) => {
      onAdvancedFormChange(SSL_KEYS.MODE, mode);

      // Clear certificate fields when switching to disabled
      if (mode === 'disabled') {
        onAdvancedFormChange(SSL_KEYS.CA_CONTENT, '');
        onAdvancedFormChange(SSL_KEYS.CLIENT_CERT_CONTENT, '');
        onAdvancedFormChange(SSL_KEYS.CLIENT_KEY_CONTENT, '');
        onAdvancedFormChange(SSL_KEYS.SERVER_NAME, '');
      }
    },
    [onAdvancedFormChange]
  );

  // Toggle between file picker and paste mode
  const toggleInputMode = useCallback((field: 'ca' | 'clientCert' | 'clientKey') => {
    setInputModes(prev => ({
      ...prev,
      [field]: prev[field] === 'file' ? 'paste' : 'file',
    }));
  }, []);

  // Get localized label for a mode
  const getModeLabel = useCallback((mode: SslModeOption) => {
    return t(`modes.${mode.Value}.label`);
  }, [t]);

  // Get localized description for a mode
  const getModeDescription = useCallback((mode: SslModeOption) => {
    return t(`modes.${mode.Value}.description`);
  }, [t]);

  // Mode dropdown options with localized labels
  const modeOptions = useMemo(() => sslModes.map((mode: SslModeOption) => ({
    value: mode.Value,
    label: getModeLabel(mode),
  })), [sslModes, getModeLabel]);

  // Get current mode info for description
  const currentModeInfo = useMemo(
    () => sslModes.find((m: SslModeOption) => m.Value === currentMode),
    [sslModes, currentMode]
  );

  // Don't render if no SSL modes available (e.g., SQLite)
  if (sslModes.length === 0 && !loading) {
    return null;
  }

  return (
    <div className={cn('flex flex-col gap-md', className)}>
      {/* HTTP Security Warning */}
      {showHttpWarning && currentMode !== 'disabled' && (
        <Alert variant="destructive">
          <ExclamationCircleIcon className="h-4 w-4" />
          <AlertDescription>
            {t('httpWarning')}
          </AlertDescription>
        </Alert>
      )}

      {/* SSL Mode Dropdown */}
      <div className="flex flex-col gap-sm">
        <Label htmlFor="ssl-mode-select">{t('sslMode')}</Label>
        <SearchSelect
          options={modeOptions}
          value={currentMode}
          onChange={handleModeChange}
          placeholder={t('selectMode')}
          buttonProps={{
            "data-testid": "ssl-mode-select",
            disabled: loading,
          }}
        />
        {currentModeInfo && (
          <span className="text-xs text-muted-foreground">
            {getModeDescription(currentModeInfo)}
          </span>
        )}
      </div>

      {/* System CA Info (for databases that don't support custom CA upload) */}
      {requiresCA && !supportsCustomCA && (
        <div className="text-sm text-muted-foreground bg-muted/50 rounded-md p-3">
          {t('systemCaOnly')}
        </div>
      )}

      {/* CA Certificate Input */}
      {requiresCA && supportsCustomCA && (
        <CertificateInput
          label={t('caCertificate')}
          contentValue={advancedForm[SSL_KEYS.CA_CONTENT] || ''}
          inputMode={inputModes.ca}
          onToggleMode={() => toggleInputMode('ca')}
          onContentChange={(value) => onAdvancedFormChange(SSL_KEYS.CA_CONTENT, value)}
          testIdPrefix="ssl-ca-certificate"
        />
      )}

      {/* Client Certificate Input (Optional) */}
      {supportsClientCert && supportsCustomCA && (
        <>
          <CertificateInput
            label={t('clientCertificate')}
            contentValue={advancedForm[SSL_KEYS.CLIENT_CERT_CONTENT] || ''}
            inputMode={inputModes.clientCert}
            onToggleMode={() => toggleInputMode('clientCert')}
            onContentChange={(value) => onAdvancedFormChange(SSL_KEYS.CLIENT_CERT_CONTENT, value)}
            testIdPrefix="ssl-client-certificate"
            optional
          />
          <CertificateInput
            label={t('clientKey')}
            contentValue={advancedForm[SSL_KEYS.CLIENT_KEY_CONTENT] || ''}
            inputMode={inputModes.clientKey}
            onToggleMode={() => toggleInputMode('clientKey')}
            onContentChange={(value) => onAdvancedFormChange(SSL_KEYS.CLIENT_KEY_CONTENT, value)}
            testIdPrefix="ssl-client-private-key"
            optional
            isPrivateKey
            showHttpWarning={showHttpWarning}
          />
        </>
      )}

      {/* Server Name Override (for verify-identity) */}
      {currentMode === 'verify-identity' && (
        <div className="flex flex-col gap-sm">
          <Label htmlFor="ssl-server-name">
            {t('serverName')}
            <span className="text-xs text-muted-foreground ml-1">({t('optional')})</span>
          </Label>
          <Input
            id="ssl-server-name"
            value={advancedForm[SSL_KEYS.SERVER_NAME] || ''}
            onChange={(e) => onAdvancedFormChange(SSL_KEYS.SERVER_NAME, e.target.value)}
            placeholder={t('serverNamePlaceholder')}
            data-testid="ssl-server-name-input"
          />
        </div>
      )}
    </div>
  );
};

interface CertificateInputProps {
  label: string;
  contentValue: string;
  inputMode: 'file' | 'paste';
  onToggleMode: () => void;
  onContentChange: (value: string) => void;
  testIdPrefix: string;
  optional?: boolean;
  isPrivateKey?: boolean;
  showHttpWarning?: boolean;
}

/**
 * Certificate input with file picker or paste PEM mode.
 * File picker reads the file content and stores it (for web uploads).
 * Paste mode allows manual PEM content entry.
 */
const CertificateInput: FC<CertificateInputProps> = ({
  label,
  contentValue,
  inputMode,
  onToggleMode,
  onContentChange,
  testIdPrefix,
  optional,
  isPrivateKey,
  showHttpWarning,
}) => {
  const { t } = useTranslation('components/ssl-config');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [fileName, setFileName] = useState<string>('');
  const [fileError, setFileError] = useState<string>('');

  // Handle file selection
  const handleFileSelect = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setFileError('');
    setFileName(file.name);

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      if (content) {
        onContentChange(content);
      }
    };
    reader.onerror = () => {
      setFileError(t('fileReadError'));
      setFileName('');
    };
    reader.readAsText(file);
  }, [onContentChange, t]);

  // Trigger file picker
  const handleChooseFile = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  // Clear the selected file
  const handleClearFile = useCallback(() => {
    setFileName('');
    onContentChange('');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, [onContentChange]);

  return (
    <div className="flex flex-col gap-sm">
      <div className="flex items-center justify-between">
        <Label>
          {label}
          {optional && (
            <span className="text-xs text-muted-foreground ml-1">({t('optional')})</span>
          )}
        </Label>
        <Button
          variant="ghost"
          size="sm"
          onClick={onToggleMode}
          className="h-6 px-2 text-xs"
          title={inputMode === 'file' ? t('switchToPaste') : t('switchToFile')}
        >
          {inputMode === 'file' ? (
            <>
              <DocumentTextIcon className="w-3 h-3 mr-1" />
              {t('pasteContent')}
            </>
          ) : (
            <>
              <FolderIcon className="w-3 h-3 mr-1" />
              {t('chooseFile')}
            </>
          )}
        </Button>
      </div>

      {/* Private key HTTP warning */}
      {isPrivateKey && showHttpWarning && inputMode === 'file' && (
        <span className="text-xs text-destructive">
          {t('privateKeyHttpWarning')}
        </span>
      )}

      {inputMode === 'file' ? (
        <div className="flex flex-col gap-sm">
          {/* Hidden file input */}
          <input
            ref={fileInputRef}
            type="file"
            accept=".pem,.crt,.cer,.key"
            onChange={handleFileSelect}
            className="hidden"
            data-testid={`${testIdPrefix}-file-input`}
          />

          {/* File picker button */}
          <div className="flex items-center gap-sm">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleChooseFile}
              className="flex-shrink-0"
              data-testid={`${testIdPrefix}-choose-file`}
            >
              <FolderIcon className="w-4 h-4 mr-2" />
              {t('chooseFile')}
            </Button>
            {fileName && (
              <div className="flex items-center gap-sm flex-1 min-w-0">
                <span className="text-sm text-muted-foreground truncate" title={fileName}>
                  {fileName}
                </span>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={handleClearFile}
                  className="h-6 px-2 text-xs flex-shrink-0"
                >
                  {t('clear')}
                </Button>
              </div>
            )}
            {!fileName && contentValue && (
              <span className="text-sm text-muted-foreground">
                {t('contentLoaded')}
              </span>
            )}
          </div>

          {fileError && (
            <span className="text-xs text-destructive">{fileError}</span>
          )}
        </div>
      ) : (
        <TextArea
          value={contentValue}
          onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => onContentChange(e.target.value)}
          placeholder={"-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"}
          className="font-mono text-xs h-24"
          data-testid={`${testIdPrefix}-content`}
        />
      )}
    </div>
  );
};