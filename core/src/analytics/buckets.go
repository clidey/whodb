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

package analytics

import "time"

// DurationBucket returns the shared low-cardinality operation latency bucket.
func DurationBucket(duration time.Duration) string {
	switch {
	case duration < 250*time.Millisecond:
		return "lt_250ms"
	case duration < time.Second:
		return "250ms_1s"
	case duration < 3*time.Second:
		return "1s_3s"
	case duration < 10*time.Second:
		return "3s_10s"
	default:
		return "gte_10s"
	}
}

// CountBucket returns the shared low-cardinality result or workload count bucket.
func CountBucket(count int64) string {
	switch {
	case count <= 0:
		return "zero"
	case count == 1:
		return "one"
	case count <= 5:
		return "2_5"
	case count <= 20:
		return "6_20"
	case count <= 100:
		return "21_100"
	case count <= 1000:
		return "101_1000"
	default:
		return "gte_1000"
	}
}
