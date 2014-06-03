package gamerules

func makeTodoAspect() (aspect IBlockAspect) {
	return &TodoAspect{}
}

// TodoAspect has the same behaviour as that of a "void" block -
// i.e none. However, its purpose is intended to mark a block type
// whose behaviour is still to be implemented. A comment allows for
// notes to be made, but provides no functional change.
type TodoAspect struct {
	VoidAspect
	Comment string
}

func (aspect *TodoAspect) Name() string {
	return "Todo"
}
