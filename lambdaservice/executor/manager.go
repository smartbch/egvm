package executor

import (
	"errors"

	"github.com/smartbch/pureauth/lambdaservice/types"
)

type SandboxManager struct {
	BoxStatusMap map[*Sandbox]bool // store sandbox => isBusy
}

func NewSandboxManager(sandboxes []*Sandbox) *SandboxManager {
	m := SandboxManager{BoxStatusMap: map[*Sandbox]bool{}}
	for _, s := range sandboxes {
		m.BoxStatusMap[s] = false
	}
	return &m
}

func (s *SandboxManager) ExecuteJob(job *types.LambdaJob) (*types.LambdaResult, error) {
	box := s.findIdleSandbox()
	if box == nil {
		return nil, errors.New("all sandbox busy, try later") // todo: add job queue?
	}
	res, err := box.executeJob(job)
	return res, err
}

func (s *SandboxManager) findIdleSandbox() *Sandbox {
	// take advantage of the randomness of map traversal
	for box, busy := range s.BoxStatusMap {
		if !busy {
			return box
		}
	}
	return nil
}
