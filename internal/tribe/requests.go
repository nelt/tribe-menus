package tribe

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// CodeRequestQueueSize is the number of accepted code requests waiting for their
// processing, beyond which a new one is dropped (ADR 0026). The limits by IP let through
// far fewer within the few milliseconds each one takes.
const CodeRequestQueueSize = 256

// TribeOpener returns the store of the tribe of a slug, or nil for an unknown tribe.
type TribeOpener func(ctx context.Context, slug string) (*Store, error)

// codeRequest is a code request accepted by the limits, whose member is looked for after
// the response.
type codeRequest struct {
	slug  string
	email Email
	token string
	at    time.Time
}

// CodeRequests is the queue of the code requests accepted by the limits (ADR 0026): the
// response is written before the member is looked for, so that nothing it shows, its time
// included, depends on the membership of the address (ENF-02).
type CodeRequests struct {
	mu      sync.Mutex
	closed  bool
	queue   chan codeRequest
	pending sync.WaitGroup
}

// NewCodeRequests returns a queue of size requests at most.
func NewCodeRequests(size int) *CodeRequests {
	return &CodeRequests{queue: make(chan codeRequest, size)}
}

// push queues a request, or reports false when the queue is full or closed.
func (q *CodeRequests) push(r codeRequest) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	q.pending.Add(1)
	select {
	case q.queue <- r:
		return true
	default:
		q.pending.Done()
		return false
	}
}

// Close stops accepting requests: ProcessCodeRequests returns once the queued ones are
// processed.
func (q *CodeRequests) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		q.closed = true
		close(q.queue)
	}
}

// Wait waits until every queued request is processed.
func (q *CodeRequests) Wait() {
	q.pending.Wait()
}

// ProcessCodeRequests processes the queued requests one at a time, until the queue is
// closed and empty. A request is processed to its end even when ctx is done, its response
// having told the browser that a code may come.
func (l *Login) ProcessCodeRequests(ctx context.Context) {
	ctx = context.WithoutCancel(ctx)
	for r := range l.Requests.queue {
		if err := l.issueCode(ctx, r); err != nil {
			l.Logger.Error("code request not processed", "error", err)
		}
		l.Requests.pending.Done()
	}
}

// issueCode stores and sends a code when the address of the request is an active member of
// an existing tribe. A request refused by the limit by tribe is counted for the alerts, and
// nothing is sent (ADR 0026).
func (l *Login) issueCode(ctx context.Context, r codeRequest) error {
	t, err := l.Tribes(ctx, r.slug)
	if err != nil {
		return fmt.Errorf("issue code: %w", err)
	}
	member, ok, err := activeMember(ctx, t, r.email)
	if err != nil || !ok {
		return err
	}
	code, err := newLoginCode()
	if err != nil {
		return fmt.Errorf("issue code: %w", err)
	}
	err = t.IssueLoginCode(ctx, member.ID, newRealCode(code, r.token, r.at), r.at)
	if errors.Is(err, ErrTooManyRequests) {
		return nil
	}
	if err != nil {
		return err
	}
	l.Mailer.SendLoginCode(string(r.email), code)
	return nil
}
