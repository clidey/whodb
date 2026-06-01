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

export const ErrorState = ({ error }: ErrorStateProps) => {
    const { t } = useTranslation('components/error-state');
    const message = typeof error === "string" ? error : error?.message ?? t('unknownError');

    return (
        <Alert variant="destructive" title={t('title')} description={message} className="group relative" role="alert" data-testid="error-state">
            <BellAlertIcon className="w-4 h-4" aria-hidden="true" />
            <AlertTitle>{t('title')}</AlertTitle>
            <AlertDescription>{message}</AlertDescription>
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