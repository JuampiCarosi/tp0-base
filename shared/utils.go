package shared

import (
	"net"
)

func WriteSafe(conn net.Conn, message []byte) error {
	written, err := conn.Write(message)
	if err != nil {
		return err
	}

	for written < len(message) {
		tmp, err := conn.Write(message[written:])
		if err != nil {
			return err
		}
		written += tmp
	}
	return nil
}
