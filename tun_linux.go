package main

import (
	"os/exec"
	"net"

	"code.google.com/p/tuntap"
)

func checkExec(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func setupTun(tun *tuntap.Interface, addr *net.IPNet) error {
	if err := checkExec("sudo", "ip", "-6", "addr", "add", "dev", tun.Name(), "local", addr.String()); err != nil {
		return err
	}
	if err := checkExec("sudo", "ip", "link", "set", tun.Name(), "mtu", "1280", "up"); err != nil {
		return err
	}
	return nil
}
