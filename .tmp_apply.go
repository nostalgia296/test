package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func main() {
	data, err := os.ReadFile("internal/image/download.go")
	if err != nil {
		panic(err)
	}
	repl := strings.Replace(string(data), "\n\t\"time\"\n", "\n", 1)
	if repl == string(data) {
		fmt.Println("no change")
		return
	}
	if err := os.WriteFile("internal/image/download.go", []byte(repl), 0o644); err != nil {
		panic(err)
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = "/data/data/com.termux/files/home/osc-ai"
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		panic(err)
	}
	cmd = exec.Command("go", "test", "./...")
	cmd.Dir = "/data/data/com.termux/files/home/osc-ai"
	out, err = cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		panic(err)
	}
	os.Remove("/data/data/com.termux/files/home/osc-ai/.tmp_apply.go")
}
