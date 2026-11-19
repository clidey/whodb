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

import {useApolloClient} from "@apollo/client/react";
import {toast} from "@clidey/ux";
import {GetStorageUnitsDocument} from "@graphql";
import {useEffect, useState} from "react";
import {useSourceContract} from "@/hooks/useSourceContract";
import {useTranslation} from "@/hooks/use-translation";
import {useAppSelector} from "@/store/hooks";
import {matchesShortcut, SHORTCUTS} from "@/utils/shortcuts";
import {ImportData} from "./import-data";

/** Opens source-wide SQL import from commands and shortcuts without requiring a table. */
export const SourceImport = () => {
    const current = useAppSelector(state => state.auth.current);
    const {supportsSqlImport} = useSourceContract(current?.Type);
    const {t} = useTranslation("components/import-data");
    const client = useApolloClient();
    const [openSource, setOpenSource] = useState<string | null>(null);
    const sourceKey = JSON.stringify([current?.Id, current?.Type, current?.Database]);

    useEffect(() => {
        if (!current) return;

        const openImport = () => {
            if (supportsSqlImport) setOpenSource(sourceKey);
            else toast.info(t("selectTableToImport"));
        };
        const handleImport = (event: Event) => {
            // Give the active table its CSV/Excel import before opening source-wide SQL import.
            queueMicrotask(() => {
                if (!event.defaultPrevented) openImport();
            });
        };
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.target instanceof HTMLElement && event.target.closest('input, textarea, [contenteditable="true"], [role="dialog"]')) return;
            if (!matchesShortcut(event, SHORTCUTS.importData)) return;
            if (event.defaultPrevented) return;
            event.preventDefault();
            window.dispatchEvent(new CustomEvent("menu:trigger-import", {cancelable: true}));
        };
        window.addEventListener("menu:trigger-import", handleImport);
        window.addEventListener("menu:trigger-sql-import", openImport);
        window.addEventListener("keydown", handleKeyDown);
        return () => {
            window.removeEventListener("menu:trigger-import", handleImport);
            window.removeEventListener("menu:trigger-sql-import", openImport);
            window.removeEventListener("keydown", handleKeyDown);
        };
    }, [current, sourceKey, supportsSqlImport, t]);

    if (!current || !supportsSqlImport) return null;
    return <ImportData
        key={sourceKey}
        open={openSource === sourceKey}
        onOpenChange={next => { setOpenSource(next ? sourceKey : null); }}
        databaseType={current.Type}
        onImportSuccess={() => { void client.refetchQueries({include: [GetStorageUnitsDocument]}); }}
    />;
};
