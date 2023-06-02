package executor

import (
	"io"
	"os"
	"os/exec"

	"github.com/tinylib/msgp/msgp"

	"github.com/smartbch/egvm/egvm-script/types"
)

type Sandbox struct {
	name   string
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (b *Sandbox) executeJob(job *types.LambdaJob) (*types.LambdaResult, error) {
	bz, err := job.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	b.stdin.Write(bz)
	// todo: why code commented below not work ?
	//err = job.EncodeMsg(msgp.NewWriter(b.stdin))
	//if err != nil {
	//	return nil, err
	//}
	var res types.LambdaResult
	err = res.DecodeMsg(msgp.NewReader(b.stdout))
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
