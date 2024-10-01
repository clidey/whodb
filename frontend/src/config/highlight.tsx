import { H } from 'highlight.run';

export const initHighlight = (env: 'development' | 'staging' | 'production') => {
    H.init('', {
        serviceName: 'WhoDB-frontend',
        tracingOrigins: ["localhost", "host.docker.internal"],
        networkRecording: {
            enabled: true,
            // recordHeadersAndBody: true,
        },
        environment: env,
        privacySetting: "strict",
        // enableOtelTracing: true,
        enablePerformanceRecording: true,
        manualStart: true
    })
}

export const startHighlight = ()=> {
    H.start({
        silent: true
    })
}

export const stopHighlight = () => {
    H.stop({
        silent: true
    })
}