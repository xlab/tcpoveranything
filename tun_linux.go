package main

import (
	"net"
	"os/exec"

	"code.google.com/p/tuntap"
)

func setupTun(tun *tuntap.Interface, addr *net.IPNet) error {
	if err := exec.Command("sudo", "ip", "-6", "addr", "add", "dev", tun.Name(), "local", addr.String()).Run(); err != nil {
		return err
	}
	if err := exec.Command("sudo", "ip", "link", "set", tun.Name(), "mtu", "1280", "up").Run(); err != nil {
		return err
	}
	return nil
}
