package checkers

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type CMDChecker struct {
	sshKeyPath string
}

func NewCMDChecker() *CMDChecker {
	return &CMDChecker{
		sshKeyPath: os.Getenv("SSH_KEY_PATH"),
	}
}

func (c *CMDChecker) Protocol() Protocol {
	return CMD
}

func (c *CMDChecker) Check(ctx context.Context, address string) CheckResult {
	start := time.Now()

	config, err := c.getSSHConfig()
	if err != nil {
		return newFailedResult(time.Since(start), fmt.Errorf("ssh config error: %w", err))
	}

	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return newFailedResult(time.Since(start), fmt.Errorf("ssh dial error: %w", err))
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return newFailedResult(time.Since(start), fmt.Errorf("session error: %w", err))
	}
	defer session.Close()

	if err := session.Run(ctx.Value("cmd").(string)); err != nil {
		return newFailedResult(time.Since(start), fmt.Errorf("command error: %w", err))
	}

	return newSuccessResult(time.Since(start))
}

func (c *CMDChecker) getSSHConfig() (*ssh.ClientConfig, error) {
	var auth []ssh.AuthMethod

	if c.sshKeyPath != "" {
		key, err := os.ReadFile(filepath.Clean(c.sshKeyPath))
		if err != nil {
			return nil, fmt.Errorf("unable to read SSH key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("unable to parse SSH key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		socket := os.Getenv("SSH_AUTH_SOCK")
		if socket != "" {
			conn, err := net.Dial("unix", socket)
			if err != nil {
				return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
			}

			agentClient := agent.NewClient(conn)
			auth = append(auth, ssh.PublicKeysCallback(agentClient.Signers))
		}
	}

	if len(auth) == 0 {
		return nil, fmt.Errorf("no SSH authentication methods available")
	}

	return &ssh.ClientConfig{
		User:            os.Getenv("SSH_USER"),
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}, nil
}
