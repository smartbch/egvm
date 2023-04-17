package executor

import (
	"bufio"
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
	_, err := res.UnmarshalMsg(out)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func NewAndStartSandbox(name string) *Sandbox {
	cmd := exec.Command("./egvmscript")
	//if string(out) != "success" {
	//	panic("new sandbox failed!")
	//}
	stdin, err := cmd.StdinPipe()
	if nil != err {
		panic("Error obtaining stdin: " + err.Error())
	}
	stdout, err := cmd.StdoutPipe()
	if nil != err {
		panic("Error obtaining stdout: " + err.Error())
	}
	cmd.Stderr = os.Stderr
	go func() {
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
	}()
	return &Sandbox{name: name, stdin: stdin, stdout: stdout}
}
