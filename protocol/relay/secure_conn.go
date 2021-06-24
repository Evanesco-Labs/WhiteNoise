package relay

import (
	"github.com/Evanesco-Labs/WhiteNoise/common"
	"github.com/Evanesco-Labs/WhiteNoise/secure"
)

func (manager *RelayMsgManager) NewSecureConnCaller(conn *CircuitConn) error {
	if _, ok := manager.secureConnMap.Load(conn.sessionId); ok {
		return nil
	}
	//if conn.remoteWhiteNoiseId == nil {
	//	log.Error("remote WhiteNoiseId nil")
	//}
	remotePeerID, err := conn.remoteWhiteNoiseId.GetPeerID()
	if err != nil {
		return err
	}
	secureConn, err := secure.NewSecureSession(manager.host.ID(), manager.privateKey, conn.ctx, conn, remotePeerID, true)
	if err != nil {
		return err
	}
	manager.secureConnMap.Store(conn.sessionId, secureConn)
	manager.eb.Publish(common.NewSecureConnCallerTopic, conn.sessionId)
	return nil
}

func (manager *RelayMsgManager) NewSecureConnAnswer(conn *CircuitConn) error {
	if _, ok := manager.secureConnMap.Load(conn.sessionId); ok {
		return nil
	}
	secureConn, err := secure.NewSecureSession(manager.host.ID(), manager.privateKey, conn.ctx, conn, "", false)
	if err != nil {
		return err
	}
	manager.secureConnMap.Store(conn.sessionId, secureConn)
	manager.eb.Publish(common.NewSecureConnAnswerTopic, conn.sessionId)
	return nil
}
