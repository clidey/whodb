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
                    href={"https://whodb.clidey.com/privacy-policy"} target={"_blank"}
                    rel="noreferrer" className={"underline text-blue-500"}>Privacy Policy.</a>
                    <br/>
                    <br/>
                    WhoDB uses <a href={"https://www.highlight.io/"} target={"_blank"} rel="noreferrer"
                                  className={"underline text-blue-500"}>Highlight.io</a> to collect and manage this
                    data. It is an open source tool and all of its source code can be found on GitHub.
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