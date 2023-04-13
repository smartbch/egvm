package executor

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/smartbch/pureauth/egvm-invoker/types"
)

type Sandbox struct {
	name string
}

func (b *Sandbox) executeJob(job *types.LambdaJob) (*types.LambdaResult, error) {
	input, _ := json.Marshal(job)
	cmd := exec.Command(b.name, string(input))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	var res types.LambdaResult
	err = json.Unmarshal(out, &res)
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
	return &Sandbox{name: name}
}
