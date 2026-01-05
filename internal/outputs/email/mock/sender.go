package mock

import (
	"context"

	"github.com/bakkerme/curator-ai/internal/outputs/email"
)

type Sender struct {
	Messages []email.Message
	Err      error
}

func (s *Sender) Send(ctx context.Context, message email.Message) error {
	_ = ctx
	if s.Err != nil {
		return s.Err
	}
	s.Messages = append(s.Messages, message)
	return nil
}
