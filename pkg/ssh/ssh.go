package ssh

import (
	"fmt"
	"os"

	cryptossh "golang.org/x/crypto/ssh"
)

// newClient builds an authenticated SSH client for the given host and port.
// TODO: Replace InsecureIgnoreHostKey with proper known_hosts verification.
func newClient(host, sshUser, sshKeyPath string, port int) (*cryptossh.Client, error) {
	key, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, err
	}

	signer, err := cryptossh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	cfg := &cryptossh.ClientConfig{
		User: sshUser,
		Auth: []cryptossh.AuthMethod{
			cryptossh.PublicKeys(signer),
		},
		HostKeyCallback: cryptossh.InsecureIgnoreHostKey(), // #nosec G106 – TODO: use known_hosts
	}

	return cryptossh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), cfg)
}

// RunCommand executes a shell command on the remote host via SSH.
func RunCommand(host, command, sshUser, sshKeyPath string, port int) error {
	client, err := newClient(host, sshUser, sshKeyPath, port)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return err
	}

	fmt.Printf("Output from %s:\n%s\n", host, output)
	return nil
}

// RunScript reads a local script file and executes its contents on the remote host via SSH.
func RunScript(host, scriptPath, sshUser, sshKeyPath string, port int) error {
	client, err := newClient(host, sshUser, sshKeyPath, port)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return err
	}

	output, err := session.CombinedOutput(string(script))
	if err != nil {
		return err
	}

	fmt.Printf("Output from %s:\n%s\n", host, output)
	return nil
}
