package core

import (
	"fmt"
	"net"
	"time"
	"log"
	"../packet/"
	"../config"
)

// BGP Finite State Machine state
const (
	_ = iota
	BGP_FSM_IDLE
	BGP_FSM_ACTIVE
	BGP_FSM_CONNECT
	BGP_FSM_OPENSENT
	BGP_FSM_OPENCONFIRM
	BGP_FSM_ESTABLISHED
)

const (
	_ = iota
	BGP_FSM_CONNECT_PROACTIVE
	BGP_FSM_CONNECT_REACTIVE
)

const (
	BGP_EVENT_MANUAL_STOP = 2
	BGP_EVENT_TCP_CONNECTION_VALID = 14
	BGP_EVENT_TCP_CONNECTION_CONFIRMED = 17
	BGP_EVENT_TCP_CONNECTION_FAILS = 18
)

const (
	_ = iota
	BGP_FSM_INTERNAL_EVT_PROCEED
	BGP_FSM_INTERNAL_EVT_STOP
)

const BGP_STANDARD_PORT = "179"
const BGP_CONNECT_RETRY_DURATION = 60
const BGP_TCP_CONN_TIMEOUT = 30

type BGPState int
type BGPEvent int
type NeighborInternalEvent int

type MessageHandler struct {
	conn		*net.TCPConn
	bytes_in	int
	bytes_out	int
	rest		[]byte
}

func (h *MessageHandler) initialize(conn *net.TCPConn) {
	h.conn = conn
}

func (h *MessageHandler) receiveMessage() (*packet.BGPMessage, error) {

	//TODO support fragment packet
	buf := make([]byte, 4096)
	read_len, err := h.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	h.bytes_in += read_len
	fmt.Println(read_len)
	msg, err := packet.ParseBGPMessage(buf)
	return msg, err
}

func (h *MessageHandler) sendMessage(msg *packet.BGPMessage) error {

	data, err := packet.SerializeBGPMessage(msg)
	write_len, err := h.conn.Write(data)
	if err != nil {
		return err
	}

	h.bytes_out += write_len
	return nil

}

func (h *MessageHandler) sendKeepAlive() error {

	var keepAlive packet.BGPMessage
	keepAlive.Header.Type = packet.BGP_MSG_KEEPALIVE
	keepAlive.Header.Len = 19 // MSG_MIN

	data, err := packet.SerializeBGPMessage(&keepAlive)
	write_len, err := h.conn.Write(data)
	if err != nil {
		return err
	}

	h.bytes_out += write_len
	return nil

}



func (h *MessageHandler) close() error {
	err := h.conn.Close()
	return err
}

type BGPFiniteStateMachine struct {

	state				BGPState
	previousState		BGPState
	IsProactive			bool
	messageHandler		*MessageHandler
	neighborConfig		*config.NeighborConfiguration
	globalConfig		*config.GlobalConfiguration
	connectRetryTimer	time.Timer
	connectRetryCount	int
	keepAliveTimer		time.Timer
	holdTimer			time.Timer
	receivedOpenMsg		packet.BGPBody
	stateChanged		chan struct{}
	eventChannel		chan NeighborInternalEvent
	internalSocket		net.TCPConn
	errorOccurred		chan error
}


func (fsm *BGPFiniteStateMachine) Close() {

	var msg packet.BGPMessage
	var notification packet.BGPNotification
	notification.ErrorCode = 1
	notification.ErrorSubcode = 1
	msg.Body = notification
	fsm.messageHandler.sendMessage(&msg)

	close(fsm.eventChannel)
	close(fsm.errorOccurred)
	close(fsm.stateChanged)
}

func (fsm *BGPFiniteStateMachine) Run() error {

	fsm.stateChanged <- struct {}{}

	for {
		// start finite state machine
		select {
		case evt := <- fsm.stateChanged:

			switch fsm.state {
			case BGP_FSM_IDLE:
				go fsm.handleIdle()
			case BGP_FSM_ACTIVE:
				go fsm.handleActive()
			case BGP_FSM_CONNECT:
				go fsm.handleConnect()
			case BGP_FSM_OPENSENT:
				go fsm.handleOpenSent()
			case BGP_FSM_OPENCONFIRM:
				go fsm.handleOpenConfirm()
			case BGP_FSM_ESTABLISHED:
				go fsm.handleEstablished()
			}

		case ctrl := <- fsm.eventChannel:
			if ctrl == BGP_EVENT_MANUAL_STOP {
				break
			}
		case err := <- fsm.errorOccurred:
			return fmt.Errorf(err.Error())
		}
	}
}

func (fsm *BGPFiniteStateMachine) Initialize(
					conn *net.TCPConn,
					isProactive bool,
					gConfig *config.GlobalConfiguration,
					nConfig *config.NeighborConfiguration,
					eventCh chan NeighborInternalEvent) {

	fsm.previousState = 0
	fsm.messageHandler = &MessageHandler{}
	fsm.messageHandler.conn = conn
	fsm.IsProactive = isProactive
	fsm.neighborConfig = nConfig
	fsm.globalConfig = gConfig
	fsm.eventChannel = eventCh
	fsm.state = BGP_FSM_IDLE
	fsm.stateChanged = make(chan struct{})
	fsm.errorOccurred = make(chan error)

}

func (fsm *BGPFiniteStateMachine) handleIdle() {

	//initialize connection
	fsm.connectRetryCount = 0
	//fsm.connectRetryTimer = time.NewTimer(BGP_CONNECT_RETRY_DURATION)
	fsm.stateChange(BGP_FSM_CONNECT)

}

func handleError(err error){
	if err != nil {
		log.Fatal(err)
	}
}

func (fsm *BGPFiniteStateMachine) handleConnect() {

	if fsm.previousState != BGP_FSM_IDLE || fsm.previousState != BGP_FSM_ACTIVE{
		fsm.errorOccurred <- fmt.Errorf("previous state is not idle or active.")
		return
	}

	tcpEvent := make(chan int)
	defer close(tcpEvent)

//	var alreadyTimedout bool = false

	go func() {
		fsm.connectRetryCount += 1
		conn, err := net.DialTimeout("tcp",fsm.neighborConfig.PeerAddress, BGP_TCP_CONN_TIMEOUT)
		if err != nil {
			tcpEvent <- BGP_EVENT_TCP_CONNECTION_FAILS
			return
		} else {
			// set socket
			fsm.messageHandler.conn = conn.(net.TCPConn)
			tcpEvent <- BGP_EVENT_TCP_CONNECTION_CONFIRMED
		}

//		if alreadyTimedout {
//			conn.Close()
//		} else {
//
//		}
	}()

	for {
		select {
		case evt := <- tcpEvent:
			if evt == BGP_EVENT_TCP_CONNECTION_CONFIRMED {
				//stop ConectionRetryTimer
				fsm.connectRetryTimer.Stop()
				var openMsg *packet.BGPOpen = new(packet.BGPOpen)
				openMsg.MyAS = fsm.globalConfig.MyAS
				openMsg.ID = fsm.globalConfig.ID
				openMsg.Version = 4
				openMsg.HoldTime = fsm.globalConfig.HoldTime
				err := fsm.messageHandler.sendMessage(*packet.BGPMessage(openMsg))
				handleError(err)
				fsm.holdTimer = time.NewTimer(time.Minute * 4)
				fsm.stateChange(BGP_FSM_OPENSENT)

				break
			} else if evt == BGP_EVENT_TCP_CONNECTION_FAILS {
				fsm.connectRetryTimer.Reset(BGP_CONNECT_RETRY_DURATION)
				fsm.stateChange(BGP_FSM_ACTIVE)
				break
			}
//		case <- fsm.connectRetryTimer.C:
//			// go to Active state
//			alreadyTimedout = true
		}
	}
	return
}

func (fsm *BGPFiniteStateMachine) stateChange(newState BGPState){
	fsm.previousState = fsm.state
	fsm.state = newState
	fsm.stateChanged <- struct{} {}
}

func (fsm *BGPFiniteStateMachine) handleActive() {
	if fsm.previousState != BGP_FSM_IDLE || fsm.previousState != BGP_FSM_CONNECT{
		fsm.errorOccurred <- fmt.Errorf("previous state is not idle or connect.")
		return
	}

	for {
		select {
		case <- fsm.connectRetryTimer.C:
			fsm.stateChange(BGP_FSM_CONNECT)
			break
		}
	}
}

func (fsm *BGPFiniteStateMachine) handleOpenSent() {

	if fsm.previousState != BGP_FSM_CONNECT || fsm.previousState != BGP_FSM_ACTIVE {
		fsm.errorOccurred <- fmt.Errorf("previous state is not connect or active.")
		return
	}

	recvCh := make(chan *packet.BGPMessage)
	defer close(recvCh)
	go fsm.receiveMessage(recvCh, false)

	for {
		select {
		case ctrl := <- fsm.eventChannel:
			if ctrl == BGP_EVENT_MANUAL_STOP {
				break
			}
		case msg := <- recvCh:

			fsm.receivedOpenMsg = packet.BGPOpen(msg.Body)

			holdtime := fsm.receivedOpenMsg.HoldTime
			if holdtime < fsm.globalConfig.HoldTime {
				holdtime = fsm.globalConfig.HoldTime
			}

			// send KeepAlive message
			fsm.messageHandler.sendKeepAlive()
			fsm.stateChange(BGP_FSM_OPENCONFIRM)
			break
		}
	}
}

func (fsm *BGPFiniteStateMachine) handleOpenConfirm() {

	if fsm.previousState != BGP_FSM_OPENSENT {
		fsm.errorOccurred <- fmt.Errorf("previous state is not opensent.")
		return
	}

	recvCh := make(chan packet.BGPMessage)
	defer recvCh.Close()
	go fsm.receiveMessage(recvCh, false)

	for {
		select {
		case ctrl := <- fsm.eventChannel:
			if ctrl == BGP_EVENT_MANUAL_STOP {
				break
			}
		case msg := <-recvCh:
			switch msg.Header.Type {
			case BGP_MSG_NOTIFICATION:
				msg.Body = &packet.BGPNotification{}
			case BGP_MSG_KEEPALIVE:
				fsm.stateChange(BGP_FSM_ESTABLISHED)
			default:
				fsm.errorOccurred <- fmt.Errorf("invalid message.")
			}
			break
		}
	}
	return
}

func (fsm *BGPFiniteStateMachine) handleEstablished() {

	if fsm.previousState != BGP_FSM_OPENCONFIRM {
		fsm.errorOccurred <- fmt.Errorf("previous state is not openconfirm.")
		return
	}

	recvCh := make(chan packet.BGPMessage)
	defer recvCh.Close()
	go fsm.receiveMessage(recvCh, true)

	for {
		select {
		case ctrl := <- fsm.eventChannel:
			if ctrl == BGP_EVENT_MANUAL_STOP {
				break
			}
		case msg := <-recvCh:
			switch msg.Header.Type {
			case BGP_MSG_NOTIFICATION:
				//do something
				break
			case BGP_MSG_KEEPALIVE:
				fsm.messageHandler.sendKeepAlive()
			case BGP_MSG_UPDATE:
				//do handleUpdate
			default:
				fsm.errorOccurred <- fmt.Errorf("invalid message.")
			}
		}
	}
	return
}

func (fsm *BGPFiniteStateMachine) receiveMessage(recvCh chan *packet.BGPMessage, forever bool){
	for {

		msg, err := fsm.messageHandler.receiveMessage()
		if err != nil {
			fsm.errorOccurred <- err
			return
		}
		recvCh <- msg
		if !forever {
			break
		}

	}
}
