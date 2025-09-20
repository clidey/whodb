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

import { Alert, AlertTitle, AlertDescription, Button, toast } from "@clidey/ux";
import { BellAlertIcon, CheckCircleIcon, ClipboardDocumentIcon } from "./heroicons";
import { useState } from "react";

interface ErrorStateProps {
    error?: { message?: string } | string | null;
}

export const ErrorState = ({ error }: ErrorStateProps) => {
    const [copied, setCopied] = useState(false);
    const message = typeof error === "string" ? error : error?.message ?? "An unknown error occurred.";

    const handleCopy = () => {
        navigator.clipboard.writeText(message);
        setCopied(true);
        toast.success("Copied to clipboard");
    };

    return (
        <Alert variant="destructive" title="Error" description={message} className="group relative">
            <BellAlertIcon className="w-4 h-4" />
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{message}</AlertDescription>
            <Button
                variant="ghost"
                size="icon"
                className="absolute top-0 right-0 opacity-0 group-hover:opacity-100 transition-all"
                onClick={handleCopy}
                aria-label="Copy error message"
            >
                {copied ? <CheckCircleIcon className="w-4 h-4" /> : <ClipboardDocumentIcon className="w-4 h-4" />}
            </Button>
        </Alert>
    );
};