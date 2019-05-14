package main

import (
	"github.com/nextabc-lab/edgex"
	"github.com/pkg/errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	disconnected uint32 = iota
	connecting
	connected
)

// Socket客户端
type SockClient struct {
	conn     net.Conn
	opts     SockOptions
	execLock *sync.Mutex
	state    uint32
}

type SockOptions struct {
	Network           string        // 网络类型
	Addr              string        // 地址
	ReadTimeout       time.Duration // 读超时
	WriteTimeout      time.Duration // 写超时
	AutoReconnect     bool          // 是否自动重连接
	ReconnectInterval time.Duration //重连接周期
	KeepAlive         bool          // KeepAlive
	KeepAliveInterval time.Duration // KeepAlive周期
	BufferSize        uint          // 读写缓存大小
}

func (c SockOptions) IsValid() bool {
	return c.Network != "" && c.Addr != ""
}

////

func NewSockClient(opts ...SockOptions) *SockClient {
	var opt SockOptions
	if 0 < len(opts) {
		opt = opts[0]
	}
	return &SockClient{
		execLock: new(sync.Mutex),
		state:    connecting,
		opts:     opt,
	}
}

func (s *SockClient) SetOptions(config SockOptions) {
	s.opts = config
}

func (s *SockClient) Options() SockOptions {
	return s.opts
}

func (s *SockClient) BufferSize() uint {
	return s.opts.BufferSize
}

// Open 创建数据连接
func (s *SockClient) Open() (err error) {
	switch s.opts.Network {
	case "tcp", "TCP":
		if s.opts.AutoReconnect {
			log := edgex.ZapSugarLogger
			go func() {
				ticker := time.NewTicker(s.opts.ReconnectInterval)
				defer ticker.Stop()
				state := atomic.LoadUint32(&s.state)
				for disconnected != state {
					if connected != state {
						if err := s.connectTcp(); nil != err {
							log.Error("Connect tcp client failed: ", err)
						}
					}
					<-ticker.C
				}
			}()
			return nil
		} else {
			err = s.connectTcp()
		}
		return

	case "udp", "UDP":
		if addr, err := net.ResolveUDPAddr("udp", s.opts.Addr); nil != err {
			return errors.WithMessage(err, "Resolve udp address failed")
		} else if conn, err := net.DialUDP("udp", nil, addr); nil != err {
			return errors.WithMessage(err, "UDP dial failed")
		} else {
			atomic.StoreUint32(&s.state, connected)
			s.conn = conn
			return nil
		}

	default:
		return errors.New("Unknown network type: " + s.opts.Network)
	}
}

// Close 关闭数据连接
func (s *SockClient) Close() error {
	atomic.StoreUint32(&s.state, disconnected)
	if nil != s.conn {
		return s.conn.Close()
	} else {
		return nil
	}
}

// Receive 从数据连接中读取数据
func (s *SockClient) Receive(buff []byte) (n int, err error) {
	if connected != atomic.LoadUint32(&s.state) {
		return 0, errors.New("Client connection is not ready")
	}
	if err := s.conn.SetReadDeadline(time.Now().Add(s.opts.ReadTimeout)); nil != err {
		return 0, s.checkSocketState(err)
	}
	n, err = s.conn.Read(buff)
	return n, s.checkSocketState(err)
}

// Send 向数据连接发送数据
func (s *SockClient) Send(data []byte) (n int, err error) {
	if connected != atomic.LoadUint32(&s.state) {
		return 0, errors.New("Client connection is not ready")
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(s.opts.WriteTimeout)); nil != err {
		return 0, s.checkSocketState(err)
	}
	n, err = s.conn.Write(data)
	return n, s.checkSocketState(err)
}

// Execute 是一个线程安全的Send-Receive原子操作
func (s *SockClient) Execute(in []byte) (out []byte, err error) {
	s.execLock.Lock()
	defer s.execLock.Unlock()
	buffer := make([]byte, s.BufferSize())
	if _, err := s.Send(in); nil != err {
		return nil, err
	}
	if n, err := s.Receive(buffer); nil != err {
		return nil, err
	} else {
		return buffer[:n], nil
	}
}

func (s *SockClient) checkSocketState(err error) error {
	if !IsNetTempErr(err) {
		atomic.StoreUint32(&s.state, connecting)
	}
	return err
}

func (s *SockClient) connectTcp() (err error) {
	if s.conn, err = newTcpConn(s.opts.Addr, &s.opts); nil == err {
		atomic.StoreUint32(&s.state, connected)
	}
	return err
}

func newTcpConn(remoteAddr string, opts *SockOptions) (conn *net.TCPConn, err error) {
	addr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if nil != err {
		return nil, err
	}
	if conn, err := net.DialTCP("tcp", nil, addr); nil != err {
		return nil, errors.WithMessage(err, "TCP dial failed")
	} else {
		conn.SetKeepAlive(opts.KeepAlive)
		conn.SetKeepAlivePeriod(opts.KeepAliveInterval)
		return conn, nil
	}
}

func IsNetTempErr(err error) bool {
	if nErr, ok := err.(net.Error); ok {
		return nErr.Timeout() || nErr.Temporary()
	} else {
		return false
	}
}
