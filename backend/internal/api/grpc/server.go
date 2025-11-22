package grpc

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	Port            int
	MaxRecvMsgSize  int
	MaxSendMsgSize  int
	EnableReflection bool
}

// DefaultServerConfig returns default gRPC server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:            4317,
		MaxRecvMsgSize:  16 * 1024 * 1024, // 16MB
		MaxSendMsgSize:  16 * 1024 * 1024, // 16MB
		EnableReflection: true,
	}
}

// Server wraps the gRPC server
type Server struct {
	grpcServer   *grpc.Server
	listener     net.Listener
	config       *ServerConfig
	logger       *zap.Logger
	traceService *OTLPTraceService
}

// NewServer creates a new gRPC server
func NewServer(config *ServerConfig, traceService *OTLPTraceService, logger *zap.Logger) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(config.MaxSendMsgSize),
	}

	grpcServer := grpc.NewServer(opts...)

	// Register OTLP trace service
	collectortrace.RegisterTraceServiceServer(grpcServer, traceService)

	// Enable reflection for debugging
	if config.EnableReflection {
		reflection.Register(grpcServer)
	}

	return &Server{
		grpcServer:   grpcServer,
		config:       config,
		logger:       logger,
		traceService: traceService,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	s.logger.Info("gRPC server listening",
		zap.String("addr", addr),
		zap.Int("port", s.config.Port),
	)

	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			s.logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop(ctx context.Context) error {
	stopped := make(chan struct{})

	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		s.grpcServer.Stop()
		return ctx.Err()
	case <-stopped:
		return nil
	}
}

// GetServer returns the underlying gRPC server
func (s *Server) GetServer() *grpc.Server {
	return s.grpcServer
}
