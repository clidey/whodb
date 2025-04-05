import { H } from 'highlight.run';

const highlightProjectId = "4d7z8oqe";

export const initHighlight = (env: 'development' | 'staging' | 'production') => {
    H.init(highlightProjectId, {
        serviceName: 'WhoDB-frontend',
        tracingOrigins: ["localhost", "host.docker.internal"],
        networkRecording: {
            enabled: true,
            // recordHeadersAndBody: true,
        },
        environment: env,
        privacySetting: "strict",
        enablePerformanceRecording: true,
        manualStart: true
    });
}

export const startHighlight = ()=> {
    H.start({
        silent: true
    });
}

export const stopHighlight = () => {
    H.stop({
        silent: true
    });
}