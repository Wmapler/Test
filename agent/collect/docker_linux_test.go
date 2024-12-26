//go:build linux
// +build linux

package collect

import (
	"encoding/json"
	"fmt"
)
import "testing"

func TestGetDocker(t *testing.T) {
	docker := GetDocker()
	if b, err := json.Marshal(docker); err != nil {
		return
	} else {
		fmt.Println("docker result: ", string(b))

	}
}
