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

import com.fasterxml.jackson.databind.ObjectMapper;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.function.Predicate;
import java.util.function.Supplier;

/** Shared helpers for the non-ontology facades: polling, encoding, JSON shapes. */
final class Facades {
    private static final ObjectMapper JSON = new ObjectMapper();
    static final int DEFAULT_PAGE_SIZE = 100;

    private Facades() {}

    /** Statuses after which a function or transform run will not change. */
    static boolean isTerminalRunStatus(Object status) {
        if (status == null) {
            return false;
        }
        return switch (status.toString().trim().toLowerCase(java.util.Locale.ROOT)) {
            case "success", "succeeded", "completed", "complete", "failed", "failure",
                "cancelled", "canceled", "error" -> true;
            default -> false;
        };
    }

    /** Reads until {@code done} or the timeout elapses (PLATFORM error RUN_TIMEOUT). */
    static Map<String, Object> pollUntil(Supplier<Map<String, Object>> read, Predicate<Map<String, Object>> done,
                                         WaitOptions options, String describe) {
        WaitOptions effective = options == null ? WaitOptions.defaults() : options;
        long deadline = System.nanoTime() + effective.timeout().toNanos();
        while (true) {
            Map<String, Object> value = read.get();
            if (done.test(value)) {
                return value;
            }
            if (System.nanoTime() >= deadline) {
                throw new WhoDBException(WhoDBException.Kind.PLATFORM,
                    describe + " did not finish within " + effective.timeout().toMillis() + "ms", "RUN_TIMEOUT");
            }
            if (!effective.poll().isZero()) {
                try {
                    Thread.sleep(effective.poll().toMillis());
                } catch (InterruptedException interrupted) {
                    Thread.currentThread().interrupt();
                    throw new WhoDBException(WhoDBException.Kind.TRANSPORT, "interrupted while waiting for " + describe);
                }
            }
        }
    }

    /** Renders a function input as the compact JSON string the platform expects. */
    static String encodeInput(Object input) {
        if (input instanceof String text) {
            return text;
        }
        if (input == null) {
            return "{}";
        }
        try {
            return JSON.writeValueAsString(input);
        } catch (Exception error) {
            throw new WhoDBException(WhoDBException.Kind.VALIDATION, "failed to encode function input as JSON");
        }
    }

    /** Builds a variables map from alternating key/value arguments; null values are kept. */
    static Map<String, Object> vars(Object... keyValues) {
        Map<String, Object> map = new LinkedHashMap<>();
        for (int index = 0; index + 1 < keyValues.length; index += 2) {
            map.put(String.valueOf(keyValues[index]), keyValues[index + 1]);
        }
        return map;
    }

    @SuppressWarnings("unchecked")
    static Map<String, Object> asMap(Object result) {
        return result instanceof Map<?, ?> map ? (Map<String, Object>) map : null;
    }

    @SuppressWarnings("unchecked")
    static List<Map<String, Object>> asList(Object result) {
        List<Map<String, Object>> out = new ArrayList<>();
        if (result instanceof List<?> entries) {
            for (Object entry : entries) {
                if (entry instanceof Map<?, ?> map) {
                    out.add((Map<String, Object>) map);
                }
            }
        }
        return out;
    }

    /** Resolves a project resource addressed by id or name. */
    static Map<String, Object> findByIdOrName(List<Map<String, Object>> items, String needle) {
        for (Map<String, Object> item : items) {
            if (needle.equals(item.get("id")) || needle.equals(item.get("name"))) {
                return item;
            }
        }
        return null;
    }

    /** Copies definition fields and injects the given values. */
    static Map<String, Object> mergeInput(Map<String, Object> definition, Object... injected) {
        Map<String, Object> input = new LinkedHashMap<>();
        if (definition != null) {
            input.putAll(definition);
        }
        input.putAll(vars(injected));
        return input;
    }

    static String require(String value, String label) {
        if (value == null || value.trim().isEmpty()) {
            throw new WhoDBException(WhoDBException.Kind.VALIDATION, label + " is required");
        }
        return value;
    }

    /** Reads the total from either wire shape (lowercase "total" or PascalCase "TotalCount"). */
    static Object totalOf(Map<String, Object> result) {
        if (result == null) {
            return null;
        }
        return result.containsKey("columns") ? result.get("total") : result.get("TotalCount");
    }

    static String blankToNull(String value) {
        return value == null || value.isEmpty() ? null : value;
    }
}
