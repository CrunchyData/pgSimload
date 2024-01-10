package main

import (
	"fmt"
	"os"
  "time"
  "golang.org/x/crypto/ssh"
)

// SSHClientConfig structure 
// Values comes from the `patroni.json` file
type SSHClientConfig struct {
	Host        string
	Port        int
	User        string
	PrivateKey  string
}

type SSHManager struct {
  Config      SSHClientConfig
	Client      *ssh.Client
}

// NewSSHManager to spawn a new SSHManager with the given configuration.
func NewSSHManager(config SSHClientConfig) *SSHManager {
	return &SSHManager{Config: config}
}

// EnsureConnected ensures that the SSH client is connected
// if not (m.Client == nil) -> reconnects  
func (m *SSHManager) EnsureConnected() error {
	if m.Client == nil {
		client, err := m.connectSSH()
		if err != nil {
			return err
		}
		m.Client = client
	}
	return nil
}

// connectSSH creates a new SSH client and returns its pointer
func (m *SSHManager) connectSSH() (*ssh.Client, error) {

  // privateKey could be read from a file, or retrieved from another storage
  // source, such as the Secret Service / GNOME Keyring
  file_privateKey, err := os.ReadFile(m.Config.PrivateKey)
  if err != nil {
    message := "\nCould not read SSH private key file defined in " 
    message = message + patroniconfigfilename.value +  ":\n"
    exit1(message, err)
  }

  key, err := ssh.ParsePrivateKey([]byte(file_privateKey))
  if err != nil {
    message := "\nCould not use SSH private key file defined in "
    message = message + patroniconfigfilename.value
    message = message + "\nYou may have entered the public key instead the private one ?\n"
    exit1(message, err)
  }

	// Create the SSH client config
	sshConfig := &ssh.ClientConfig{
		User: m.Config.User,
    // https://github.com/golang/go/issues/19767 
    // as clientConfig is non-permissive by default 
    // you can set ssh.InsercureIgnoreHostKey to allow any host 
    HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    Auth: []ssh.AuthMethod{
      ssh.PublicKeys(key),
    },
		Timeout:         5 * time.Second,
	}

  //JPAREM : maybe do a 1.0.2 to support both private key _and_ password
  //methods?
  //alternative with password 
  /*
  Auth: []ssh.AuthMethod{
      ssh.Password("PASSWORD"),
  */
   
	// Connect to the SSH server
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", m.Config.Host, m.Config.Port), sshConfig)
	if err != nil {
    message := "\nFailed to connect to SSH server: \n"
		return nil, fmt.Errorf(message+"%v", err)
	}

	return client, nil
}

// RunCommand to run the specified command in parameter
// to the current SSH client opened remotely
func (m *SSHManager) RunCommand(command string) (string, error) {
	// Ensure the SSH client is connected
	if err := m.EnsureConnected(); err != nil {
		return "", err
	}

	// Create a session
	session, err := m.Client.NewSession()
	if err != nil {
    //return "", fmt.Errorf("Failed to create SSH session: %v", err)
    message := "Failed to create SSH session: \n"
		return "", fmt.Errorf(message+"%v", err)
	}
	defer session.Close()

	// Run the command
	output, err := session.CombinedOutput(command)
	if err != nil {
		message := "Failed to run command: \n"
		return "", fmt.Errorf(message+"%v", err)
	}

	return string(output), nil
}
