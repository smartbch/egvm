package executor

import "github.com/smartbch/pureauth/lambdaservice/types"

type Sandbox struct {
}

func (b *Sandbox) executeJob(job *types.LambdaJob) (*types.LambdaResult, error) {
	return nil, nil //todo
}
