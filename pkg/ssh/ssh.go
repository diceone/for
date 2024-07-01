package ssh

import (
    "golang.org/x/crypto/ssh"
    "io/ioutil"
    "fmt"
)

func RunCommand(host, command, sshUser, sshKeyPath string) error {
    key, err := ioutil.ReadFile(sshKeyPath)
    if err != nil {
        return err
    }

    signer, err := ssh.ParsePrivateKey(key)
    if err != nil {
        return err
    }

    config := &ssh.ClientConfig{
        User: sshUser,
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(signer),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

    client, err := ssh.Dial("tcp", host+":22", config)
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

func RunScript(host, scriptPath, sshUser, sshKeyPath string) error {
    key, err := ioutil.ReadFile(sshKeyPath)
    if err != nil {
        return err
    }

    signer, err := ssh.ParsePrivateKey(key)
    if err != nil {
        return err
    }

    config := &ssh.ClientConfig{
        User: sshUser,
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(signer),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

    client, err := ssh.Dial("tcp", host+":22", config)
    if err != nil {
        return err
    }
    defer client.Close()

    session, err := client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()

    script, err := ioutil.ReadFile(scriptPath)
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
