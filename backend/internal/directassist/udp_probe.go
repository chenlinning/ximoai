package directassist

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"time"
)

func ListenAndServeUDPProbe(ctx context.Context, cfg Config, store *Store, logger *slog.Logger) error {
	conn, err := net.ListenPacket("udp", cfg.UDPProbeAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	return ServeUDPProbe(ctx, conn, cfg, store, logger)
}

func ServeUDPProbe(ctx context.Context, conn net.PacketConn, cfg Config, store *Store, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	buffer := make([]byte, cfg.UDPPacketLimitBytes+1)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					return nil
				default:
					continue
				}
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		if n > cfg.UDPPacketLimitBytes {
			continue
		}
		payload := make([]byte, n)
		copy(payload, buffer[:n])
		go handleUDPProbePacket(ctx, conn, addr, payload, cfg, store, logger)
	}
}

func handleUDPProbePacket(ctx context.Context, conn net.PacketConn, addr net.Addr, payload []byte, cfg Config, store *Store, logger *slog.Logger) {
	var req UDPProbeRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return
	}
	if req.Type != "udp_probe" || req.DeviceID == "" || req.SessionID == "" || req.Nonce == "" || req.Token == "" {
		return
	}
	if req.Timestamp > 0 {
		delta := time.Since(time.Unix(req.Timestamp, 0))
		if delta > 5*time.Minute || delta < -5*time.Minute {
			return
		}
	}
	token, err := store.GetUDPToken(ctx, req.DeviceID)
	if err != nil || token != req.Token {
		return
	}
	session, err := store.GetSession(ctx, req.SessionID)
	if err != nil {
		return
	}
	if req.DeviceID != session.ControllerDeviceID && req.DeviceID != session.TargetDeviceID {
		return
	}
	host, port, ok := splitUDPAddr(addr)
	if !ok {
		return
	}
	resp := UDPProbeResponse{
		Type:       "udp_probe_result",
		PublicIP:   host,
		PublicPort: port,
		Nonce:      req.Nonce,
		ServerTime: time.Now().Unix(),
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	if len(data) > cfg.UDPPacketLimitBytes {
		return
	}
	if _, err := conn.WriteTo(data, addr); err != nil {
		logger.Debug("failed to write udp probe response", "error", err)
	}
}

func splitUDPAddr(addr net.Addr) (string, int, bool) {
	switch v := addr.(type) {
	case *net.UDPAddr:
		return v.IP.String(), v.Port, true
	default:
		host, portText, err := net.SplitHostPort(addr.String())
		if err != nil {
			return "", 0, false
		}
		port, err := net.LookupPort("udp", portText)
		if err != nil {
			return "", 0, false
		}
		return host, port, true
	}
}
