package types

type LambdaJob struct {
	Script string   `msg:"script"` // lambdaJs
	Cert   string   `msg:"cert"`   // certs script will access
	Config string   `msg:"config"` // script config
	Inputs []string `msg:"inputs"`
	State  string   `msg:"state"` // to be resolved to orderedMap in sandbox
}

type LambdaResult struct {
	Outputs []string `msg:"outputs"`
	State   string   `msg:"state"` // usually, this is the serialized result of ordered map
}
