//go:build !cgo
// +build !cgo

package service

import (
	"context"
	"errors"

	"vibemeet/pkg/logger"

	"github.com/pion/mediadevices"
)

// Stub implementation for when CGO is disabled (e.g., local Windows development)
// Full implementation with codecs is in screen_capture.go (requires CGO)

type ScreenCaptureService interface {
	StartCapture(ctx context.Context) (mediadevices.MediaStream, error)
	StopCapture()
}

type screenCaptureService struct {
	log        logger.Logger
	stream     mediadevices.MediaStream
	cancelFunc context.CancelFunc
}

func NewScreenCaptureService(log logger.Logger) ScreenCaptureService {
	return &screenCaptureService{
		log: log,
	}
}

func (s *screenCaptureService) StartCapture(ctx context.Context) (mediadevices.MediaStream, error) {
	s.log.Error("Screen capture requires CGO and codec libraries (libvpx, libopus). Build with CGO_ENABLED=1 in Docker.")
	return nil, errors.New("screen capture not available: requires CGO - use Docker build")
}

func (s *screenCaptureService) StopCapture() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	if s.stream != nil {
		for _, track := range s.stream.GetTracks() {
			track.Close()
		}
	}
}

var ErrNoDisplayFound = errors.New("no display found")
