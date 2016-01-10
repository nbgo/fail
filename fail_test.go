package fail_test

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"errors"
	"github.com/nbgo/fail"
	"fmt"
	"strings"
)

type MyError struct {
	msg    string
	reason error
}

func (this MyError) Error() string {
	return fmt.Sprintf("MyError: %v. Reason: %v", this.msg, this.reason)
}
func (this MyError) InnerError() error {
	return this.reason
}

func TestFail(t *testing.T) {
	Convey("Standard error", t, func() {
		err := errors.New("Error 1 occurred.")
		Convey("should have no inner", func() {
			So(fail.GetInner(err), ShouldBeNil)
		})
		Convey("should have no location", func() {
			So(fail.GetLocation(err), ShouldBeEmpty)
		})
		Convey("should have no stack trace", func() {
			So(fail.GetStackTrace(err), ShouldBeEmpty)
		})
	})

	Convey("Simple extended error", t, func() {
		errInner := errors.New("Error 1 occurred.")
		err := fail.New(errInner)
		Convey("should have correct message", func() {
			So(err.Error(), ShouldEqual, "Error 1 occurred.")
		})
		Convey("should have no inner", func() {
			So(fail.GetInner(err), ShouldBeNil)
		})
		Convey("should have correct location", func() {
			So(fail.GetLocation(err), ShouldContainSubstring, "github.com/nbgo/fail/fail_test.go:40")
			So(fail.GetLocation(err), ShouldContainSubstring, "TestFail.")
		})
		Convey("should have correct stack trace", func() {
			stackTrace := fail.GetStackTrace(err)
			So(stackTrace, ShouldContainSubstring, "TestFail.")
			So(stackTrace, ShouldContainSubstring, "fail_test.go:40")
			So(stackTrace, ShouldContainSubstring, "fail_test.go:57")
		})
	})

	Convey("Composite extended error", t, func() {
		err1 := fail.Newf("Error %v occurred.", 1)
		err2 := fail.News("Error 2 occurred.")
		err1c := fail.NewWithInner(err2, err1)
		err2c := fail.New(&MyError{"Error 3 occured", err1c})
		Convey("stack trace 1 check", func() {
			So(strings.Split(fail.GetStackTrace(err2c), "\n")[0], ShouldContainSubstring, "fail_test.go")
		})
		Convey("stack trace 2 check", func() {
			So(strings.Split(fail.GetStackTrace(err1c), "\n")[0], ShouldContainSubstring, "fail_test.go")
		})
		Convey("stack trace 3 check", func() {
			So(strings.Split(fail.GetStackTrace(err2), "\n")[0], ShouldContainSubstring, "fail_test.go")
		})
		Convey("stack trace 4 check", func() {
			So(strings.Split(fail.GetStackTrace(err1), "\n")[0], ShouldContainSubstring, "fail_test.go")
		})
		Convey("should have correct full details", func() {
			details := fail.GetFullDetails(err2c)
			fmt.Println("\n" + details)
			So(details, ShouldNotContainSubstring, "fail.extendedError")
		})
		Convey("should have correct message", func() {
			So(err2c.Error(), ShouldEqual, "MyError: Error 3 occured. Reason: Error 2 occurred.")
		})
		Convey("should have correct inner 1 level error", func() {
			inner := fail.GetInner(err2c)
			So(inner, ShouldNotBeNil)
			So(inner.Error(), ShouldEqual, "Error 2 occurred.")
			Convey("should have correct inner 2 level error", func() {
				inner2 := fail.GetInner(inner)
				So(inner2, ShouldNotBeNil)
				So(inner2.Error(), ShouldEqual, "Error 1 occurred.")
			})
		})
		Convey("GetOriginalError() should return correct error", func() {
			So(fail.GetOriginalError(err1c), ShouldEqual, fail.GetOriginalError(err2))
		})
	})

	Convey("ErrWithReason", t, func() {
		innerErr := fail.News("inner error")
		err := fail.NewErrWithReason("some error", innerErr)

		Convey("should have its own text and reason", func() {
			So(err.Error(), ShouldEqual, "some error: inner error")
		})

		Convey("should return correct inner error", func() {
			gotInnerErr := fail.GetInner(err)
			So(gotInnerErr, ShouldNotBeNil)
			So(gotInnerErr.Error(), ShouldEqual, "inner error")
			So(gotInnerErr, ShouldEqual, innerErr)
		})

		Convey("should have correct stack trace", func() {
			So(strings.Split(fail.GetStackTrace(err), "\n")[0], ShouldContainSubstring, "fail_test.go")
		})
	});

	Convey("StackTrace()", t, func() {
		stackTrace := fail.StackTrace()
		fmt.Println("\n" + stackTrace)
		So(strings.Split(stackTrace, "\n")[0], ShouldContainSubstring, "fail_test.go")

		func() {
			stackTrace := fail.StackTrace(1)
			fmt.Println("\n" + stackTrace)
			So(strings.Split(stackTrace, "\n")[0], ShouldContainSubstring, "fail_test.go")
		}()
	})

	Convey("IsError()", t, func() {
		innerErr := errors.New("test1")
		err2 := fail.NewErrWithReason("test2", innerErr)
		err3 := fail.NewErrWithReason("test3", err2)
		err4 := fail.News("test4")
		Convey("should return true when composite error has checked error in its hierarchy", func() {
			So(fail.IsError(err3, innerErr), ShouldBeTrue)
		})
		Convey("should return true when checked error is itself", func() {
			So(fail.IsError(innerErr, innerErr), ShouldBeTrue)
		})
		Convey("should return false when composite error does not have checked error in its hierarchy", func() {
			So(fail.IsError(err4, innerErr), ShouldBeFalse)
		})
	})

	Convey("AreErrorsOfEqualType()", t, func() {
		err1 := &MyError{}
		err2 := &MyError{}
		err3 := MyError{}
		err4 := fail.News("test")
		So(fail.AreErrorsOfEqualType(err1, err2), ShouldBeTrue)
		So(fail.AreErrorsOfEqualType(err1, err3), ShouldBeTrue)
		So(fail.AreErrorsOfEqualType(err3, err2), ShouldBeTrue)
		So(fail.AreErrorsOfEqualType(err1, err4), ShouldBeFalse)
		So(fail.AreErrorsOfEqualType(err3, err4), ShouldBeFalse)
	})

	Convey("GetErrorByType()", t, func() {
		innerErr := &MyError{}
		err2 := fail.NewErrWithReason("test2", innerErr)
		err3 := fail.NewErrWithReason("test3", err2)
		err4 := fail.News("test4")
		So(fail.GetErrorByType(err3, MyError{}), ShouldEqual, innerErr)
		So(fail.GetErrorByType(err4, MyError{}), ShouldBeNil)
	})
}