/*
 * Copyright 2026 Clidey, Inc.
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

package com.clidey.whodb;

import java.time.Duration;

/**
 * Bounds for run-polling methods ({@code wait}).
 *
 * @param timeout caps the wait (default 60s)
 * @param poll delay between reads (default 2s); zero polls without delay
 */
public record WaitOptions(Duration timeout, Duration poll) {
    /** The default 60s timeout with a 2s poll. */
    public static WaitOptions defaults() {
        return new WaitOptions(Duration.ofSeconds(60), Duration.ofSeconds(2));
    }
}
