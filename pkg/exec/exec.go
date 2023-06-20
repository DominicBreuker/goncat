package exec

import (
	"fmt"
	"net"
	"os/exec"
)

func Run(conn net.Conn, program string) error {
	cmd := exec.Command(program)

	cmd.Stdout = conn
	cmd.Stdin = conn
	cmd.Stderr = conn

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Run(): %s", err)
	}

	cmd.Wait()
	conn.Close()

	return nil
}
