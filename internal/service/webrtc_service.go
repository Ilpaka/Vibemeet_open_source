package service

import (
	"context"
	"errors"
	"sync"

	"vibemeet/pkg/logger"

	"github.com/google/uuid"
	"github.com/pion/mediadevices"
	"github.com/pion/webrtc/v4"
)

type WebRTCService interface {
	CreatePeerConnection(ctx context.Context, screenStream, audioStream mediadevices.MediaStream) (*webrtc.PeerConnection, error)
	AddTracksToPeerConnection(peerConnection *webrtc.PeerConnection, screenStream, audioStream mediadevices.MediaStream) error
	ClosePeerConnection(peerConnectionID uuid.UUID) error
}

type webrtcService struct {
	log             logger.Logger
	peerConnections map[uuid.UUID]*webrtc.PeerConnection
	mu              sync.RWMutex
}

func NewWebRTCService(log logger.Logger) WebRTCService {
	return &webrtcService{
		log:             log,
		peerConnections: make(map[uuid.UUID]*webrtc.PeerConnection),
	}
}

func (s *webrtcService) CreatePeerConnection(ctx context.Context, screenStream, audioStream mediadevices.MediaStream) (*webrtc.PeerConnection, error) {
	// WebRTC configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create the PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// Store the connection
	peerConnectionID := uuid.New()
	s.mu.Lock()
	s.peerConnections[peerConnectionID] = peerConnection
	s.mu.Unlock()

	// Handle connection close
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		s.log.Info("Peer connection state changed", "state", state.String())
		if state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateFailed {
			s.mu.Lock()
			delete(s.peerConnections, peerConnectionID)
			s.mu.Unlock()
		}
	})

	return peerConnection, nil
}

// AddTracksToPeerConnection adds tracks to a PeerConnection.
func (s *webrtcService) AddTracksToPeerConnection(peerConnection *webrtc.PeerConnection, screenStream, audioStream mediadevices.MediaStream) error {
	// Add video tracks from the screen stream
	videoTracks := screenStream.GetVideoTracks()
	s.log.Info("Adding video tracks to peer connection", "count", len(videoTracks))

	if len(videoTracks) == 0 {
		s.log.Error("No video tracks in screen stream!")
		return errors.New("no video tracks in screen stream")
	}

	for i, track := range videoTracks {
		s.log.Info("Adding video track", "index", i, "track_id", track.ID(), "kind", track.Kind())

		// Try to add the track
		transceiver, err := peerConnection.AddTransceiverFromTrack(track, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendonly,
		})
		if err != nil {
			s.log.Error("Failed to add video track", "error", err, "track_id", track.ID())
			return err
		}

		s.log.Info("Video track added successfully",
			"track_id", track.ID(),
			"transceiver_direction", transceiver.Direction().String(),
			"transceiver_mid", transceiver.Mid())

		// Confirm the track was actually added
		if transceiver.Sender() == nil {
			s.log.Error("Transceiver has no sender after adding track!")
		} else {
			s.log.Info("Transceiver sender created", "track_id", transceiver.Sender().Track().ID())
		}
	}

	// Add audio tracks from the audio stream (if available)
	if audioStream != nil {
		audioTracks := audioStream.GetAudioTracks()
		s.log.Info("Adding audio tracks to peer connection", "count", len(audioTracks))

		if len(audioTracks) == 0 {
			s.log.Warn("Audio stream provided but has no audio tracks!")
		}

		for i, track := range audioTracks {
			s.log.Info("Adding audio track", "index", i, "track_id", track.ID(), "kind", track.Kind())
			transceiver, err := peerConnection.AddTransceiverFromTrack(track, webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendonly,
			})
			if err != nil {
				s.log.Error("Failed to add audio track, continuing without audio", "error", err, "track_id", track.ID())
				// Do not tear down the connection, just continue without audio
			} else {
				s.log.Info("Audio track added successfully",
					"track_id", track.ID(),
					"transceiver_direction", transceiver.Direction().String(),
					"transceiver_mid", transceiver.Mid())

				// Verify a sender was created
				if transceiver.Sender() == nil {
					s.log.Error("Transceiver has no sender after adding audio track!")
				} else {
					s.log.Info("Audio transceiver sender created", "track_id", transceiver.Sender().Track().ID())
				}
			}
		}
	} else {
		s.log.Info("No audio stream provided, continuing without audio")
	}

	s.log.Info("All tracks added to peer connection")
	return nil
}

func (s *webrtcService) ClosePeerConnection(peerConnectionID uuid.UUID) error {
	s.mu.RLock()
	peerConnection, exists := s.peerConnections[peerConnectionID]
	s.mu.RUnlock()

	if !exists {
		return ErrPeerConnectionNotFound
	}

	if err := peerConnection.Close(); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.peerConnections, peerConnectionID)
	s.mu.Unlock()

	return nil
}

var ErrPeerConnectionNotFound = errors.New("peer connection not found")
