package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"neotunnel/util"
	"net"
	"net/url"
	"time"

	mcNet "github.com/Tnze/go-mc/net"
	mcPacket "github.com/Tnze/go-mc/net/packet"
	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
)

var ServerStatus util.ServerStatus
var ServerStatusError error
var ServerStatusLastUpdate time.Time

func handle(conn net.Conn, log *logrus.Entry) {
	defer conn.Close()
	handleHandShake(conn, log)
}

func handleHandShake(conn net.Conn, log *logrus.Entry) {
	mcConn := mcNet.Conn{
		Socket: conn,
		Reader: conn,
		Writer: conn,
	}
	mcConn.SetThreshold(-1)
	var (
		p                   mcPacket.Packet
		Protocol, Intention mcPacket.VarInt
		ServerAddress       mcPacket.String        // ignored
		ServerPort          mcPacket.UnsignedShort // ignored
	)
	mcConn.ReadPacket(&p)
	err := p.Scan(&Protocol, &ServerAddress, &ServerPort, &Intention)
	if err != nil {
		log.Printf("Error when reading handshake packet: %v", err)
		return
	}

	//log.Infof("ServerAddress: %v, ServerPort: %v, Protocol: %v, Intention: %v", ServerAddress, ServerPort, Protocol, Intention)

	switch int32(Intention) {
	default: // unknown error
		log.Printf("Unknown handshake intention: %d", Intention)
	case 1: // for status
		acceptListPing(mcConn)
	case 2: // for login
		handlePlaying(conn, p, log)
	}

}

func updateServerStatus() {
	logrus.WithField("ID", "Request").Debug("Update for server status")
	ServerStatusLastUpdate = time.Now()
	u := url.URL{Scheme: "https", Host: realHost, Path: "/status"}
	begin := time.Now()
	req, err := newHTTPCLient().Get(u.String())
	end := time.Now()
	if err != nil {
		ServerStatusError = err
		return
	}
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ServerStatusError = err
		return
	}

	var d util.HTTPResponse
	err = json.Unmarshal(bytes, &d)
	if d.Code != 200 {
		ServerStatusError = errors.New("Server Error")
		return
	}
	if err != nil {
		ServerStatusError = err
		return
	}

	d.Data.Delay += end.Sub(begin) / 3
	ServerStatus = d.Data
	ServerStatusError = nil
}

func getServerStatus() (*util.ServerStatus, error) {
	if ServerStatusLastUpdate.Add(time.Second * 5).Before(time.Now()) {
		updateServerStatus()
	}

	return &ServerStatus, ServerStatusError
}

// listResp return server status as JSON string
func listResp(ss *util.ServerStatus) string {
	list := util.PingList{}
	list.Version.Name = ss.Name
	list.Version.Protocol = ss.Protocol
	list.Players.Max = ss.MaxPlayerCount
	list.Players.Online = ss.OnlinePlayerCount
	list.Players.Sample = ss.SamplePlayers // must init. can't be nil
	list.Description = ss.Description

	data, err := json.Marshal(list)
	if err != nil {
		log.Panic("Marshal JSON for status checking fail")
	}
	return string(data)
}

func acceptListPing(conn mcNet.Conn) {
	var p mcPacket.Packet
	ss, err := getServerStatus()
	if err != nil {
		return
	}
	for i := 0; i < 2; i++ { // ping or list. Only accept twice
		err := conn.ReadPacket(&p)
		if err != nil {
			return
		}
		switch p.ID {
		case 0x00: // List
			err = conn.WritePacket(mcPacket.Marshal(0x00, mcPacket.String(listResp(ss))))
			//err = conn.WritePacket(mcPacket.Marshal(0x00, mcPacket.String(""))) //TODO: listResp()
		case 0x01: // Ping
			time.Sleep(ss.Delay)
			err = conn.WritePacket(p)
		}
		if err != nil {
			return
		}
	}
}

func handlePlaying(conn net.Conn, p mcPacket.Packet, log *logrus.Entry) {
	//如果超出访问速率，直接结束连接
	if !rateLimiter.Allow() {
		log.Errorf("Refuse to handle request from %v: connect too fast!", conn.RemoteAddr())
		return
	}

	// 如果正在处理，直接结束连接
	// if !handleStatusMutex.TryLock() {
	// 	log.Errorf("Refuse to handle request from %v: Only one connection can be established at the same time!", conn.RemoteAddr())
	// 	return
	// }
	// defer handleStatusMutex.Unlock()

	log.Infof("Accept to handle request from %v", conn.RemoteAddr())
	defer log.Infof("Finish to handle request from %v", conn.RemoteAddr())
	c, err := NewWSConnection()
	if err != nil {
		log.Errorf("Connect Websocket Server Error: %v", err)
		return
	}
	log.Infof("Connected Websocket Server")
	defer c.Close(websocket.StatusAbnormalClosure, "")
	netConn := websocket.NetConn(context.TODO(), c, websocket.MessageBinary)

	p.Pack(netConn, -1)

	go func() {
		for cnt, err := int64(0), error(nil); err == nil; cnt, err = io.CopyN(netConn, conn, 64) {
			util.AddUploadBytes(cnt)
		}
		log.Debug(err)
	}()

	for cnt, err := int64(0), error(nil); err == nil; cnt, err = io.CopyN(conn, netConn, 64) {
		util.AddDownloadBytes(cnt)
	}
	log.Debug(err)

}
