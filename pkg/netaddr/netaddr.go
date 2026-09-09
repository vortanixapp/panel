package netaddr

import "os"

func Listen(port string) string {
	return os.Getenv("BIND_ADDR") + ":" + port
}
