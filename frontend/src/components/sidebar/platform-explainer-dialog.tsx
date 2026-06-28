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
import { useCallback, useState } from "react";
import logoImage from "../../../public/images/logo.svg";
import { useTranslation } from "@/hooks/use-translation";
import type { LocalLoginProfile } from "@/store/auth";
import { PLATFORM_URL } from "@/utils/platform-funnel";
import { ArrowTopRightOnSquareIcon, ChevronRightIcon, CircleStackIcon, RectangleGroupIcon, Squares2X2Icon } from "../heroicons";
import { PlatformImportSheet } from "./platform-import-sheet";

/** Props for the platform explainer dialog. */
export type PlatformExplainerDialogProps = {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    /** The currently selected connection, when one exists. Drives the primary action. */
    profile: LocalLoginProfile | undefined;
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
export const PlatformExplainerDialog: FC<PlatformExplainerDialogProps> = ({ open, onOpenChange, profile }) => {
    const { t } = useTranslation("components/sidebar");
    const [activeSlide, setActiveSlide] = useState(0);
    const [showImport, setShowImport] = useState(false);

    const slides: Slide[] = [
        { image: logoImage, title: t("platformSlideManagedTitle"), caption: t("platformSlideManagedCaption") },
        { image: logoImage, title: t("platformSlideDatasetsTitle"), caption: t("platformSlideDatasetsCaption") },
        { image: logoImage, title: t("platformSlideAgentsTitle"), caption: t("platformSlideAgentsCaption") },
    ];

    const goTo = useCallback((index: number) => {
        setActiveSlide((index + slides.length) % slides.length);
    }, [slides.length]);

    const handleOpenPlatform = () => {
        window.open(PLATFORM_URL, "_blank", "noopener,noreferrer");
        onOpenChange(false);
    };

    // When a connection is selected, the primary action carries it across:
    // the import sheet shows the non-secret preview before opening the platform.
    const handleBringConnection = () => {
        setShowImport(true);
    };

    const slide = slides[activeSlide];

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle className="text-xl">{t("platformExplainerTitle")}</DialogTitle>
                    <DialogDescription>{t("platformExplainerIntro")}</DialogDescription>
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
                    {profile ? (
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
                            <Button variant="ghost" onClick={() => { onOpenChange(false); }}>
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
                profile={profile}
            />
        </Dialog>
    );
};
