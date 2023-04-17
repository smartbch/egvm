package types

//go:generate msgp

type LambdaJob struct {
	Script string `msg:"script"` // lambdaJs
	Cert   string `msg:"cert"`   // certs script will access
	Config string `msg:"config"` // script config
	Inputs []byte `msg:"inputs"`
	State  []byte `msg:"state"` // to be resolved to orderedMap in sandbox
}

type LambdaResult struct {
	Outputs []byte `msg:"outputs"`
	State   []byte `msg:"state"` // usually, this is the serialized result of ordered map
}
