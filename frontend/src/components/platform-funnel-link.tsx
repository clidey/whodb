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

import { cn } from "@clidey/ux";
import type { FC } from "react";
import { useState } from "react";
import { featureFlags } from "@/config/features";
import type { PlatformFunnelTrigger } from "@/utils/platform-funnel";
import { PlatformExplainerDialog } from "./sidebar/platform-explainer-dialog";

/** Props for the inline platform funnel link. */
export type PlatformFunnelLinkProps = {
    /** The CE surface this link lives on; carried into analytics and URLs. */
    trigger: PlatformFunnelTrigger;
    /** Localized link text. */
    label: string;
    className?: string;
};

/**
 * A one-line text link that opens the WhoDB Platform explainer dialog.
 * Renders nothing unless the platform funnel feature flag is enabled, so
 * call sites can mount it unconditionally.
 */
export const PlatformFunnelLink: FC<PlatformFunnelLinkProps> = ({ trigger, label, className }) => {
    const [showExplainer, setShowExplainer] = useState(false);

    if (!featureFlags.platformFunnel) {
        return null;
    }

    return (
        <>
            <button
                type="button"
                onClick={() => { setShowExplainer(true); }}
                className={cn("text-xs text-left text-neutral-500 underline underline-offset-2 hover:text-neutral-700 dark:hover:text-neutral-300 transition-colors", className)}
                data-testid={`platform-funnel-link-${trigger}`}
            >
                {label}
            </button>
            <PlatformExplainerDialog
                open={showExplainer}
                onOpenChange={setShowExplainer}
                trigger={trigger}
            />
        </>
    );
};
