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
    Sheet,
    SheetContent,
    SheetDescription,
    SheetFooter,
    SheetHeader,
    SheetTitle,
} from "@clidey/ux";
import type { FC } from "react";
import { useTranslation } from "@/hooks/use-translation";
import type { LocalLoginProfile } from "@/store/auth";
import { buildPlatformImportPrefill, buildPlatformImportUrl, PLATFORM_URL } from "@/utils/platform-funnel";
import { ph } from "@/utils/privacy";
import { ArrowTopRightOnSquareIcon } from "../heroicons";

/** Props for the platform import sheet. */
export type PlatformImportSheetProps = {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    profile: LocalLoginProfile | undefined;
};

/** A single read-only field row in the connection preview. */
const PreviewRow: FC<{ label: string; value: string; mask?: boolean }> = ({ label, value, mask }) => (
    <div className="flex items-center justify-between gap-4 py-1.5">
        <span className="text-xs text-neutral-500">{label}</span>
        <span className={`text-xs font-medium truncate ${mask ? ph.mask : ""}`}>{value || "—"}</span>
    </div>
);

/**
 * Sheet that lets a CE user carry a saved connection to WhoDB Platform as a
 * managed source. Shows the non-secret fields that travel and opens the hosted
 * import deep-link in a new tab. The password is never sent in the URL.
 */
export const PlatformImportSheet: FC<PlatformImportSheetProps> = ({ open, onOpenChange, profile }) => {
    const { t } = useTranslation("components/sidebar");

    if (!profile) return null;

    const prefill = buildPlatformImportPrefill(profile);
    const platformHost = new URL(PLATFORM_URL).host;

    const handleOpen = () => {
        window.open(buildPlatformImportUrl(profile), "_blank", "noopener,noreferrer");
        onOpenChange(false);
    };

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent side="right" className="p-8 flex flex-col gap-6">
                <SheetHeader>
                    <SheetTitle>{t("platformImportTitle")}</SheetTitle>
                    <SheetDescription>{t("platformImportDescription")}</SheetDescription>
                </SheetHeader>

                <div className="rounded-lg border border-neutral-200 dark:border-neutral-800 p-4">
                    <PreviewRow label={t("platformFieldName")} value={prefill.displayName} mask />
                    <PreviewRow label={t("platformFieldType")} value={prefill.type} />
                    <PreviewRow label={t("platformFieldHostname")} value={prefill.hostname} mask />
                    {prefill.port && <PreviewRow label={t("platformFieldPort")} value={prefill.port} />}
                    <PreviewRow label={t("platformFieldDatabase")} value={prefill.database} mask />
                    <PreviewRow label={t("platformFieldUsername")} value={prefill.username} mask />
                    <PreviewRow label={t("platformFieldPassword")} value={t("platformPasswordReentry")} />
                </div>

                <p className="text-xs text-neutral-500">{t("platformDestinationNote", { host: platformHost })}</p>

                <SheetFooter className="mt-auto flex flex-row justify-end gap-2">
                    <Button variant="ghost" onClick={() => { onOpenChange(false); }}>
                        {t("platformCancel")}
                    </Button>
                    <Button onClick={handleOpen} className="flex items-center gap-2">
                        {t("platformOpenInPlatform")}
                        <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                    </Button>
                </SheetFooter>
            </SheetContent>
        </Sheet>
    );
};
