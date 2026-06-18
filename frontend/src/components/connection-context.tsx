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

import { cn, Tooltip, TooltipContent, TooltipTrigger } from "@clidey/ux";
import type { FC } from "react";
import { cloneElement } from "react";
import { useLocation } from "react-router-dom";
import { useAppSelector } from "../store/hooks";
import { useSourceContract } from "../hooks/useSourceContract";
import { useSourceTypeItem } from "../hooks/useSourceCatalog";
import { ChevronRightIcon } from "./heroicons";

/**
 * Persistent connection context indicator shown in the page header.
 * Displays the current source type, hostname/database, schema, and
 * active table name regardless of sidebar open/closed state.
 */
export const ConnectionContext: FC = () => {
    const current = useAppSelector(state => state.auth.current);
    const schema = useAppSelector(state => state.database.schema);
    const { supportsSchema, supportsDatabaseSwitching } = useSourceContract(current?.Type);
    const { item } = useSourceTypeItem(current?.Type);
    const location = useLocation();

    if (!current) return null;

    const segments: string[] = [];

    if (current.Hostname) {
        segments.push(current.Hostname);
    }

    if (supportsDatabaseSwitching && current.Database) {
        segments.push(current.Database);
    }

    if (supportsSchema && schema) {
        segments.push(schema);
    }

    const locationState = location.state as { unit?: { Name?: string } } | undefined;
    if (locationState?.unit?.Name) {
        segments.push(locationState.unit.Name);
    }

    if (segments.length === 0) return null;

    const fullPath = segments.join(" / ");

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <div
                    className={cn(
                        "inline-flex items-center gap-1 px-2 py-1 rounded-md w-fit",
                        "text-xs text-muted-foreground",
                        "bg-muted/50 dark:bg-muted/30",
                        "border border-transparent hover:border-border/50",
                        "transition-colors duration-150"
                    )}
                    data-testid="connection-context"
                    aria-label={`Connected to ${fullPath}`}
                >
                    {item?.icon && cloneElement(item.icon, {
                        className: "w-3.5 h-3.5 shrink-0 text-primary",
                    })}
                    {segments.map((segment, i) => (
                        <span key={i} className="flex items-center gap-1 shrink-0">
                            {i > 0 && <ChevronRightIcon className="w-3 h-3 shrink-0 opacity-40" />}
                            <span>{segment}</span>
                        </span>
                    ))}
                </div>
            </TooltipTrigger>
            <TooltipContent side="bottom">
                <p className="text-xs">{fullPath}</p>
            </TooltipContent>
        </Tooltip>
    );
};
