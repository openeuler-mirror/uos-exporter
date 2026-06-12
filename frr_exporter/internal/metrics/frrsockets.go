
package metrics

import (
    "bytes"
    "fmt"
    "net"
    "path/filepath"
    "time"
)

type ProtocolType string

const (
    BFDProtocol       ProtocolType = "bfdd"
    BGPProtocol       ProtocolType = "bgpd"
    OSPFProtocol      ProtocolType = "ospfd"
    OSPFMultiProtocol ProtocolType = "ospfd-%d"
    PIMProtocol       ProtocolType = "pimd"
    VRRPProtocol      ProtocolType = "vrrpd"
    ZebraProtocol     ProtocolType = "zebra"
)

type SocketConnector interface {
    Dial(socketPath string) (net.Conn, error)
    SetDeadline(conn net.Conn, t time.Time) error
}

type DefaultSocketConnector struct{}

type CmdExecutor interface {
    Execute(cmd string) ([]byte, error)
}

type ProtocolCommandExecutor struct {
    protocol    ProtocolType
    instanceID  int
    dirPath     string
    timeout     time.Duration
    connector   SocketConnector
    cmdHandler  CommandHandler
}

type CommandHandler interface {
    PrepareConnection(conn net.Conn) error
    SendCommand(conn net.Conn, cmd string) error
    ReadResponse(conn net.Conn) ([]byte, error)
}

type VTYCommandHandler struct {
    bufferSize int
}

type Connection struct {
    dirPath    string
    timeout    time.Duration
    connector  SocketConnector
    executors  map[ProtocolType]CmdExecutor
}

func NewConnection(dirPath string, timeout time.Duration) *Connection {
    conn := &Connection{
        dirPath:   dirPath,
        timeout:   timeout,
        connector: &DefaultSocketConnector{},
        executors: make(map[ProtocolType]CmdExecutor),
    }
    conn.initExecutors()
    return conn
}

func (c *Connection) initExecutors() {
    c.executors[BFDProtocol] = c.newProtocolExecutor(BFDProtocol, 0)
    c.executors[BGPProtocol] = c.newProtocolExecutor(BGPProtocol, 0)
    c.executors[OSPFProtocol] = c.newProtocolExecutor(OSPFProtocol, 0)
    c.executors[PIMProtocol] = c.newProtocolExecutor(PIMProtocol, 0)
    c.executors[VRRPProtocol] = c.newProtocolExecutor(VRRPProtocol, 0)
    c.executors[ZebraProtocol] = c.newProtocolExecutor(ZebraProtocol, 0)
}

func (c *Connection) newProtocolExecutor(protocol ProtocolType, instanceID int) *ProtocolCommandExecutor {
    return &ProtocolCommandExecutor{
        protocol:   protocol,
        instanceID: instanceID,
        dirPath:    c.dirPath,
        timeout:    c.timeout,
        connector:  c.connector,
        cmdHandler: &VTYCommandHandler{bufferSize: 4096},
    }
}

func (c *Connection) ExecBFDCmd(cmd string) ([]byte, error) {
    return c.executors[BFDProtocol].Execute(cmd)
}

func (c *Connection) ExecBGPCmd(cmd string) ([]byte, error) {
    return c.executors[BGPProtocol].Execute(cmd)
}

func (c *Connection) ExecOSPFCmd(cmd string) ([]byte, error) {
    return c.executors[OSPFProtocol].Execute(cmd)
}

func (c *Connection) ExecOSPFMultiInstanceCmd(cmd string, instanceID int) ([]byte, error) {
    executor := c.newProtocolExecutor(OSPFMultiProtocol, instanceID)
    return executor.Execute(cmd)
}

func (c *Connection) ExecPIMCmd(cmd string) ([]byte, error) {
    return c.executors[PIMProtocol].Execute(cmd)
}

func (c *Connection) ExecVRRPCmd(cmd string) ([]byte, error) {
    return c.executors[VRRPProtocol].Execute(cmd)
}

func (c *Connection) ExecZebraCmd(cmd string) ([]byte, error) {
    return c.executors[ZebraProtocol].Execute(cmd)
}

func (d *DefaultSocketConnector) Dial(socketPath string) (net.Conn, error) {
    return net.DialUnix("unix", nil, &net.UnixAddr{Net: "unix", Name: socketPath})
}

func (d *DefaultSocketConnector) SetDeadline(conn net.Conn, t time.Time) error {
    return conn.SetDeadline(t)
}

func (p *ProtocolCommandExecutor) Execute(cmd string) ([]byte, error) {
    socketPath := p.getSocketPath()
    conn, err := p.connector.Dial(socketPath)
    if err != nil {
        return nil, err
    }
    defer conn.Close()

    if err := p.connector.SetDeadline(conn, time.Now().Add(p.timeout)); err != nil {
        return nil, err
    }

    if err := p.cmdHandler.PrepareConnection(conn); err != nil {
        return nil, err
    }

    if err := p.cmdHandler.SendCommand(conn, cmd); err != nil {
        return nil, err
    }

    return p.cmdHandler.ReadResponse(conn)
}

func (p *ProtocolCommandExecutor) getSocketPath() string {
    if p.protocol == OSPFMultiProtocol {
        return filepath.Join(p.dirPath, fmt.Sprintf(string(p.protocol), p.instanceID)+".vty")
    }
    return filepath.Join(p.dirPath, string(p.protocol)+".vty")
}

func (v *VTYCommandHandler) PrepareConnection(conn net.Conn) error {
    if _, err := conn.Write([]byte("enable\x00")); err != nil {
        return err
    }
    buf := make([]byte, v.bufferSize)
    _, err := conn.Read(buf)
    return err
}

func (v *VTYCommandHandler) SendCommand(conn net.Conn, cmd string) error {
    _, err := conn.Write([]byte(cmd + "\x00"))
    return err
}

func (v *VTYCommandHandler) ReadResponse(conn net.Conn) ([]byte, error) {
    var response bytes.Buffer
    buf := make([]byte, v.bufferSize)

    for {
        n, err := conn.Read(buf)
        if err != nil {
            return response.Bytes(), err
        }

        response.Write(buf[:n])

        if n > 0 && buf[n-1] == 0 {
            return bytes.TrimRight(response.Bytes(), "\x00"), nil
        }
    }
}
