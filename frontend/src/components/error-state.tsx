/*
 * Copyright 2025 Clidey, Inc.
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

import {Alert, AlertDescription, AlertTitle, CopyButton, toast} from "@clidey/ux";
import {BellAlertIcon} from "./heroicons";
import {useTranslation} from "@/hooks/use-translation";

interface ErrorStateProps {
    error?: { message?: string } | string | null;
}

interface ParsedError {
    explanation: string;
    raw: string;
}

function parseError(message: string, t: (key: string) => string): ParsedError | null {
    const lower = message.toLowerCase();

    if (/column .+ does not exist/i.test(message) || /unknown column/i.test(message)) {
        return { explanation: t('hint.columnNotFound'), raw: message };
    }
    if (/relation .+ does not exist/i.test(message) || /table .+ doesn't exist/i.test(message)) {
        return { explanation: t('hint.tableNotFound'), raw: message };
    }
    if (/syntax error/i.test(message)) {
        return { explanation: t('hint.syntaxError'), raw: message };
    }
    if (/permission denied/i.test(message) || /access denied/i.test(message)) {
        return { explanation: t('hint.permissionDenied'), raw: message };
    }
    if (/duplicate key/i.test(message) || /unique constraint/i.test(message) || /duplicate entry/i.test(message)) {
        return { explanation: t('hint.duplicateKey'), raw: message };
    }
    if (/null value in column/i.test(message) || /cannot be null/i.test(message) || /violates not-null/i.test(message)) {
        return { explanation: t('hint.notNull'), raw: message };
    }
    if (/foreign key constraint/i.test(message) || /violates foreign key/i.test(message)) {
        return { explanation: t('hint.foreignKey'), raw: message };
    }
    if (lower.includes('timeout') || lower.includes('timed out')) {
        return { explanation: t('hint.timeout'), raw: message };
    }
    if (lower.includes('connection refused') || lower.includes('could not connect') || lower.includes('no such host')) {
        return { explanation: t('hint.connectionRefused'), raw: message };
    }
    if (lower.includes('authentication failed') || lower.includes('password authentication') || /login failed/i.test(message)) {
        return { explanation: t('hint.authFailed'), raw: message };
    }

    return null;
}

export const ErrorState = ({ error }: ErrorStateProps) => {
    const { t } = useTranslation('components/error-state');
    const message = typeof error === "string" ? error : error?.message ?? t('unknownError');
    const parsed = parseError(message, t);

    return (
        <Alert variant="destructive" className="group relative" role="alert" data-testid="error-state">
            <BellAlertIcon className="w-4 h-4" aria-hidden="true" />
            <AlertTitle>{t('title')}</AlertTitle>
            {parsed ? (
                <AlertDescription>
                    <p className="font-medium">{parsed.explanation}</p>
                    <p className="mt-1 text-xs opacity-75 font-mono">{parsed.raw}</p>
                </AlertDescription>
            ) : (
                <AlertDescription>{message}</AlertDescription>
            )}
            <CopyButton
                text={message}
                variant="ghost"
                size="icon"
                className="absolute top-0 right-0 opacity-0 group-hover:opacity-100 focus:opacity-100 transition-all"
                onCopy={() => toast.success(t('copiedToClipboard'))}
            />
        </Alert>
    );
};