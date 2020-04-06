package plan

import "fmt"

type ShouldExecute func(log []byte) bool

type compoundCommand struct {
	left          Command
	right         Command
	shouldExecute ShouldExecute
}

func NewCompoundCommand(left Command, right Command, shouldExecute ShouldExecute) Command {
	return compoundCommand{
		left:          left,
		right:         right,
		shouldExecute: shouldExecute,
	}
}

func (c compoundCommand) String() string {
	return fmt.Sprintf("%s || %s", c.left, c.right)
}

func (c compoundCommand) Args() []string {
	panic("Args in compoundCommand should never be used")
}

func (c compoundCommand) Env() []string {
	return append(c.left.Env(), c.right.Env()...)
}

func (c compoundCommand) AddToArgs(args ...string) Command {
	c.left = c.left.AddToArgs(args...)
	return c
}

func (c compoundCommand) AddToEnv(env ...string) Command {
	panic("AddToEnv in compoundCommand should never be used")
}
