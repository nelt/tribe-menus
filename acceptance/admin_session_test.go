package acceptance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/nelt/tribe-menus/internal/admin"
)

const sessionTimeout = 5 * time.Second

// defaultAnswers answer the questions a scenario does not mention.
var defaultAnswers = map[string]string{
	admin.AskTribeName:   "Tribu de test",
	admin.AskSlug:        "tribu-de-test",
	admin.AskEmail:       "alice@exemple.fr",
	admin.AskDisplayName: "",
}

// adminSession drives an interactive admin command, as an administrator at a terminal:
// it waits for each question before answering it.
type adminSession struct {
	in   *io.PipeWriter
	out  *outputBuffer
	done chan error

	read     int // length of the output already consumed
	finished bool
	err      error
}

// startInit starts admin init in the background.
func startInit(ctx context.Context, cmd *admin.Command) *adminSession {
	inReader, inWriter := io.Pipe()
	s := &adminSession{in: inWriter, out: newOutputBuffer(), done: make(chan error, 1)}
	cmd.In = inReader
	cmd.Out = s.out
	go func() {
		err := cmd.Init(ctx)
		_ = inReader.Close()
		s.done <- err
	}()
	return s
}

// next waits for the next question, or the end of the command (question ""), and returns
// it with the output shown since the previous one.
func (s *adminSession) next() (question, shown string, err error) {
	deadline := time.After(sessionTimeout)
	for {
		output := s.out.String()
		pending := output[s.read:]
		for q := range defaultAnswers {
			if strings.HasSuffix(pending, q) {
				s.read = len(output)
				return q, strings.TrimSuffix(pending, q), nil
			}
		}
		if s.finished {
			s.read = len(output)
			return "", pending, nil
		}
		select {
		case err := <-s.done:
			s.finished, s.err = true, err
		case <-s.out.changed:
		case <-deadline:
			return "", pending, fmt.Errorf("admin command: no question nor end after %s, output: %q", sessionTimeout, pending)
		}
	}
}

func (s *adminSession) write(answer string) error {
	if _, err := io.WriteString(s.in, answer+"\n"); err != nil {
		return fmt.Errorf("answer %q: %w", answer, err)
	}
	return nil
}

// answer answers the question, after answering the ones asked before it with defaults.
func (s *adminSession) answer(question, value string) error {
	for {
		q, _, err := s.next()
		if err != nil {
			return err
		}
		if q == "" {
			return fmt.Errorf("admin command ended before asking %q: %v", question, s.err)
		}
		if q == question {
			return s.write(value)
		}
		if err := s.write(defaultAnswers[q]); err != nil {
			return err
		}
	}
}

// finish answers the remaining questions with defaults and returns the result of the command.
func (s *adminSession) finish() error {
	for {
		q, _, err := s.next()
		if err != nil {
			return err
		}
		if q == "" {
			return s.err
		}
		if err := s.write(defaultAnswers[q]); err != nil {
			return err
		}
	}
}

// close interrupts the command if it is still waiting for an answer.
func (s *adminSession) close() error {
	_ = s.in.Close()
	if s.finished {
		return nil
	}
	select {
	case err := <-s.done:
		s.finished = true
		if errors.Is(err, admin.ErrInputClosed) {
			return nil
		}
		return err
	case <-time.After(sessionTimeout):
		return errors.New("admin command did not stop")
	}
}

// outputBuffer collects the output of the command and signals each write.
type outputBuffer struct {
	mu      sync.Mutex
	buf     strings.Builder
	changed chan struct{}
}

func newOutputBuffer() *outputBuffer {
	return &outputBuffer{changed: make(chan struct{}, 1)}
}

func (b *outputBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := b.buf.Write(p)
	select {
	case b.changed <- struct{}{}:
	default:
	}
	return n, err
}

func (b *outputBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
