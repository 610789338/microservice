// author: youjun
// date: 2021-03-11

package main


import (
	msf "ms_framework"
)

func main() {
	msf.Init()

	// msf.OnRemoteDiscover("YJ", "testService", "127.0.0.1", 6666)
	// msf.OnRemoteDiscover("YJ", "testService", "127.0.0.1", 5555)
	msf.Start()
}
