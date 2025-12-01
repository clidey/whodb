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

package output

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/clidey/whodb/cli/pkg/styles"
	"golang.org/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Spinner struct {
	message string
	out     io.Writer
	done    chan struct{}
	wg      sync.WaitGroup
	active  bool
	mu      sync.Mutex
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		out:     os.Stderr,
	}
}

func (s *Spinner) isTTY() bool {
	f, ok := s.out.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func (s *Spinner) Start() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}

	if !s.isTTY() {
		s.mu.Unlock()
		return
	}

	s.active = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go s.spin()
}

func (s *Spinner) spin() {
	defer s.wg.Done()

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	frame := 0
	for {
		select {
		case <-s.done:
			s.clear()
			return
		case <-ticker.C:
			s.render(frame)
			frame = (frame + 1) % len(spinnerFrames)
		}
	}
}

func (s *Spinner) render(frame int) {
	spinner := spinnerFrames[frame]
	if styles.ColorEnabled() {
		spinner = "\033[36m" + spinner + "\033[0m"
	}
	fmt.Fprintf(s.out, "\r%s %s", spinner, s.message)
}

func (s *Spinner) clear() {
	// Clear the line
	fmt.Fprintf(s.out, "\r\033[K")
}

func (s *Spinner) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	close(s.done)
	s.mu.Unlock()

	s.wg.Wait()
}

func (s *Spinner) StopWithSuccess(message string) {
	if s == nil {
		return
	}
	s.Stop()
	if s.isTTY() {
		prefix := "✓ "
		if styles.ColorEnabled() {
			prefix = "\033[32m✓\033[0m "
		}
		fmt.Fprintln(s.out, prefix+message)
	}
}

func (s *Spinner) StopWithError(message string) {
	if s == nil {
		return
	}
	s.Stop()
	if s.isTTY() {
		prefix := "✗ "
		if styles.ColorEnabled() {
			prefix = "\033[31m✗\033[0m "
		}
		fmt.Fprintln(s.out, prefix+message)
	}
}
