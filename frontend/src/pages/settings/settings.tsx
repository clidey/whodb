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
import { Label, Switch, Separator } from "@clidey/ux";

export const SettingsPage: FC = () => {
    const dispatch = useAppDispatch();
    const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled);

    const handleMetricsToggle = useCallback((enabled: boolean) => {
        dispatch(SettingsActions.setMetricsEnabled(enabled));
    }, [dispatch]);

    return (
        <InternalPage routes={[InternalRoutes.Settings!]}>
            <div className="flex flex-col items-center w-full max-w-2xl mx-auto py-10 gap-8">
                <div className="w-full flex flex-col gap-0">
                    <div className="flex flex-col gap-2 mb-4">
                        <p className="text-2xl font-bold flex items-center gap-2">
                            Telemetry and Performance Metrics
                        </p>
                    </div>
                    <Separator />
                    <div className="flex flex-col gap-6 py-6">
                        {isEEMode ? (
                            <div className="flex flex-col gap-2 items-center">
                                <h3 className="text-base">
                                    WhoDB Enterprise does not collect telemetry and performance metrics.
                                </h3>
                            </div>
                        ) : (
                            <div className="flex flex-col gap-4">
                                <h3 className="text-base">
                                    We use this information solely to enhance the performance of WhoDB.
                                    For details on what data we collect, how it's collected, stored, and used, please refer to our <a
                                    href={"https://clidey.com/privacy-policy"} target={"_blank"}
                                    rel="noreferrer" className={"underline text-blue-500"}>Privacy Policy.</a>
                                    <br/>
                                    <br/>
                                    WhoDB uses <a href={"https://posthog.com/"} target={"_blank"} rel="noreferrer"
                                                className={"underline text-blue-500"}>Posthog</a> to collect and manage this
                                    data. More information about this tool can be found on its <a href={"https://github.com/PostHog/posthog"} target={"_blank"} rel="noreferrer" className={"underline text-blue-500"}>Github</a>.
                                    We have taken measures to redact as much sensitive information as we can and will continuously
                                    evaluate to make sure that it fits yours and our needs without sacrificing anything.
                                    <br/>
                                    <br/>
                                    If you know of a tool that might serve us better, we’d love to hear from you! Just reach out via the
                                    "Contact Us" option in the bottom left corner of the screen.
                                </h3>
                                <div className="flex gap-2">
                                    <Label>{metricsEnabled ? "Enabled" : "Disabled"}</Label>
                                    <Switch checked={metricsEnabled} onChange={() => handleMetricsToggle(!metricsEnabled)}/>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </InternalPage>
    );
}