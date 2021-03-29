package common

import (
	"time"
)

type SessionRole int

const (
	CallerRole SessionRole = 1
	EntryRole  SessionRole = 2
	JointRole  SessionRole = 3
	RelayRole  SessionRole = 4
	ExitRole   SessionRole = 5
	AnswerRole SessionRole = 6
)

const (
	SetSessionTimeout       time.Duration = time.Second
	ExpendSessionTimeout    time.Duration = time.Second * 3
	RegisterProxyTimeout    time.Duration = time.Second
	NewCircuitTimeout       time.Duration = time.Second * 5
	DecryptReqTimeout       time.Duration = time.Millisecond * 500
	ReadHandShakeMsgTimeout time.Duration = time.Second
)

const (
	CircuitConnReadPollCycle   time.Duration = time.Millisecond * 10
	CircuitConnReadPollTimeout time.Duration = time.Second * 10
)

const RetryTimes = 3

const RequestFutureDuration time.Duration = time.Second

const (
	NewSecureConnCallerTopic string = "topic:NewCaller"
	NewSecureConnAnswerTopic string = "topic:NewAnswer"
)

const BootstrapDuration = time.Hour
const GetMainnetPeersTimeout = time.Second * 5

const ReqDHTPeersMaxAmount = 100

const UnreadableTimeout = time.Minute * 5