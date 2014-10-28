package core

import (
	"fmt"
	"net"
	"time"
	"log"
	"packet/packet"
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
)

const (
	_ = iota
	BGP_FSM_INTERNAL_EVT_PROCEED
	BGP_FSM_INTERNAL_EVT_STOP
)

const BGP_STANDARD_PORT = "179"
const BGP_CONNECT_RETRY_DURATION = 60

type BGPState int
type BGPEvent int

type MessageHandler struct {
	conn		*net.TCPConn
	bytes_in	int
	bytes_out	int
	rest		[]byte
}

func (h *MessageHandler) initialize(conn *net.TCPConn) {
	h.conn = conn
}

func (h *MessageHandler) receiveMessage(msgCh <- chan BGPMessage) error {

	//TODO support fragment packet
	buf := make([]byte, 4096)
	read_len, err := sock.Read(buf)
	if err != nil {
		return err
	}

	h.bytes_in += read_len
	fmt.Println(read_len)
	var msg BGPMessage
	msg = ParseBGPMessage(buf)
	msgCh <- msg

	return nil
}

func (h *MessageHandler) sendMessage(msg *BGPMessage) error {

	data, err := SerializeBGPMessage(msg)
	write_len, err = h.conn.Write(data)
	if err != nil {
		return err
	}

	h.bytes_out += write_len
	return nil

}

func (h *MessageHandler) run(sendCh , recvCh <- chan *BGPMessage) error {

	internalCh := make(chan BGPMessage)
	go h.receiveMessage(<-internalCh)
	for {
		select{
		case msg := <- internalCh:
			switch msg.Header.Type {
			case BGP_MSG_UPDATE:
			recvCh <- msg
			case BGP_MSG_NOTIFICATION:
			recvCh <- msg
			case BGP_MSG_KEEPALIVE:
				// respond to peer
			case BGP_MSG_ROUTE_REFRESH:
			recvCh <- msg
			default:
			recvCh <- msg
			}
		case msg := sendCh:
			h.sendMessage(msg)
		}
	}
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
	neighborConfig		*NeighborConfiguration
	globalConfig		*GlobalConfiguration
	connectRetryTimer	time.Timer
	connectRetryCount	int
	keepAliveTimer		time.Timer
	holdTimer			time.Timer
	receivedOpenMsg		BGPOpen
	changeState			chan int
}

func (fsm *BGPFiniteStateMachine) Run(ctrlCh chan BGPEvent) {

	fsm.state = BGP_FSM_IDLE
	fsm.changeState = BGP_FSM_INTERNAL_EVT_PROCEED

	for {
		// start finite state machine
		select {
		case evt := <- fsm.changeState:

			switch fsm.state{
			case BGP_FSM_IDLE:
				fsm.handleIdle()
			case BGP_FSM_ACTIVE:
				fsm.handleActive()
			case BGP_FSM_CONNECT:
				fsm.handleConnect()
			case BGP_FSM_OPENSENT:
				fsm.handleOpenSent()
			case BGP_FSM_OPENCONFIRM:
				fsm.handleIdle()
			case BGP_FSM_ESTABLISHED:
				fsm.handleIdle()
			}

		case ctrl <- ctrlCh:

		}
	}
}

func (fsm *BGPFiniteStateMachine) Initialize(sock *TCPConn, gConfig *GlobalConfiguration, nConfig *NeighborConfiguration, chan BGPEvent) {

	fsm.previousState = nil
	fsm.messageHandler = &MessageHandler{}
	if sock != nil {
		fsm.messageHandler.conn = sock
		fsm.IsProactive = true
	}
	fsm.neighborConfig = nConfig
	fsm.globalConfig = gConfig

}

func (fsm *BGPFiniteStateMachine) handleIdle() err {

	//initialize connection
	fsm.connectRetryCount = 0
	fsm.connectRetryTimer = time.NewTimer(BGP_CONNECT_RETRY_DURATION)
	fsm.previousState= BGP_FSM_IDLE
	return nil
}

func handleError(err error){
	if err != nil {
		log.Fatal(err)
	}
}

func (fsm *BGPFiniteStateMachine) handleConnect(bgpEvent <- chan BGPEvent) err {
	if fsm.state != BGP_FSM_IDLE || fsm.state != BGP_FSM_ACTIVE{
		return error.New("state is not idle or active.")
	}

	go func() {
		conn, err := net.DialTimeout("tcp",fsm.neighborConfig.PeerAddress.String(),60)
		if err != nil {
			//do something
			log.Fatalln(err)
			fsm.internalSocket = nil
		}
		fsm.internalSocket = conn
		bgpEvent <- BGP_EVENT_TCP_CONNECTION_CONFIRMED
	}()

	for {
		select {
		case evt <- bgpEvent:
			if evt == BGP_EVENT_TCP_CONNECTION_CONFIRMED {
				//stop ConectionRetryTimer
				fsm.connectRetryTimer.Stop()
				var openMsg *BGPOpen = new(BGPOpen)
				openMsg.MyAS = fsm.globalConfig.MyAS
				openMsg.ID = fsm.globalConfig.ID
				openMsg.Version = 4
				openMsg.HoldTime = fsm.globalConfig.HoldTime
				data, err:= openMsg.Encode()
				handleError(err)

				//stop connectRetryTimer
				fsm.connectRetryTimer.Stop()
				err = fsm.internalSocket.Write(data)
				if err != nil {
					return err
				}

				fsm.holdTimer = time.NewTimer(time.Minute * 4)
				break
			}
		case <- fsm.connectRetryTimer.C:
			// go to Active state

		}
	}
	return nil
}

func (fsm *BGPFiniteStateMachine) handleOpenSent(bgpEvent <- chan BGPEvent) error {
	if fsm.previousState != BGP_FSM_CONNECT || fsm.previousState != BGP_FSM_ACTIVE {
		return error.New("previous state is not valid.")
	}
	recvCh := make(chan struct{})

	go func() {
		sock := fsm.internalSocket
		buf := make([]byte, 4096)
		read_len, err := sock.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(read_len)
		var openMsg *BGPOpen = new(BGPOpen)
		openMsg.DecodeFromBytes(buf)
		fsm.receivedOpenMsg = openMsg
		recvCh <- struct{}
	}()

	for {
		select {
		case evt := <- bgpEvent:
			if evt == BGP_EVENT_MANUAL_STOP {
				//go to IdleState
				//
			}
		case <-recvCh:
			holdtime := fsm.receivedOpenMsg.HoldTime
			if holdtime < fsm.globalConfig.HoldTime {
				holdtime = fsm.globalConfig.HoldTime
			}
			// send KeepAlive message
			var keepAlive BGPMessage
			keepAlive.Header.Type = BGP_MSG_KEEPALIVE
			keepAlive.Header.Len = 19 // MSG_MIN
			data := keepAlive.Encode()
			fsm.internalSocket.Write(data)

			// set connectretrytimer
			// set holdtimer
			// set keepalivetimer

			fsm.previousState = fsm.state
			fsm.state = BGP_FSM_OPENCONFIRM
			break
		}
	}
	return nil
}

func (fsm *BGPFiniteStateMachine) handleOpenConfirm(bgpEvent <- chan BGPEvent) err {
	if fsm.previousState != BGP_FSM_OPENSENT {
		return error.New("previous state is not valid.")
	}

	recvCh := make(chan BGPMessage)

	go func() {
		sock := fsm.internalSocket
		buf := make([]byte, 4096)
		read_len, err := sock.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(read_len)
		openMsg ,err := ParseBGPMessage(buf)
		recvCh <- openMsg
	}()

	for {
		select {
		case evt <- bgpEvent:
			if evt == BGP_EVENT_MANUAL_STOP {
				//go to IdleState
				//
			}
		case msg := <-recvCh:

			fsm.previousState = fsm.state
			fsm.state = BGP_FSM_ESTABLISHED
			break
		}
	}
	return nil
}


