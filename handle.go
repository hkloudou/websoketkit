package websoketkit

import "sync"

var funcHandler = &sync.Map{}

//HandleFunc Handle Func
func HandleFunc(funcName string, fun func(data FunctionData)) {
	funcHandler.Store(funcName, fun)
}

//FireFunc fire the function
func FireFunc(data FunctionData) bool {
	if fun, found := funcHandler.Load(data.FuncName); found {
		go func() {
			fun.(func(data FunctionData))(data)
		}()
		return true
	}
	return false
}
