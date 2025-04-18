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
import { Text, ToggleInput } from "../../components/input";
import { InternalPage } from "../../components/page";
import { InternalRoutes } from "../../config/routes";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { SettingsActions } from "../../store/settings";

export const SettingsPage: FC = () => {
    const dispatch = useAppDispatch();
    const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled)

    const handleMetricsToggle = useCallback((enabled: boolean) => {
        dispatch(SettingsActions.setMetricsEnabled(enabled))
    }, [dispatch]);

    return <InternalPage routes={[InternalRoutes.Settings]}>
        <div className="flex justify-center items-center w-full">
            <div className="w-full max-w-[1000px] flex flex-col gap-4">
                <h2 className="text-2xl text-neutral-700 dark:text-neutral-300">Telemetry and Performance Metrics</h2>
                <h3 className="text-base text-neutral-700 dark:text-neutral-300">
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
                <div className="flex gap-2 items-center mr-4">
                    <Text label={metricsEnabled ? "Enabled" : "Disabled"}/>
                    <ToggleInput value={metricsEnabled} setValue={handleMetricsToggle}/>
                </div>
            </div>
        </div>

    </InternalPage>
}