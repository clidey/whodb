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

import { FC, ReactElement, ReactNode } from "react";
import { cn } from "@clidey/ux";
import { CloudIcon } from "../heroicons";
import { useTranslation } from "@/hooks/use-translation";

interface DatabaseIconWithBadgeProps {
    /** The database icon element */
    icon: ReactElement | null;
    /** Whether to show the cloud badge */
    showCloudBadge?: boolean;
    /** Additional class name for the container */
    className?: string;
    /** Size variant */
    size?: "sm" | "md" | "lg";
}

/**
 * Wraps a database icon and optionally displays a small cloud badge
 * to indicate the connection is from a cloud provider (AWS).
 */
export const DatabaseIconWithBadge: FC<DatabaseIconWithBadgeProps> = ({
    icon,
    showCloudBadge = false,
    className,
    size = "md",
}) => {
    const { t } = useTranslation("components/aws-providers-section");

    const sizeClasses = {
        sm: "w-4 h-4",
        md: "w-6 h-6",
        lg: "w-8 h-8",
    };

    const badgeSizeClasses = {
        sm: "w-2 h-2 -right-0.5 -bottom-0.5",
        md: "w-3 h-3 -right-1 -bottom-1",
        lg: "w-4 h-4 -right-1 -bottom-1",
    };

    if (!icon) return null;

    return (
        <div className={cn("relative inline-flex", sizeClasses[size], className)}>
            {icon}
            {showCloudBadge && (
                <div
                    className={cn(
                        "absolute rounded-full bg-background border border-border flex items-center justify-center",
                        badgeSizeClasses[size]
                    )}
                    title={t('cloudConnection')}
                >
                    <CloudIcon className={cn(
                        "text-brand",
                        size === "sm" ? "w-1.5 h-1.5" : size === "md" ? "w-2 h-2" : "w-2.5 h-2.5"
                    )} />
                </div>
            )}
        </div>
    );
};

/**
 * Helper to determine if a profile ID indicates an AWS connection.
 * AWS connections have IDs prefixed with "aws-".
 */
export function isAwsConnection(profileId: string | undefined): boolean {
    return profileId?.startsWith("aws-") ?? false;
}
