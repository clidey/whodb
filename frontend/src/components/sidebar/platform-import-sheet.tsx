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

import {
    Button,
    Checkbox,
    Label,
    Sheet,
    SheetContent,
    SheetDescription,
    SheetFooter,
    SheetHeader,
    SheetTitle,
} from "@clidey/ux";
import type { FC } from "react";
import { useMemo, useState } from "react";
import { useTranslation } from "@/hooks/use-translation";
import { useAppSelector } from "@/store/hooks";
import { buildImportConnection, buildPlatformImportLandingUrl, PLATFORM_URL, postConnectionsToPlatform } from "@/utils/platform-funnel";
import { ph } from "@/utils/privacy";
import { ArrowTopRightOnSquareIcon } from "../heroicons";

/** Props for the platform import sheet. */
export type PlatformImportSheetProps = {
    open: boolean;
    onOpenChange: (open: boolean) => void;
};

/**
 * Sheet that lets a CE user carry their saved connections to WhoDB Platform.
 * Connections are read from the local auth store, multi-selected, optionally
 * sent with credentials, then staged on the platform via a POST that returns a
 * short-lived token. The hosted import page is opened with that token.
 */
export const PlatformImportSheet: FC<PlatformImportSheetProps> = ({ open, onOpenChange }) => {
    const { t } = useTranslation("components/sidebar");
    const profiles = useAppSelector(state => state.auth.profiles);
    const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set(profiles.map(profile => profile.Id)));
    const [sendCredentials, setSendCredentials] = useState(false);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState(false);

    const platformHost = new URL(PLATFORM_URL).host;
    const allSelected = profiles.length > 0 && selectedIds.size === profiles.length;

    const selectedProfiles = useMemo(
        () => profiles.filter(profile => selectedIds.has(profile.Id)),
        [profiles, selectedIds],
    );

    const toggleProfile = (id: string) => {
        setSelectedIds(previous => {
            const next = new Set(previous);
            if (next.has(id)) {
                next.delete(id);
            } else {
                next.add(id);
            }
            return next;
        });
    };

    const toggleAll = () => {
        setSelectedIds(allSelected ? new Set() : new Set(profiles.map(profile => profile.Id)));
    };

    const handleImport = async () => {
        setSubmitting(true);
        setError(false);
        try {
            const connections = selectedProfiles.map(profile => buildImportConnection(profile, sendCredentials));
            const { token } = await postConnectionsToPlatform(connections);
            window.open(buildPlatformImportLandingUrl(token), "_blank", "noopener,noreferrer");
            onOpenChange(false);
        } catch {
            setError(true);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent side="right" className="p-8 flex flex-col gap-6">
                <SheetHeader>
                    <SheetTitle>{t("platformImportTitle")}</SheetTitle>
                    <SheetDescription>{t("platformImportDescription")}</SheetDescription>
                </SheetHeader>

                <div className="flex flex-col gap-3 overflow-y-auto">
                    <div className="flex items-center gap-2">
                        <Checkbox
                            checked={allSelected}
                            onCheckedChange={toggleAll}
                            data-testid="platform-import-select-all"
                        />
                        <Label className="cursor-pointer" onClick={toggleAll}>{t("platformImportSelectAll")}</Label>
                    </div>
                    <div className="flex flex-col gap-1 rounded-lg border border-neutral-200 dark:border-neutral-800 p-2">
                        {profiles.map(profile => (
                            <label
                                key={profile.Id}
                                className="flex items-center gap-3 rounded-md px-2 py-2 cursor-pointer hover:bg-neutral-50 dark:hover:bg-neutral-900"
                            >
                                <Checkbox
                                    checked={selectedIds.has(profile.Id)}
                                    onCheckedChange={() => { toggleProfile(profile.Id); }}
                                />
                                <div className="flex flex-col min-w-0">
                                    <span className={`text-sm font-medium truncate ${ph.mask}`}>{profile.DisplayName ?? profile.Id}</span>
                                    <span className="text-xs text-neutral-500 truncate">
                                        {profile.Type}{profile.Hostname ? ` · ` : ""}<span className={ph.mask}>{profile.Hostname}</span>
                                    </span>
                                </div>
                            </label>
                        ))}
                    </div>
                </div>

                <div className="flex flex-col gap-1">
                    <div className="flex items-start gap-sm">
                        <Checkbox
                            checked={sendCredentials}
                            onCheckedChange={(value) => { setSendCredentials(Boolean(value)); }}
                            data-testid="platform-import-send-credentials"
                        />
                        <Label className="cursor-pointer" onClick={() => { setSendCredentials(value => !value); }}>
                            {t("platformImportSendCredentials")}
                        </Label>
                    </div>
                    <p className="text-xs text-neutral-500">
                        {sendCredentials ? t("platformImportCredentialsSent") : t("platformImportCredentialsReentry")}
                    </p>
                </div>

                <p className="text-xs text-neutral-500">{t("platformDestinationNote", { host: platformHost })}</p>

                {error && <p className="text-xs text-red-500">{t("platformImportError")}</p>}

                <SheetFooter className="mt-auto flex flex-row justify-end gap-2">
                    <Button variant="ghost" onClick={() => { onOpenChange(false); }}>
                        {t("platformCancel")}
                    </Button>
                    <Button
                        onClick={() => { void handleImport(); }}
                        disabled={selectedProfiles.length === 0 || submitting}
                        className="flex items-center gap-2"
                    >
                        {submitting ? t("platformImporting") : t("platformOpenInPlatform")}
                        <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                    </Button>
                </SheetFooter>
            </SheetContent>
        </Sheet>
    );
};
