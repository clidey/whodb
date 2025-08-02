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
import { isEEMode } from "@/config/ee-imports";
import { useUpdateSettingsMutation } from "./update-settings.generated";

export const SettingsPage: FC = () => {
    const dispatch = useAppDispatch();
    const metricsEnabled = useAppSelector(state => state.settings.metricsEnabled)
    const performanceMonitoringEnabled = useAppSelector(state => state.settings.performanceMonitoringEnabled)
    const performanceMetricsConfig = useAppSelector(state => state.settings.performanceMetricsConfig)
    const [updateSettings] = useUpdateSettingsMutation();

    const handleMetricsToggle = useCallback((enabled: boolean) => {
        dispatch(SettingsActions.setMetricsEnabled(enabled))
        updateSettings({
            variables: {
                newSettings: {
                    MetricsEnabled: enabled.toString(),
                },
            },
        });
    }, [dispatch, updateSettings]);

    const handlePerformanceMonitoringToggle = useCallback((enabled: boolean) => {
        dispatch(SettingsActions.setPerformanceMonitoringEnabled(enabled))
        updateSettings({
            variables: {
                newSettings: {
                    PerformanceMonitoringEnabled: enabled.toString(),
                },
            },
        });
    }, [dispatch, updateSettings]);

    const handleMetricConfigChange = useCallback((metric: string, enabled: boolean) => {
        const newConfig = { [metric]: enabled };
        dispatch(SettingsActions.setPerformanceMetricsConfig(newConfig))
        updateSettings({
            variables: {
                newSettings: {
                    PerformanceMetricsConfig: {
                        [metric]: enabled.toString(),
                    },
                },
            },
        });
    }, [dispatch, updateSettings]);

    return <InternalPage routes={[InternalRoutes.Settings!]}>
        <div className="flex justify-center items-center w-full">
            <div className="w-full max-w-[1000px] flex flex-col gap-4">
                <h2 className="text-2xl text-neutral-700 dark:text-neutral-300">Telemetry and Performance Metrics</h2>
                {isEEMode ? (
                    <h3 className="text-base text-neutral-700 dark:text-neutral-300">
                        WhoDB Enterprise does not collect telemetry and performance metrics.
                    </h3>
                ) : (
                    <>
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
                    </>
                )}
                <div className="mt-8">
                    <h2 className="text-2xl text-neutral-700 dark:text-neutral-300">Database Performance Monitoring</h2>
                    <h3 className="text-base text-neutral-700 dark:text-neutral-300 mt-2">
                        Monitor and analyze database performance metrics to optimize your queries and identify bottlenecks.
                    </h3>
                    <div className="flex gap-2 items-center mr-4 mt-4">
                        <Text label={performanceMonitoringEnabled ? "Enabled" : "Disabled"}/>
                        <ToggleInput value={performanceMonitoringEnabled} setValue={handlePerformanceMonitoringToggle}/>
                    </div>
                    {performanceMonitoringEnabled && (
                        <div className="mt-6">
                            <h3 className="text-lg text-neutral-700 dark:text-neutral-300 mb-4">Metrics to Collect</h3>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div className="flex justify-between items-center">
                                    <Text label="Query Latency" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.query_latency} 
                                        setValue={(v) => handleMetricConfigChange('query_latency', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="Query Count" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.query_count} 
                                        setValue={(v) => handleMetricConfigChange('query_count', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="Connection Count" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.connection_count} 
                                        setValue={(v) => handleMetricConfigChange('connection_count', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="Error Count" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.error_count} 
                                        setValue={(v) => handleMetricConfigChange('error_count', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="CPU Usage" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.cpu_usage} 
                                        setValue={(v) => handleMetricConfigChange('cpu_usage', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="Memory Usage" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.memory_usage} 
                                        setValue={(v) => handleMetricConfigChange('memory_usage', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="Disk I/O" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.disk_io} 
                                        setValue={(v) => handleMetricConfigChange('disk_io', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="Cache Hit Ratio" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.cache_hit_ratio} 
                                        setValue={(v) => handleMetricConfigChange('cache_hit_ratio', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="Transaction Count" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.transaction_count} 
                                        setValue={(v) => handleMetricConfigChange('transaction_count', v)}
                                    />
                                </div>
                                <div className="flex justify-between items-center">
                                    <Text label="Lock Wait Time" className="text-sm"/>
                                    <ToggleInput 
                                        value={performanceMetricsConfig.lock_wait_time} 
                                        setValue={(v) => handleMetricConfigChange('lock_wait_time', v)}
                                    />
                                </div>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>

    </InternalPage>
}