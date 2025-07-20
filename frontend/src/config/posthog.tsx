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

import posthog from "posthog-js";

const options = {
    api_host: "https://us.i.posthog.com",
}

const posthogKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH"

// Only initialize PostHog in Community Edition
const isEE = import.meta.env.VITE_BUILD_EDITION === 'ee';

export const initPosthog = () => {
    if (!isEE) {
        posthog.init(posthogKey, options);
        return posthog;
    }
    // Return a dummy client for EE that does nothing
    return null;
}

export const optOutUser = () => {
    if (!isEE) {
        posthog.opt_out_capturing()
    }
}

export const optInUser = () => {
    if (!isEE) {
        posthog.opt_in_capturing()
    }
}