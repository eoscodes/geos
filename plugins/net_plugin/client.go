package net_plugin

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/eosspark/eos-go/chain/types"
	//"github.com/eosspark/eos-go/chainp2p"
	"github.com/eosspark/eos-go/common"
	"github.com/eosspark/eos-go/crypto"
	"github.com/eosspark/eos-go/crypto/ecc"
	"github.com/eosspark/eos-go/crypto/rlp"
	"github.com/eosspark/eos-go/log"
	"math"
	"net"
	"runtime"
	"time"
)

type Client struct {
	p2pAddress            string
	ChainID               common.ChainIdType
	NetWorkVersion        uint16
	Conn                  net.Conn
	NodeID                common.NodeIdType
	SigningKey            ecc.PrivateKey
	AgentName             string
	LastHandshakeReceived *HandshakeMessage
}

func NewClient(p2pAddr string, chainID common.ChainIdType, networkVersion uint16) *Client {
	nodeID := make([]byte, 32)
	rand.Read(nodeID)
	data := *crypto.NewSha256Byte(nodeID)

	return &Client{
		p2pAddress:     p2pAddr,
		ChainID:        chainID,
		NetWorkVersion: networkVersion,
		AgentName:      "EOS Test Agent",
		NodeID:         common.NodeIdType(data),
	}
}

func (c *Client) StartConnect() error {
	return c.connect(0, 0)
}

func (c *Client) connect(headBlock uint32, lib uint32) (err error) {
	conn, err := net.Dial("tcp", c.p2pAddress)
	if err != nil {
		return err
	}
	c.Conn = conn
	fmt.Println("connecting to: ", c.p2pAddress)
	ready := make(chan bool)
	errChannel := make(chan error)

	signedBlock := make(chan types.SignedBlock)
	packedTransaction := make(chan types.PackedTransaction)
	go c.handleConnection(ready, errChannel, signedBlock, packedTransaction)

	/*go chainp2p.GetSignedBlock(signedBlock)
	go chainp2p.GetPackedTrx(packedTransaction)
	<-ready*/

	//fmt.Println(c.p2pAddress, " Connected")

	err = c.SendHandshake(&HandshakeInfo{
		HeadBlockNum:             headBlock,
		LastIrreversibleBlockNum: lib,
		Generation:               1,
	})
	if err != nil {
		return err
	}
	return <-errChannel
}

type HandshakeInfo struct {
	HeadBlockNum             uint32
	HeadBlockID              common.BlockIdType
	LastIrreversibleBlockNum uint32
	LastIrreversibleBlockID  common.BlockIdType
	Generation               uint16
}

func (c *Client) SendHandshake(info *HandshakeInfo) (err error) {
	publickey, err := ecc.NewPublicKey("EOS1111111111111111111111111111111114T1Anm")
	if err != nil {
		fmt.Println("publickey:", err)
		return err
	}

	signature := ecc.Signature{
		Curve:   ecc.CurveK1,
		Content: [65]byte{},
	}

	handshake := &HandshakeMessage{
		NetworkVersion:           c.NetWorkVersion,
		ChainID:                  c.ChainID,
		NodeID:                   c.NodeID,
		Key:                      publickey,
		Time:                     common.Now(),
		Token:                    *crypto.NewSha256Nil(),
		Signature:                signature,
		P2PAddress:               c.p2pAddress,
		LastIrreversibleBlockNum: info.LastIrreversibleBlockNum,
		LastIrreversibleBlockID:  info.LastIrreversibleBlockID,
		HeadNum:                  info.HeadBlockNum,
		HeadID:                   info.HeadBlockID,
		OS:                       runtime.GOOS,
		Agent:                    c.AgentName,
		Generation:               info.Generation,
	}

	err = c.sendMessage(handshake)
	if err != nil {
		fmt.Println("send HandshakeMessage, ", err)
	}
	return
}

func (c *Client) SendSyncRequest(startBlocknum uint32, endBlocknum uint32) (err error) {
	fmt.Printf("Requestion block from %d to %d \n", startBlocknum, endBlocknum)
	syncRequest := &SyncRequestMessage{
		StartBlock: startBlocknum,
		EndBlock:   endBlocknum,
	}
	return c.sendMessage(syncRequest)

}

func (c *Client) sendMessage(message P2PMessage) (err error) {
	payload, err := rlp.EncodeToBytes(message)
	if err != nil {
		err = fmt.Errorf("p2p message, %s", err)
		return
	}
	messageLen := uint32(len(payload) + 1)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, messageLen)
	sendBuf := append(buf, byte(message.GetType()))
	sendBuf = append(sendBuf, payload...)

	c.Conn.Write(sendBuf)

	//fmt.Println("已发送Message", sendBuf)
	log.Info("sended Message: %v", sendBuf)
	data, err := json.Marshal(message)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("struct:  ", string(data))

	return
}

var (
	peerHeadBlock  = uint32(0)
	syncHeadBlock  = uint32(0)
	RequestedBlock = uint32(0)
	syncing        = false
	headBlock      = uint32(0)
)

func (c *Client) handleConnection(ready chan bool, errChannel chan error, signedBlock chan types.SignedBlock, packedTransaction chan types.PackedTransaction) {
	r := bufio.NewReader(c.Conn)
	ready <- true
	for {
		p2pMessage, err := ReadP2PMessageData(r)
		if err != nil {
			fmt.Println("Error reading from p2p client:", err)
			errChannel <- err
			//continue
			return
		}

		data, err := json.Marshal(p2pMessage)
		if err != nil {
			fmt.Println(err)
		}
		//fmt.Println("Receive P2PMessag ", string(data))
		log.Warn("receive p2pMessage: %s", string(data))

		//encResult, err := rlp.EncodeToBytes(p2pMessage)
		//fmt.Printf("encode result: %#v\n", encResult)

		switch msg := p2pMessage.(type) {
		case *HandshakeMessage:
			c.LastHandshakeReceived = msg
			hInfo := &HandshakeInfo{
				HeadBlockNum:             msg.HeadNum,
				HeadBlockID:              msg.HeadID,
				LastIrreversibleBlockNum: msg.LastIrreversibleBlockNum,
				LastIrreversibleBlockID:  msg.LastIrreversibleBlockID,
			}

			if msg.HeadNum > headBlock {
				syncHeadBlock = headBlock + 1
				peerHeadBlock = msg.HeadNum
				delta := peerHeadBlock - syncHeadBlock
				RequestedBlock = syncHeadBlock + uint32(math.Min(float64(delta), 250))
				syncing = true
				c.SendSyncRequest(syncHeadBlock, RequestedBlock)
				// return
			} else {
				fmt.Println("In sync ... Sending handshake!")
				// hInfo = &HandshakeInfo{
				// 	HeadBlockNum:             headBlock,
				// 	HeadBlockID:              headBlockID,
				// 	LastIrreversibleBlockNum: lib,
				// 	LastIrreversibleBlockID:  libID,
				// }
			}

			if err := c.SendHandshake(hInfo); err != nil {
				fmt.Println(err)
			}
		case *ChainSizeMessage:

		case *GoAwayMessage:
			fmt.Printf("GO AWAY Reason[%d] \n", msg.Reason)
		case *TimeMessage:
			msg.Dst = common.Now()
			data, err := json.Marshal(msg)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("time message: ", string(data))

		case *NoticeMessage:

		case *RequestMessage:

		case *SyncRequestMessage:

		case *SignedBlockMessage:

			syncHeadBlock = msg.BlockNumber()
			fmt.Printf("signed Block Num: %d\n", syncHeadBlock)

			signedBlock <- msg.SignedBlock

			if syncHeadBlock == RequestedBlock {

				delta := peerHeadBlock - syncHeadBlock
				if delta == 0 {

					syncing = false
					fmt.Println("Sync completed ... Sending handshake")
					id := msg.BlockID()
					if err != nil {
						fmt.Println("blockID: ", err)
						return
					}
					hInfo := &HandshakeInfo{
						HeadBlockNum:             msg.BlockNumber(),
						HeadBlockID:              id,
						LastIrreversibleBlockNum: c.LastHandshakeReceived.LastIrreversibleBlockNum,
						LastIrreversibleBlockID:  c.LastHandshakeReceived.LastIrreversibleBlockID,
						Generation:               2,
					}
					if err := c.SendHandshake(hInfo); err != nil {
						fmt.Println(err)
						return
					}
					return
				}

				RequestedBlock = syncHeadBlock + uint32(math.Min(float64(delta), 250))
				syncHeadBlock++
				fmt.Println("************************************")
				fmt.Printf("Requestion more block from %d to %d\n", syncHeadBlock, RequestedBlock)
				fmt.Println("************************************")
				c.SendSyncRequest(syncHeadBlock, RequestedBlock)
			}
		case *PackedTransactionMessage:
			packedTransaction <- msg.PackedTransaction
		default:
			fmt.Println("unsupport p2pmessage type")
		}

		time.Sleep(100 * time.Millisecond)

	}
}
