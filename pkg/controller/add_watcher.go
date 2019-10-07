package controller

import (
	"github.com/speak2jc/namespacewatch/pkg/controller/watcher"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, watcher.Add)
}
