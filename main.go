package main

import (
	"fmt"
)
import "github.com/bdragon300/gozkaccess/sdk"

func main() {
	defer func() {
		fmt.Println("?????????")
		//if r := recover(); r != nil {
		//	fmt.Println("recovered from ", r)
		//	debug.PrintStack()
		//}
	}()
	z, err := sdk.NewZKSDK("plcommpro.dll")
	if err != nil {
		fmt.Println(err)
	}

	err = z.Connect("protocol=TCP,ipaddress=192.168.1.201,port=4370,timeout=4000,passwd=")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("!!!!!!!!!!")
	err = z.ControlDevice("output", 2, 2, 5, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
}