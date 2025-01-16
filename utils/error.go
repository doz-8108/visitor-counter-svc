package utils

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type customError struct {
	code *codes.Code
	error
}

type Err struct {
	Logger *zap.SugaredLogger
}

// store the custom error to the pointer from caller
func (e Err) HandleError(err *error) {
	if r := recover(); r != nil {
		ce, ok := r.(customError)
		if !ok {
			errMsg := fmt.Sprintf("Error is not in format of a custom error: %v, failed to handle error", ce)
			e.Logger.Error(errMsg)
			*err = status.Error(codes.Internal, errMsg)
		}

		e.Logger.Error(ce.Error())
		if ce.code != nil {
			*err = status.Error(*ce.code, ce.Error())
		} else {
			*err = status.Error(codes.Internal, ce.Error())
		}
	}
}

func (e Err) CatchError(err error) {
	if err == redis.Nil {
		return
	}

	if err != nil {
		panic(customError{error: err})
	}
}

func (e Err) CatchErrorWithCode(err error, customCode codes.Code) {
	if err == redis.Nil {
		return
	}

	if err != nil {
		panic(customError{code: &customCode, error: err})
	}
}
