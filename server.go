package main

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const (
	tlsRecordHeaderLen = 5
	expectedRecords    = 4
	tlsMinorVersion    = 0x04
	dialTimeout        = 10 * time.Second
	handshakeTimeout   = 15 * time.Second
)

type SNIProxy struct {
	ListenAddr string
	TargetPort string // 默认目标端口
	Secret     string // 可选：用于进一步的特征校验（如果需要）
}

func NewSNIProxy(addr string) *SNIProxy {
	return &SNIProxy{
		ListenAddr: addr,
		TargetPort: "443",
	}
}

func (s *SNIProxy) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	log.Printf("[SNI Proxy] Listening on %s", s.ListenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[Accept] Error: %v", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *SNIProxy) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	// 设置握手超时
	_ = clientConn.SetReadDeadline(time.Now().Add(handshakeTimeout))

	var (
		fullPayload  []byte
		rawFragments [][]byte
		targetHost   string
	)

	// 1. 读取并校验 4 个 TLS 记录
	for i := 1; i <= expectedRecords; i++ {
		header := make([]byte, tlsRecordHeaderLen)
		if _, err := io.ReadFull(clientConn, header); err != nil {
			log.Printf("[%s] Read header %d failed: %v", clientConn.RemoteAddr(), i, err)
			return
		}

		// 指纹校验：必须是特定的 Minor Version (默认 0x04)
		if header[2] != tlsMinorVersion {
			log.Printf("[%s] Security Reject: Record %d MinorVersion=0x%02x, expected 0x%02x. Connection DROPPED.", 
				clientConn.RemoteAddr(), i, header[2], tlsMinorVersion)
			return
		}

		payloadLen := binary.BigEndian.Uint16(header[3:5])
		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(clientConn, payload); err != nil {
			log.Printf("[%s] Read payload %d failed: %v", clientConn.RemoteAddr(), i, err)
			return
		}

		// 保存原始完整数据用于后续“零修改”回放
		rawRec := append(header, payload...)
		rawFragments = append(rawFragments, rawRec)
		fullPayload = append(fullPayload, payload...)
	}

	// 2. 解析 SNI
	sni, err := extractSNI(fullPayload)
	if err != nil {
		log.Printf("[%s] SNI Extraction failed: %v", clientConn.RemoteAddr(), err)
		return
	}
	targetHost = sni
	log.Printf("[%s] Routing Request -> %s", clientConn.RemoteAddr(), targetHost)

	// 3. 拨号目标服务器 (默认采用 TargetPort)
	targetAddr := net.JoinHostPort(targetHost, s.TargetPort)
	upstreamConn, err := net.DialTimeout("tcp", targetAddr, dialTimeout)
	if err != nil {
		log.Printf("[%s] Dial %s failed: %v", clientConn.RemoteAddr(), targetAddr, err)
		return
	}
	defer upstreamConn.Close()

	// 清除超时设置，进入转发阶段
	_ = clientConn.SetReadDeadline(time.Time{})

	// 4. 回放原始分片记录给目标服务器
	for _, frag := range rawFragments {
		if _, err := upstreamConn.Write(frag); err != nil {
			log.Printf("[%s] Replay fragments to upstream failed: %v", clientConn.RemoteAddr(), err)
			return
		}
	}

	// 5. 进入双向全双工转发
	log.Printf("[%s] Tunnel established: %s <-> %s", clientConn.RemoteAddr(), clientConn.RemoteAddr(), targetAddr)
	s.relay(clientConn, upstreamConn)
	log.Printf("[%s] Tunnel closed", clientConn.RemoteAddr())
}

func (s *SNIProxy) relay(client, upstream net.Conn) {
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(upstream, client)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(client, upstream)
		errCh <- err
	}()

	// 等待任意一端关闭
	<-errCh
}

// extractSNI 实现标准的 TLS SNI 提取逻辑
func extractSNI(data []byte) (string, error) {
	if len(data) < 43 {
		return "", errors.New("payload too short")
	}

	// 跳过 Handshake Header (4) + Version (2) + Random (32)
	offset := 38

	// Session ID
	sessionIDLen := int(data[offset])
	offset += 1 + sessionIDLen
	if offset+2 > len(data) { return "", errors.New("malformed ClientHello") }

	// Cipher Suites
	csLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2 + csLen
	if offset+1 > len(data) { return "", errors.New("malformed ClientHello") }

	// Compression Methods
	compLen := int(data[offset])
	offset += 1 + compLen
	if offset+2 > len(data) { return "", nil } // No extensions

	// Extensions
	extensionsLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2
	end := offset + extensionsLen
	if end > len(data) { return "", errors.New("extension length mismatch") }

	for offset+4 <= end {
		eType := binary.BigEndian.Uint16(data[offset : offset+2])
		eLen := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		offset += 4
		if offset+eLen > end { break }

		if eType == 0x0000 { // Server Name Indication
			// List Length (2) + Type (1) + Name Length (2)
			if eLen < 5 { return "", errors.New("malformed SNI ext") }
			nameLen := int(binary.BigEndian.Uint16(data[offset+3 : offset+5]))
			if offset+5+nameLen > end { return "", errors.New("SNI name exceeds ext") }
			return strings.ToLower(string(data[offset+5 : offset+5+nameLen])), nil
		}
		offset += eLen
	}

	return "", errors.New("SNI extension not found")
}
