package service

import (
	"context"
	"errors"

	"vibemeet/pkg/logger"

	"github.com/pion/mediadevices"
)

type AudioCaptureService interface {
	StartCapture(ctx context.Context) (mediadevices.MediaStream, error)
	StopCapture()
}

type audioCaptureService struct {
	log        logger.Logger
	stream     mediadevices.MediaStream
	cancelFunc context.CancelFunc
}

func NewAudioCaptureService(log logger.Logger) AudioCaptureService {
	return &audioCaptureService{
		log: log,
	}
}

func (s *audioCaptureService) StartCapture(ctx context.Context) (mediadevices.MediaStream, error) {
	// Note: GetUserMedia is typically a browser API, not available server-side
	// Server-side audio capture would require platform-specific audio capture libraries
	// For now, return an error - the screen capture handler will handle this gracefully
	// by checking if audio is already included in the screen stream (via GetDisplayMedia)
	s.log.Warn("Audio capture via GetUserMedia not available server-side")
	return nil, errors.New("server-side audio capture not implemented - use screen capture with audio")
}

func (s *audioCaptureService) StopCapture() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	if s.stream != nil {
		// Close all tracks in the stream
		for _, track := range s.stream.GetTracks() {
			track.Close()
		}
	}
}
