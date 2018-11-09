package exception_test

import (
	"fmt"
	"github.com/eosspark/eos-go/exception/try"
	. "github.com/eosspark/eos-go/exception"
	"github.com/stretchr/testify/assert"
	"testing"
	)

func TestExceptionCode(t *testing.T) {
	assert.Equal(t, ExcTypes(21), DivideByZeroCode)
	assert.Equal(t, ExcTypes(10), AssertExceptionCode)
	assert.Equal(t, ExcTypes(15), UnknownHostExceptionCode)
}

func TestEosAssert(t *testing.T) {
	EosAssert(true, &BlockValidateException{}, "block #%s error :%s", "00000006367c1f4...", "msg")
}

func catches() {
	fmt.Println("\ncaught Exception------------------------")
	try.Try(func() {
		EosAssert(false, &ChainException{}, "test")
	}).Catch(func(e Exception) {
	}).End()

	fmt.Println("\ncaught none-----------------------------")
	try.Try(func() {
	}).Catch(func(e Exception) {
	}).End()

	defer func() {
		recover()
	}()
	fmt.Println("\ncaught failed---------------------------")
	try.Try(func() {
		try.Throw(1)
	}).Catch(func(e string) {
	}).Catch(func(e Exception) {
	}).End()
}

func catchExc() {
	fmt.Println("\ncaughtExc success-----------------------")
	try.Try(func() {
		EosAssert(false, &ChainException{}, "test")
	}).CatchException(func(e Exception) {
	}).End()

	defer func() { recover() }()
	fmt.Println("\ncaughtExc failed------------------------")
	try.Try(func() {
		try.Throw(1)
	}).CatchException(func(e Exception) {
	}).End()
}

func Test_performance(t *testing.T) {
	catches()
	//catchExc()
}

func TestEosAssert_catch(t *testing.T) {
	var scopeExit int
	defer func() {
		assert.Equal(t, 1, scopeExit)
	}()

	try.Try(func() {
		EosAssert(false, &ChainException{}, "test")
	}).Catch(func(e Exception) {
		fmt.Println(e.What())
	}).End()

	scopeExit = 1

}

func TestException_catch_same(t *testing.T) {
	try.Try(func() {
		EosAssert(false, &NameTypeException{}, "name error")

	}).Catch(func(e NameTypeException) {
		assert.Equal(t, "name error", e.Message())

	}).End()
}

func TestException_catch_same_pointer(t *testing.T) {
	try.Try(func() {
		EosAssert(false, &NameTypeException{}, "name error")

	}).Catch(func(e NameTypeException) {
		assert.Equal(t, "name error", e.Message())
		assert.Equal(t, ExcTypes(3010001), e.Code())

	}).End()
}

func TestException_catch_diff(t *testing.T) {
	try.Try(func() {
		try.Try(func() {
			EosAssert(false, &NameTypeException{}, "name error")

		}).Catch(func(e BlockValidateException) {
			// BlockValidateException is not conclude NameTypeException, can't be caught

		}).End()

	}).Catch(func(e Exception) {
		assert.Equal(t, "name error", e.Message())
		assert.Equal(t, ExcTypes(3010001), e.Code())

	}).End()
}

func TestException_catch_diff_pointer(t *testing.T) {
	try.Try(func() {
		try.Try(func() {
			EosAssert(false, &NameTypeException{}, "name error")

		}).Catch(func(e BlockValidateException) {
			// BlockValidateException is not conclude NameTypeException, can't be caught

		}).End()

	}).Catch(func(e Exception) {
		assert.Equal(t, "name error", e.Message())
		assert.Equal(t, ExcTypes(3010001), e.Code())

	}).End()
}

func TestException_catch_interface(t *testing.T) {
	try.Try(func() {
		EosAssert(false, &NameTypeException{}, "name error")

	}).Catch(func(e ChainTypeExceptions) {
		assert.Equal(t, "name error", e.Message())
		assert.Equal(t, ExcTypes(3010001), e.Code())

	}).End()
}

func TestExceptions(t *testing.T) {
	try.Try(func() {
		EosAssert(false, &ChainTypeException{}, "wrong chain type of type:%s", "abc")
	}).Catch(func(e ChainExceptions) {
		assert.Equal(t, "wrong chain type of type:abc", e.Message())
	}).End()

	try.Try(func() {
		EosAssert(false, &ChainException{}, "wrong chain id:%d", 12345)
	}).Catch(func(e Exception) {
		assert.Equal(t, "wrong chain id:12345", e.Message())
	}).End()

	try.Try(func() {
		EosAssert(false, &BlockValidateException{}, "test")
	}).Catch(func(e BlockValidateExceptions) {
		assert.Equal(t, "test", e.Message())
	}).End()

	try.Try(func() {
		EosAssert(false, &ChainTypeException{}, "test")
	}).Catch(func(e ChainTypeException) {
		fmt.Println(e.Message())
	}).End()

	//TODO more exceptions
}

func TestReThrow(t *testing.T) {
	try.Try(func() {
		try.Try(func() {
			EosAssert(false, &ChainTypeException{}, "wrong chain type of type:%s", "abc")
		}).Catch(func(e Exception) {
			try.Throw(e) // always == panic(e)
		}).End()

	}).Catch(func(e ChainTypeExceptions) {

		assert.Equal(t, "wrong chain type of type:abc", e.Message())
	}).End()

}

func returnFunction(a int) (r int) {

	defer try.HandleReturn()
	try.Try(func() {
		if a == 0 {
			r = -1       // return -1
			try.Return() //
		}

		EosAssert(a != 1, &ChainTypeException{}, "error")

	}).Catch(func(e ChainTypeExceptions) {
		r = 1        // return 1
		try.Return() //

	}).End()

	return 0

}
func TestReturnFunction(t *testing.T) {
	assert.Equal(t, -1, returnFunction(0))
	assert.Equal(t, 1, returnFunction(1))
	assert.Equal(t, 2, returnFunction(2))
	assert.Equal(t, 0, returnFunction(3))
}

func TestCatchExceptionsRethrow(t *testing.T) {
	try.Try(func() {
		EosAssert(false, &ChainTypeException{}, "")
	}).Catch(func(e Exception) {
		try.Try(func() {
			try.Throw(e)
		}).Catch(func(e ChainTypeException) {
			//shouldn't throw
		}).End()
	}).End()
}
