package log

import (
	"context"
	"testing"
	"time"
)

func TestContextTracer(t *testing.T) {
	// skip
	if testing.Short() {
		t.Skip()
	}

	ctx := AddTracer(context.Background())

	Tracer(ctx).Trace("api: request received, checking security")
	time.Sleep(1 * time.Millisecond)
	Tracer(ctx).Trace("login: logging in user")
	time.Sleep(1 * time.Millisecond)
	Tracer(ctx).Trace("database: fetching requested resources")
	time.Sleep(10 * time.Millisecond)
	Tracer(ctx).Warning("database: partial failure")
	time.Sleep(10 * time.Microsecond)
	Tracer(ctx).Trace("renderer: rendering output")
	time.Sleep(1 * time.Millisecond)
	Tracer(ctx).Trace("api: returning request")

	DebugTrace(ctx, "api: completed request")
	time.Sleep(100 * time.Millisecond)
}
