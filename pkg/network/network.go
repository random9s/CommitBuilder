package network

import (
	"fmt"
	"net"
)

var portRangeLow = 9000
var portRangeHigh = 10000

func portIsAvailable(port string) bool {
	var avail bool
	ln, err := net.Listen("tcp", port)
	if err == nil {
		avail = true
	}
	ln.Close()
	return avail
}

//NextAvailablePort ...
func NextAvailablePort() int {
	var available int
	for i := portRangeLow; i < portRangeHigh; i++ {
		var port = fmt.Sprintf(":%d", i)
		if portIsAvailable(port) {
			available = i
			break
		}
	}
	fmt.Println("found available port :", available)
	return available
}
