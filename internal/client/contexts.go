package client

import "context"

type CtxWithCancel struct {
	Ctx        context.Context
	CancelFunc context.CancelFunc
}

type Contexts struct {
	ChatCtx *CtxWithCancel
}

func (ctxwc *CtxWithCancel) Cancel() {
	if ctxwc.CancelFunc != nil {
		ctxwc.CancelFunc()
	}
}

func NewCtxWithCancel(ctx context.Context) *CtxWithCancel {
	if ctx == context.TODO() {
		ctx = context.Background()
	}
	newCtx, cancelFunc := context.WithCancel(ctx)
	return &CtxWithCancel{
		newCtx,
		cancelFunc,
	}
}
