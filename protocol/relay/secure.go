package relay

import "whitenoise/secure"

func (manager *RelayMsgManager) NewSecureConnCaller(conn *CircuitConn) error {
	if _, ok := manager.secureConnMap[conn.sessionId]; ok {
		return nil
	}
	secureConn, err := secure.NewSecureSession(manager.host.ID(), manager.privateKey, conn.ctx, conn, conn.remotePeerId, true)
	if err != nil {
		return err
	}
	manager.secureConnMap[conn.sessionId] = secureConn
	return nil
}

func (manager *RelayMsgManager) NewSecureConnAnswer(conn *CircuitConn) error {
	if _, ok := manager.secureConnMap[conn.sessionId]; ok {
		return nil
	}
	secureConn, err := secure.NewSecureSession(manager.host.ID(), manager.privateKey, conn.ctx, conn, "", false)
	if err != nil {
		return err
	}
	manager.secureConnMap[conn.sessionId] = secureConn
	return nil
}
