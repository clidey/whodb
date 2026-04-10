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

package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Tunnel manages an SSH tunnel that forwards local connections to a remote host
// through an SSH server. It opens a local TCP listener on a random port and
// forwards each accepted connection to the specified remote host:port via the
// SSH connection.
type Tunnel struct {
	sshClient  *ssh.Client
	listener   net.Listener
	remoteHost string
	remotePort int
	done       chan struct{}
	closeOnce  sync.Once
}

// NewTunnel creates a new SSH tunnel configuration. It connects to the SSH server
// at sshHost:sshPort using the provided credentials. Authentication methods are
// tried in order: key file (if provided), SSH agent (if SSH_AUTH_SOCK is set),
// then password (if provided).
//
// The tunnel will forward local connections to remoteHost:remotePort through the
// SSH server. Call Start() to begin accepting connections.
func NewTunnel(sshHost string, sshPort int, sshUser, sshKeyFile, sshPassword string, remoteHost string, remotePort int) (*Tunnel, error) {
	authMethods, err := buildAuthMethods(sshKeyFile, sshPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to build SSH auth methods: %w", err)
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH authentication method available: provide a key file, password, or ensure ssh-agent is running")
	}

	sshConfig := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshAddr := fmt.Sprintf("%s:%d", sshHost, sshPort)
	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server %s: %w", sshAddr, err)
	}

	return &Tunnel{
		sshClient:  client,
		remoteHost: remoteHost,
		remotePort: remotePort,
		done:       make(chan struct{}),
	}, nil
}

// Start opens a local TCP listener on a random port and begins forwarding
// connections to the remote host through the SSH tunnel. It returns after
// the listener is ready to accept connections.
func (t *Tunnel) Start() error {
	var err error
	t.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to open local listener: %w", err)
	}

	go t.acceptLoop()
	return nil
}

// Stop closes the local listener and the SSH connection. It is safe to call
// multiple times.
func (t *Tunnel) Stop() {
	t.closeOnce.Do(func() {
		close(t.done)
		if t.listener != nil {
			t.listener.Close()
		}
		if t.sshClient != nil {
			t.sshClient.Close()
		}
	})
}

// LocalPort returns the local port the tunnel listener is bound to.
// Returns 0 if the tunnel has not been started.
func (t *Tunnel) LocalPort() int {
	if t.listener == nil {
		return 0
	}
	return t.listener.Addr().(*net.TCPAddr).Port
}

func (t *Tunnel) acceptLoop() {
	remoteAddr := fmt.Sprintf("%s:%d", t.remoteHost, t.remotePort)

	for {
		localConn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.done:
				return
			default:
				continue
			}
		}
		go t.forward(localConn, remoteAddr)
	}
}

func (t *Tunnel) forward(localConn net.Conn, remoteAddr string) {
	remoteConn, err := t.sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		localConn.Close()
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(remoteConn, localConn)
		remoteConn.Close()
	}()

	go func() {
		defer wg.Done()
		io.Copy(localConn, remoteConn)
		localConn.Close()
	}()

	wg.Wait()
}

// buildAuthMethods constructs SSH authentication methods from the provided
// credentials. It tries key file first, then SSH agent, then password.
func buildAuthMethods(keyFile, password string) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// Key file authentication
	if keyFile != "" {
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key file %q: %w", keyFile, err)
		}

		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH key file %q: %w", keyFile, err)
		}

		methods = append(methods, ssh.PublicKeys(signer))
	}

	// SSH agent authentication
	if authSock := os.Getenv("SSH_AUTH_SOCK"); authSock != "" {
		conn, err := net.Dial("unix", authSock)
		if err == nil {
			agentClient := agent.NewClient(conn)
			methods = append(methods, ssh.PublicKeysCallback(agentClient.Signers))
		}
	}

	// Password authentication
	if password != "" {
		methods = append(methods, ssh.Password(password))
	}

	return methods, nil
}
