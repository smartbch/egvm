package executor

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"

	"github.com/smartbch/pureauth/egvm-script/types"
)

type Sandbox struct {
	name   string
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (b *Sandbox) executeJob(job *types.LambdaJob) (*types.LambdaResult, error) {
	input, _ := job.MarshalMsg(nil)
	reader := bufio.NewReader(b.stdout)
	scanner := bufio.NewScanner(reader)
	b.stdin.Write(input)
	b.stdin.Write([]byte("\n"))
	var out []byte
	for scanner.Scan() {
		out = scanner.Bytes()
		break
	}
	var res types.LambdaResult
	err := json.Unmarshal(out, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func NewAndStartSandbox(name string) *Sandbox {
	cmd := exec.Command("./new_sandbox.sh", name)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	if string(out) != "success" {
		panic("new sandbox failed!")
	}
	stdin, err := cmd.StdinPipe()
	if nil != err {
		panic("Error obtaining stdin: " + err.Error())
	}
	stdout, err := cmd.StdoutPipe()
	if nil != err {
		panic("Error obtaining stdout: " + err.Error())
	}
	cmd.Stderr = os.Stderr

	return &Sandbox{name: name, stdin: stdin, stdout: stdout}
}
