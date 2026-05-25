//go:build cgo
// +build cgo

package service

import (
	"context"
	"errors"
	"image"
	"time"

	"vibemeet/pkg/logger"

	"github.com/kbinani/screenshot"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/opus"
	"github.com/pion/mediadevices/pkg/codec/vpx"
	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
)

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
	s.log.Info("Starting screen capture with GetDisplayMedia (includes audio)")

	// Configure the VP8 encoder for video
	vpx8Params, err := vpx.NewVP8Params()
	if err != nil {
		s.log.Error("Failed to create VP8 params", "error", err)
		return nil, err
	}
	vpx8Params.BitRate = 800000
	vpx8Params.KeyFrameInterval = 30

	// Configure the Opus encoder for audio
	opusParams, err := opus.NewParams()
	if err != nil {
		s.log.Error("Failed to create Opus params", "error", err)
		return nil, err
	}
	opusParams.BitRate = 64000

	codecSelector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&vpx8Params),
		mediadevices.WithAudioEncoders(&opusParams),
	)

	// Try GetDisplayMedia to capture the screen along with audio.
	// This should capture both video and audio in one call.
	s.log.Info("Attempting to use GetDisplayMedia for screen + audio capture")
	stream, err := mediadevices.GetDisplayMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			// Capture the screen
		},
		Audio: func(constraint *mediadevices.MediaTrackConstraints) {
			// Capture system audio
			constraint.SampleRate = prop.Int(48000)
			constraint.ChannelCount = prop.Int(2)
		},
		Codec: codecSelector,
	})

	if err != nil {
		s.log.Warn("GetDisplayMedia failed, falling back to custom screen capture", "error", err)
		// If GetDisplayMedia is unavailable, fall back to the legacy method
		return s.startCustomCapture(ctx)
	}

	s.stream = stream
	videoTracks := stream.GetVideoTracks()
	audioTracks := stream.GetAudioTracks()

	s.log.Info("Screen capture with GetDisplayMedia created successfully",
		"total_tracks", len(stream.GetTracks()),
		"video_tracks", len(videoTracks),
		"audio_tracks", len(audioTracks))

	if len(videoTracks) == 0 {
		s.log.Error("No video tracks in stream after GetDisplayMedia!")
		return nil, errors.New("no video tracks in stream")
	}

	return stream, nil
}

// startCustomCapture is the legacy screen capture method (no audio via GetDisplayMedia).
func (s *screenCaptureService) startCustomCapture(ctx context.Context) (mediadevices.MediaStream, error) {
	// Get the number of displays
	numDisplays := screenshot.NumActiveDisplays()
	if numDisplays == 0 {
		return nil, ErrNoDisplayFound
	}

	// Use the first display
	displayIndex := 0
	bounds := screenshot.GetDisplayBounds(displayIndex)
	s.log.Info("Starting custom screen capture", "display_index", displayIndex, "bounds", bounds)

	// Configure the VP8 encoder for video
	vpx8Params, err := vpx.NewVP8Params()
	if err != nil {
		s.log.Error("Failed to create VP8 params", "error", err)
		return nil, err
	}
	vpx8Params.BitRate = 800000
	vpx8Params.KeyFrameInterval = 30

	codecSelector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&vpx8Params),
	)

	// Create the custom video source
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	videoSource := &screenVideoSource{
		displayIndex: displayIndex,
		bounds:       bounds,
		fps:          30,
		log:          s.log,
		ctx:          ctx,
	}

	s.log.Info("Creating video track from screen source", "bounds", bounds, "fps", 30)

	// Build a track from the source
	track := mediadevices.NewVideoTrack(videoSource, codecSelector)
	s.log.Info("Video track created", "track_id", track.ID(), "kind", track.Kind())

	s.log.Info("Creating media stream from track")
	// Create the media stream
	stream, err := mediadevices.NewMediaStream(track)
	if err != nil {
		cancel()
		track.Close()
		s.log.Error("Failed to create media stream", "error", err)
		return nil, err
	}

	s.stream = stream
	videoTracks := stream.GetVideoTracks()
	s.log.Info("Custom screen capture stream created successfully",
		"total_tracks", len(stream.GetTracks()),
		"video_tracks", len(videoTracks))

	if len(videoTracks) == 0 {
		s.log.Error("No video tracks in stream after creation!")
		return nil, errors.New("no video tracks in stream")
	}

	return stream, nil
}

func (s *screenCaptureService) StopCapture() {
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

// screenVideoSource implements mediadevices.VideoSource.
type screenVideoSource struct {
	displayIndex int
	bounds       image.Rectangle
	fps          int
	log          logger.Logger
	ctx          context.Context
}

var frameCount int64

func (s *screenVideoSource) Read() (image.Image, func(), error) {
	// Non-blocking context check
	select {
	case <-s.ctx.Done():
		s.log.Info("Read() cancelled via context")
		return nil, nil, s.ctx.Err()
	default:
		// Context not cancelled, continue
	}

	img, err := screenshot.CaptureDisplay(s.displayIndex)
	if err != nil {
		s.log.Error("Failed to capture display", "error", err, "display_index", s.displayIndex)
		return nil, nil, err
	}

	frameCount++
	bounds := img.Bounds()

	// Log the first 5 frames and then every 30 frames after that
	if frameCount <= 5 || frameCount%30 == 0 {
		s.log.Info("Captured frame", "frame", frameCount, "size", bounds.Dx(), "x", bounds.Dy())
	}

	// Small delay to throttle the frame rate
	time.Sleep(time.Second / time.Duration(s.fps))

	return img, func() {}, nil
}

func (s *screenVideoSource) Close() error {
	return nil
}

func (s *screenVideoSource) ID() string {
	return "screen-capture"
}

func (s *screenVideoSource) Properties() []prop.Media {
	// Use RGBA since screenshot.CaptureDisplay returns RGBA;
	// mediadevices will convert to the required format automatically.
	return []prop.Media{
		{
			Video: prop.Video{
				Width:       s.bounds.Dx(),
				Height:      s.bounds.Dy(),
				FrameFormat: frame.FormatRGBA,
				FrameRate:   float32(s.fps),
			},
		},
	}
}

var ErrNoDisplayFound = errors.New("no display found")
