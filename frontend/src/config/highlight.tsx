// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
        // enableOtelTracing: true,
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