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

import { FC, useCallback } from "react";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { SettingsActions } from "../../store/settings";
import { isEEMode } from "@/config/ee-imports";
import { Label, Switch, Separator, DropdownMenu, DropdownMenuTrigger, Button, DropdownMenuContent, DropdownMenuItem, SelectTrigger, Select, SelectContent, SelectItem, SelectValue, Card, CardContent, CardHeader, Badge } from "@clidey/ux";
import { ExternalLink } from "../../utils/external-links";

export const SettingsPage: FC = () => {
    const dispatch = useAppDispatch();
    const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);
    const storageUnitView = useAppSelector(state => state.settings.storageUnitView);
    const fontSize = useAppSelector(state => state.settings.fontSize);
    const borderRadius = useAppSelector(state => state.settings.borderRadius);
    const spacing = useAppSelector(state => state.settings.spacing);
    const whereConditionMode = useAppSelector(state => state.settings.whereConditionMode);

    const handleMetricsToggle = useCallback((enabled: boolean) => {
        dispatch(SettingsActions.setMetricsEnabled(enabled));
    }, [dispatch]);

    const handleStorageUnitViewToggle = useCallback((view: 'list' | 'card') => {
        dispatch(SettingsActions.setStorageUnitView(view));
    }, [dispatch]);

    const handleFontSizeChange = useCallback((size: 'small' | 'medium' | 'large') => {
        dispatch(SettingsActions.setFontSize(size));
    }, [dispatch]);

    const handleBorderRadiusChange = useCallback((radius: 'none' | 'small' | 'medium' | 'large') => {
        dispatch(SettingsActions.setBorderRadius(radius));
    }, [dispatch]);

    const handleSpacingChange = useCallback((space: 'compact' | 'comfortable' | 'spacious') => {
        dispatch(SettingsActions.setSpacing(space));
    }, [dispatch]);

    const handleWhereConditionModeChange = useCallback((mode: 'popover' | 'sheet') => {
        dispatch(SettingsActions.setWhereConditionMode(mode));
    }, [dispatch]);


    return (
        <InternalPage routes={[InternalRoutes.Settings!]}>
            <div className="flex flex-col items-center w-full max-w-2xl mx-auto py-10 gap-8">
                <div className="w-full flex flex-col gap-0">
                    <div className="flex flex-col gap-2">
                        <p className="text-2xl font-bold flex items-center gap-2">
                            Telemetry and Performance Metrics
                        </p>
                    </div>
                    <div className="flex flex-col gap-xl py-6">
                        {isEEMode ? (
                            <div className="flex flex-col gap-sm items-center">
                                <h3 className="text-base">
                                    WhoDB Enterprise does not collect telemetry and performance metrics.
                                </h3>
                            </div>
                        ) : (
                            <div className="flex flex-col gap-4">
                                <h3 className="text-base">
                                    We use this information solely to enhance the performance of WhoDB.
                                    For details on what data we collect, how it's collected, stored, and used, please
                                    refer to our <ExternalLink
                                    href={"https://clidey.com/privacy-policy"}
                                    className={"underline text-blue-500"}>Privacy Policy.</ExternalLink>
                                    <br/>
                                    <br/>
                                    WhoDB uses <ExternalLink href={"https://posthog.com/"}
                                                  className={"underline text-blue-500"}>Posthog</ExternalLink> to collect and
                                    manage this
                                    data. More information about this tool can be found on its <ExternalLink
                                    href={"https://github.com/PostHog/posthog"}
                                    className={"underline text-blue-500"}>Github</ExternalLink>.
                                    We have taken measures to redact as much sensitive information as we can and will
                                    continuously
                                    evaluate to make sure that it fits yours and our needs without sacrificing anything.
                                    <br/>
                                    <br/>
                                    If you know of a tool that might serve us better, we’d love to hear from you! Just
                                    reach out via the
                                    "Contact Us" option in the bottom left corner of the screen.
                                </h3>
                                <div className="flex justify-between">
                                    <Label>{metricsEnabled ? "Enable Telemetry" : "Disable Telemetry"}</Label>
                                    <Switch checked={metricsEnabled} onCheckedChange={handleMetricsToggle}/>
                                </div>
                                <Separator className="mt-4" />
                            </div>
                        )}
                        <div className="flex flex-col gap-sm mb-2">
                            <p className="text-lg font-bold">
                                Personalize Your Experience
                            </p>
                            <p className="text-base">
                                Make WhoDB your own by customizing the appearance and feel of the WhoDB interface
                            </p>
                        </div>
                        <div className="flex justify-between">
                            <Label>Storage Unit Default View</Label>
                            <Select value={storageUnitView} onValueChange={handleStorageUnitViewToggle}>
                                <SelectTrigger id="storage-unit-view" className="w-[135px]">
                                    <SelectValue placeholder="Select a view" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="list">List</SelectItem>
                                    <SelectItem value="card">Card</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>Font Size</Label>
                            <Select value={fontSize} onValueChange={handleFontSizeChange}>
                                <SelectTrigger id="font-size" className="w-[135px]">
                                    <SelectValue placeholder="Select font size" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="small">Small</SelectItem>
                                    <SelectItem value="medium">Medium</SelectItem>
                                    <SelectItem value="large">Large</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>Border Radius</Label>
                            <Select value={borderRadius} onValueChange={handleBorderRadiusChange}>
                                <SelectTrigger id="border-radius" className="w-[135px]">
                                    <SelectValue placeholder="Select border radius" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="none">None</SelectItem>
                                    <SelectItem value="small">Small</SelectItem>
                                    <SelectItem value="medium">Medium</SelectItem>
                                    <SelectItem value="large">Large</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>Spacing</Label>
                            <Select value={spacing} onValueChange={handleSpacingChange}>
                                <SelectTrigger id="spacing" className="w-[135px]">
                                    <SelectValue placeholder="Select spacing" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="compact">Compact</SelectItem>
                                    <SelectItem value="comfortable">Comfortable</SelectItem>
                                    <SelectItem value="spacious">Spacious</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex justify-between">
                            <Label>Where Condition Mode</Label>
                            <Select value={whereConditionMode} onValueChange={handleWhereConditionModeChange}>
                                <SelectTrigger id="where-condition-mode" className="w-[135px]">
                                    <SelectValue placeholder="Select mode" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="popover">Popover</SelectItem>
                                    <SelectItem value="sheet">Sheet</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                    </div>
                </div>
            </div>
        </InternalPage>
    );
}