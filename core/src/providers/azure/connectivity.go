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

package azure

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/providers"
)

const (
	connectivityTimeout     = 3 * time.Second
	maxConcurrentChecks     = 10
	connectivityReachable   = "reachable"
	connectivityUnreachable = "unreachable"
)

// CheckConnectivity performs TCP dial tests on discovered connections.
// Called during RefreshDiscovery (not cached DiscoverAll) to avoid slowing down page loads.
func CheckConnectivity(connections []providers.DiscoveredConnection) {
	log.Debugf("Azure Connectivity: checking %d connections", len(connections))
	sem := make(chan struct{}, maxConcurrentChecks)
	var wg sync.WaitGroup

	var reachable, unreachable atomic.Int32

	for i := range connections {
		conn := &connections[i]
		endpoint := conn.Metadata["endpoint"]
		port := conn.Metadata["port"]
		if endpoint == "" || port == "" {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			addr := net.JoinHostPort(endpoint, port)
			ctx, cancel := context.WithTimeout(context.Background(), connectivityTimeout)
			defer cancel()
			c, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
			if err != nil {
				log.Debugf("Azure Connectivity: %s unreachable: %v", addr, err)
				conn.Metadata["connectivity"] = connectivityUnreachable
				unreachable.Add(1)
				return
			}
			_ = c.Close()
			log.Debugf("Azure Connectivity: %s reachable", addr)
			conn.Metadata["connectivity"] = connectivityReachable
			reachable.Add(1)
		}()
	}

	wg.Wait()
	log.Debugf("Azure Connectivity: done — %d reachable, %d unreachable", reachable.Load(), unreachable.Load())
}
