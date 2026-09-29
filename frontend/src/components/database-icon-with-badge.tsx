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

import type { FC, ReactElement } from "react";
import { cn, Tooltip, TooltipContent, TooltipTrigger } from "@clidey/ux";
import { ShieldCheckIcon } from "./heroicons";
import { useTranslation } from "@/hooks/use-translation";

interface DatabaseIconWithBadgeProps {
    /** The database icon element. */
    icon: ReactElement | null;
    /** SSL status for the connection. */
    sslStatus?: { IsEnabled: boolean; Mode: string } | null;
    /** Size variant. */
    size?: "sm" | "md" | "lg";
}

/** Wraps a database icon and displays its SSL status when enabled. */
export const DatabaseIconWithBadge: FC<DatabaseIconWithBadgeProps> = ({
    icon,
    sslStatus,
    size = "md",
}) => {
    const { t } = useTranslation("components/sidebar");
    if (!icon) return null;

    const sizeClasses = { sm: "w-4 h-4", md: "w-6 h-6", lg: "w-8 h-8" };
    const badgeSizeClasses = {
        sm: "w-2 h-2 -right-0.5 -top-0.5",
        md: "w-3 h-3 -right-1 -top-1",
        lg: "w-4 h-4 -right-1 -top-1",
    };

    return (
        <div className={cn("relative inline-flex", sizeClasses[size])}>
            {icon}
            {sslStatus?.IsEnabled && (
                <Tooltip>
                    <TooltipTrigger asChild>
                        <div
                            data-testid="ssl-badge"
                            className={cn(
                                "absolute rounded-full bg-background border border-border flex items-center justify-center",
                                badgeSizeClasses[size],
                            )}
                        >
                            <ShieldCheckIcon className={cn(
                                "text-green-500",
                                size === "sm" ? "w-1.5 h-1.5" : size === "md" ? "w-2 h-2" : "w-2.5 h-2.5",
                            )} />
                        </div>
                    </TooltipTrigger>
                    <TooltipContent>{t("sslSecured", { mode: sslStatus.Mode })}</TooltipContent>
                </Tooltip>
            )}
        </div>
    );
};
