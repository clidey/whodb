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

const options = {
    api_host: "https://us.i.posthog.com",
}

const posthogKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH"

// Only initialize PostHog in Community Edition
const isEE = import.meta.env.VITE_BUILD_EDITION === 'ee';

let posthogPromise: Promise<typeof import("posthog-js")> | null = null;

const getPosthog = () => {
    if (!posthogPromise) {
        posthogPromise = import("posthog-js");
    }
    return posthogPromise;
};


export const initPosthog = async () => {
    if (!isEE) {
        const posthog = (await getPosthog()).default;
        posthog.init(posthogKey, options);
        return posthog;
    }
    return null;
};

export const optOutUser = async () => {
    if (!isEE) {
        const posthog = (await getPosthog()).default;
        posthog.opt_out_capturing();
    }
};

export const optInUser = async () => {
    if (!isEE) {
        const posthog = (await getPosthog()).default;
        posthog.opt_in_capturing();
    }
};