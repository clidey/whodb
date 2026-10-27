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

import (
	"testing"
	"time"
)

func TestOperationBucketBoundaries(t *testing.T) {
	for _, tc := range []struct {
		duration time.Duration
		want     string
	}{
		{0, "lt_250ms"}, {249 * time.Millisecond, "lt_250ms"},
		{250 * time.Millisecond, "250ms_1s"}, {time.Second, "1s_3s"},
		{3 * time.Second, "3s_10s"}, {10 * time.Second, "gte_10s"},
	} {
		if got := DurationBucket(tc.duration); got != tc.want {
			t.Errorf("DurationBucket(%v) = %q; want %q", tc.duration, got, tc.want)
		}
	}
	for _, tc := range []struct {
		count int64
		want  string
	}{
		{0, "zero"}, {1, "one"}, {2, "2_5"}, {5, "2_5"}, {6, "6_20"},
		{20, "6_20"}, {21, "21_100"}, {100, "21_100"},
		{101, "101_1000"}, {1000, "101_1000"}, {1001, "gte_1000"},
	} {
		if got := CountBucket(tc.count); got != tc.want {
			t.Errorf("CountBucket(%d) = %q; want %q", tc.count, got, tc.want)
		}
	}
}
