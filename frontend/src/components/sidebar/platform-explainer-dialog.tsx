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
    cn,
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@clidey/ux";
import type { FC, ReactNode } from "react";
import { useCallback, useEffect, useState } from "react";
import logoImage from "../../../public/images/logo.svg";
import { ANALYTICS_EVENTS } from "@/config/analytics-events";
import { trackPlatformFunnel } from "@/config/frontend-analytics";
import { useTranslation } from "@/hooks/use-translation";
import { useAppSelector } from "@/store/hooks";
import { buildPlatformUrl, type PlatformFunnelTrigger } from "@/utils/platform-funnel";
import { ArrowTopRightOnSquareIcon, ChevronRightIcon, CircleStackIcon, RectangleGroupIcon, Squares2X2Icon } from "../heroicons";
import { PlatformImportSheet } from "./platform-import-sheet";

/** Props for the platform explainer dialog. */
export type PlatformExplainerDialogProps = {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    /** The CE surface that opened this dialog; carried into analytics and outbound URLs. */
    trigger: PlatformFunnelTrigger;
    /** When set, the intro highlights that this source type is available on the platform. */
    sourceType?: string;
    /** Catalog id for the source type; when set, outbound links deep-link to creating it. */
    sourceTypeId?: string;
};

/** A single step card in the three-step funnel flow. */
const FlowStep: FC<{ icon: ReactNode; title: string; caption: string }> = ({ icon, title, caption }) => (
    <div className="flex flex-col items-center text-center gap-1 w-28">
        <div className="flex items-center justify-center w-14 h-14 rounded-lg border border-neutral-200 dark:border-neutral-800 text-neutral-500">
            {icon}
        </div>
        <span className="text-sm font-medium">{title}</span>
        <span className="text-xs text-neutral-500">{caption}</span>
    </div>
);

/**
 * A single slide: an image placeholder above a title and supporting line.
 * Images are swapped in later; for now every slide uses the same placeholder.
 */
type Slide = { image: string; title: string; caption: string };

/**
 * Static explainer describing how WhoDB Platform builds on the connections a
 * CE user already has. Opened from the sidebar footer; one click deep and
 * dismissible. Not a banner and not shown before any connection exists.
 */
export const PlatformExplainerDialog: FC<PlatformExplainerDialogProps> = ({ open, onOpenChange, trigger, sourceType, sourceTypeId }) => {
    const { t } = useTranslation("components/sidebar");
    const hasProfiles = useAppSelector(state => state.auth.profiles.length > 0);
    const [activeSlide, setActiveSlide] = useState(0);
    const [showImport, setShowImport] = useState(false);

    useEffect(() => {
        if (open) {
            trackPlatformFunnel(ANALYTICS_EVENTS.PLATFORM_FUNNEL_OPENED, trigger, {
                ...(sourceType ? { database_type: sourceType } : {}),
            });
        }
    }, [open, sourceType, trigger]);

    const handleDialogOpenChange = useCallback((nextOpen: boolean) => {
        if (!nextOpen) {
            trackPlatformFunnel(ANALYTICS_EVENTS.PLATFORM_FUNNEL_DISMISSED, trigger, {
                ...(sourceType ? { database_type: sourceType } : {}),
            });
        }
        onOpenChange(nextOpen);
    }, [onOpenChange, sourceType, trigger]);

    const slides: Slide[] = [
        { image: logoImage, title: t("platformSlideManagedTitle"), caption: t("platformSlideManagedCaption") },
        { image: logoImage, title: t("platformSlideDatasetsTitle"), caption: t("platformSlideDatasetsCaption") },
        { image: logoImage, title: t("platformSlideAgentsTitle"), caption: t("platformSlideAgentsCaption") },
    ];

    const goTo = useCallback((index: number) => {
        setActiveSlide((index + slides.length) % slides.length);
    }, [slides.length]);

    // With a source type in context, land on the hosted "create this source"
    // flow instead of the generic platform home.
    const handleOpenPlatform = () => {
        trackPlatformFunnel(ANALYTICS_EVENTS.PLATFORM_LINK_OPENED, trigger, {
            ...(sourceTypeId ? { database_type: sourceTypeId } : {}),
        });
        const url = sourceTypeId
            ? buildPlatformUrl(trigger, "/import", { type: sourceTypeId })
            : buildPlatformUrl(trigger);
        window.open(url, "_blank", "noopener,noreferrer");
        onOpenChange(false);
    };

    // When saved connections exist, the primary action opens the import sheet,
    // which lets the user pick which connections to carry to the platform.
    const handleBringConnection = () => {
        setShowImport(true);
    };

    const slide = slides[activeSlide];

    return (
        <Dialog open={open} onOpenChange={handleDialogOpenChange}>
            <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle className="text-xl">{t("platformExplainerTitle")}</DialogTitle>
                    <DialogDescription>
                        {sourceType
                            ? t("platformExplainerSourceIntro", { sourceType })
                            : t("platformExplainerIntro")}
                    </DialogDescription>
                </DialogHeader>

                <div className="flex flex-col gap-3">
                    <div className="relative rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-neutral-900 overflow-hidden">
                        <div className="flex items-center justify-center h-40">
                            <img
                                src={slide.image}
                                alt={slide.title}
                                className="max-h-24 max-w-full w-auto object-contain opacity-90"
                            />
                        </div>
                        <div className="flex flex-col gap-1 px-6 pb-4 text-center">
                            <span className="text-sm font-medium">{slide.title}</span>
                            <span className="text-xs text-neutral-500">{slide.caption}</span>
                        </div>
                        <button
                            type="button"
                            aria-label={t("platformCarouselPrev")}
                            onClick={() => { goTo(activeSlide - 1); }}
                            className="absolute left-3 top-1/2 -translate-y-1/2 flex items-center justify-center w-8 h-8 rounded-full bg-white/80 dark:bg-neutral-800/80 hover:bg-white dark:hover:bg-neutral-700 transition-colors"
                        >
                            <ChevronRightIcon className="w-4 h-4 rotate-180" />
                        </button>
                        <button
                            type="button"
                            aria-label={t("platformCarouselNext")}
                            onClick={() => { goTo(activeSlide + 1); }}
                            className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center justify-center w-8 h-8 rounded-full bg-white/80 dark:bg-neutral-800/80 hover:bg-white dark:hover:bg-neutral-700 transition-colors"
                        >
                            <ChevronRightIcon className="w-4 h-4" />
                        </button>
                    </div>
                    <div className="flex items-center justify-center gap-2">
                        {slides.map((slideItem, index) => (
                            <button
                                key={slideItem.title}
                                type="button"
                                aria-label={t("platformCarouselGoTo", { index: index + 1 })}
                                onClick={() => { goTo(index); }}
                                className={cn(
                                    "w-2 h-2 rounded-full transition-colors",
                                    index === activeSlide ? "bg-neutral-700 dark:bg-neutral-200" : "bg-neutral-300 dark:bg-neutral-700"
                                )}
                            />
                        ))}
                    </div>
                </div>

                <div className="flex items-center justify-center gap-3 py-2">
                    <FlowStep
                        icon={<CircleStackIcon className="w-6 h-6" />}
                        title={t("platformStep1Title")}
                        caption={t("platformStep1Caption")}
                    />
                    <ChevronRightIcon className="w-4 h-4 text-neutral-600 shrink-0" />
                    <FlowStep
                        icon={<RectangleGroupIcon className="w-6 h-6" />}
                        title={t("platformStep2Title")}
                        caption={t("platformStep2Caption")}
                    />
                    <ChevronRightIcon className="w-4 h-4 text-neutral-600 shrink-0" />
                    <FlowStep
                        icon={<Squares2X2Icon className="w-6 h-6" />}
                        title={t("platformStep3Title")}
                        caption={t("platformStep3Caption")}
                    />
                </div>

                <ul className="flex flex-col gap-2 text-sm text-neutral-600 dark:text-neutral-400">
                    <li>{t("platformBenefitCredentials")}</li>
                    <li>{t("platformBenefitDatasets")}</li>
                    <li>{t("platformBenefitAgents")}</li>
                </ul>

                <DialogFooter className="flex flex-row justify-end gap-2">
                    {sourceType ? (
                        // Source-specific entry: the user wants this source on the
                        // platform, so the deep link is the primary action.
                        <>
                            <Button variant="ghost" onClick={() => { handleDialogOpenChange(false); }}>
                                {t("platformMaybeLater")}
                            </Button>
                            <Button onClick={handleOpenPlatform} className="flex items-center gap-2">
                                {t("platformConnectSource", { sourceType })}
                                <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                            </Button>
                        </>
                    ) : hasProfiles ? (
                        <>
                            <Button variant="ghost" onClick={handleOpenPlatform} className="flex items-center gap-2">
                                {t("platformLearnMore")}
                                <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                            </Button>
                            <Button onClick={handleBringConnection} className="flex items-center gap-2">
                                {t("platformImportConnection")}
                            </Button>
                        </>
                    ) : (
                        <>
                            <Button variant="ghost" onClick={() => { handleDialogOpenChange(false); }}>
                                {t("platformMaybeLater")}
                            </Button>
                            <Button onClick={handleOpenPlatform} className="flex items-center gap-2">
                                {t("platformLearnMore")}
                                <ArrowTopRightOnSquareIcon className="w-4 h-4" />
                            </Button>
                        </>
                    )}
                </DialogFooter>
            </DialogContent>

            <PlatformImportSheet
                open={showImport}
                onOpenChange={setShowImport}
                trigger={trigger}
            />
        </Dialog>
    );
};
