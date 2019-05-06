package main

import (
	"fmt"

	"github.com/Safing/portbase/utils/osdetail"
)

func main() {
	names, err := osdetail.GetAllServiceNames()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", names)
}
