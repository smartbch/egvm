package executor

import (
	"errors"
	"fmt"
	"sync"

	"github.com/smartbch/egvm/egvm-script/types"
)

var (
	defaultSandboxNums = 1
)

type SandboxManager struct {
	lock         sync.RWMutex
	BoxStatusMap map[*Sandbox]bool // store sandbox => isBusy
}

func NewSandboxManager(sandboxes []*Sandbox) *SandboxManager {
	m := SandboxManager{BoxStatusMap: map[*Sandbox]bool{}}
	if len(sandboxes) == 0 {
		sandboxes = make([]*Sandbox, 0, defaultSandboxNums)
		for i := 0; i < defaultSandboxNums; i++ {
			sandboxes = append(sandboxes, NewAndStartSandbox(fmt.Sprintf("sandbox%d", i)))
		}
	}
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
	if err == nil {
		s.lock.Lock()
		s.BoxStatusMap[box] = false
		s.lock.Unlock()
	}
	return res, err
}

func (s *SandboxManager) findIdleSandbox() *Sandbox {
	var b *Sandbox
	s.lock.Lock()
	// take advantage of the randomness of map traversal
	for box, busyOrDead := range s.BoxStatusMap {
		if !busyOrDead {
			b = box
			break
		}
	}
	s.BoxStatusMap[b] = true
	s.lock.Unlock()
	return b
}
