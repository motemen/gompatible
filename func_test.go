package gompatible

// make sure FuncChange implements Change
var _ = Change((*FuncChange)(nil))
