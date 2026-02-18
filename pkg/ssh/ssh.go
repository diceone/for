package ssh

import (
	"fmt"
	"os"
	"sync"

	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Config holds all SSH connection parameters.
type Config struct {
	User           string
	KeyPath        string
	Password       string
	Port           int
	// JumpHost is an optional bastion host in host:port form.
	JumpHost string
	// KnownHostsFile enables proper host-key verification.
	// When empty, InsecureIgnoreHostKey is used (not recommended for production).
	KnownHostsFile string
}

// ---------------------------------------------------------------------------
// Internal client factory
// ---------------------------------------------------------------------------

func newClient(host string, cfg Config) (*cryptossh.Client, error) {
	var authMethods []cryptossh.AuthMethod

	if cfg.KeyPath != "" {
		key, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, err
		}
		signer, err := cryptossh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		authMethods = append(authMethods, cryptossh.PublicKeys(signer))
	}

	if cfg.Password != "" {
		authMethods = append(authMethods, cryptossh.Password(cfg.Password))
	}

	var hostKeyCallback cryptossh.HostKeyCallback
	if cfg.KnownHostsFile != "" {
		cb, err := knownhosts.New(cfg.KnownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("loading known_hosts %q: %w", cfg.KnownHostsFile, err)
		}
		hostKeyCallback = cb
	} else {
		hostKeyCallback = cryptossh.InsecureIgnoreHostKey() // #nosec G106 – set known_hosts_file in config
	}

	clientCfg := &cryptossh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	addr := fmt.Sprintf("%s:%d", host, cfg.Port)

	if cfg.JumpHost != "" {
		jumpClient, err := cryptossh.Dial("tcp", cfg.JumpHost, clientCfg)
		if err != nil {
			return nil, fmt.Errorf("dial jump host %s: %w", cfg.JumpHost, err)
		}
		conn, err := jumpClient.Dial("tcp", addr)
		if err != nil {
			jumpClient.Close()
			return nil, fmt.Errorf("dial via jump host to %s: %w", addr, err)
		}
		ncc, chans, reqs, err := cryptossh.NewClientConn(conn, addr, clientCfg)
		if err != nil {
			jumpClient.Close()
			return nil, err
		}
		return cryptossh.NewClient(ncc, chans, reqs), nil
	}

	return cryptossh.Dial("tcp", addr, clientCfg)
}

// ---------------------------------------------------------------------------
// Connection pool (SSH multiplexing)
// ---------------------------------------------------------------------------

// Pool caches SSH client connections, keyed by user@host:port.
// Multiple goroutines may use the pool safely; each gets an independent session.
type Pool struct {
	mu      sync.Mutex
	clients map[string]*cryptossh.Client
}

// NewPool returns a new, empty connection pool.
func NewPool() *Pool {
	return &Pool{clients: make(map[string]*cryptossh.Client)}
}

func (p *Pool) key(host string, cfg Config) string {
	return fmt.Sprintf("%s@%s:%d", cfg.User, host, cfg.Port)
}

// session returns a new SSH session from a pooled (or freshly created) client.
// The returned cleanup function must be called (defer) to close the session.
func (p *Pool) session(host string, cfg Config) (*cryptossh.Session, func(), error) {
	k := p.key(host, cfg)

	p.mu.Lock()
	client, ok := p.clients[k]
	p.mu.Unlock()

	if ok {
		sess, err := client.NewSession()
		if err == nil {
			return sess, func() { sess.Close() }, nil
		}
		// Connection dead – remove and reconnect.
		p.mu.Lock()
		delete(p.clients, k)
		p.mu.Unlock()
	}

	client, err := newClient(host, cfg)
	if err != nil {
		return nil, nil, err
	}
	p.mu.Lock()
	p.clients[k] = client
	p.mu.Unlock()

	sess, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	return sess, func() { sess.Close() }, nil
}

// RunCommandOutput runs a command on the remote host using a pooled connection and
// returns the combined stdout+stderr output.
func (p *Pool) RunCommandOutput(host, command string, cfg Config) (string, error) {
	sess, cleanup, err := p.session(host, cfg)
	if err != nil {
		return "", err
	}
	defer cleanup()
	out, err := sess.CombinedOutput(command)
	return string(out), err
}

// RunScript uploads and executes a local script file via a pooled connection.
func (p *Pool) RunScript(host, scriptPath string, cfg Config) (string, error) {
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return p.RunCommandOutput(host, string(script), cfg)
}

// CopyFile uploads a local file to the remote host using a pooled connection.
func (p *Pool) CopyFile(host, src, dest string, cfg Config) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading local file %s: %w", src, err)
	}
	sess, cleanup, err := p.session(host, cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	if err := sess.Start(fmt.Sprintf("cat > %q", dest)); err != nil {
		return fmt.Errorf("starting copy to %s:%s: %w", host, dest, err)
	}
	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("writing data: %w", err)
	}
	stdin.Close()
	return sess.Wait()
}

// Close shuts down all cached connections.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.clients {
		c.Close()
	}
	p.clients = make(map[string]*cryptossh.Client)
}

// ---------------------------------------------------------------------------
// Stateless (non-pooled) helpers
// ---------------------------------------------------------------------------

// RunCommandOutput executes a command on the remote host and returns combined output.
func RunCommandOutput(host, command string, cfg Config) (string, error) {
	client, err := newClient(host, cfg)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	out, err := session.CombinedOutput(command)
	return string(out), err
}

// RunCommand executes a shell command on the remote host via SSH and prints output.
func RunCommand(host, command string, cfg Config) error {
	out, err := RunCommandOutput(host, command, cfg)
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	fmt.Printf("Output from %s:\n%s\n", host, out)
	return nil
}

// RunScript reads a local script file and executes it on the remote host via SSH.
func RunScript(host, scriptPath string, cfg Config) error {
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return err
	}
	out, err := RunCommandOutput(host, string(script), cfg)
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	fmt.Printf("Output from %s:\n%s\n", host, out)
	return nil
}

// CopyFile uploads a local file to the remote host via SSH stdin pipe.
func CopyFile(host, src, dest string, cfg Config) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading local file %s: %w", src, err)
	}

	client, err := newClient(host, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	if err := session.Start(fmt.Sprintf("cat > %q", dest)); err != nil {
		return fmt.Errorf("starting copy to %s:%s: %w", host, dest, err)
	}
	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("writing data: %w", err)
	}
	stdin.Close()
	if err := session.Wait(); err != nil {
		return fmt.Errorf("copy to %s:%s failed: %w", host, dest, err)
	}
	fmt.Printf("Copied %s -> %s:%s\n", src, host, dest)
	return nil
}

