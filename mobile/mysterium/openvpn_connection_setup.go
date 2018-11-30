/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package mysterium

import (
	"errors"
	"sync"

	"github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-openvpn/openvpn3"
	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/openvpn/session"
)

type sessionWrapper struct {
	session *openvpn3.Session
}

func (wrapper *sessionWrapper) Start() error {

	wrapper.session.Start()
	return nil
}

func (wrapper *sessionWrapper) Stop() {
	wrapper.session.Stop()
}

func (wrapper *sessionWrapper) Wait() error {
	return wrapper.session.Wait()
}

func channelToCallbacks(stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) openvpn3.MobileSessionCallbacks {
	return channelToCallbacksAdapter{
		stateChannel:      stateChannel,
		statisticsChannel: statisticsChannel,
	}
}

type channelToCallbacksAdapter struct {
	stateChannel      connection.StateChannel
	statisticsChannel connection.StatisticsChannel
}

func (adapter channelToCallbacksAdapter) OnEvent(event openvpn3.Event) {
	switch event.Name {
	case "CONNECTING":
		adapter.stateChannel <- connection.Connecting
	case "CONNECTED":
		adapter.stateChannel <- connection.Connected
	case "DISCONNECTED":
		adapter.stateChannel <- connection.NotConnected
		close(adapter.stateChannel)
	default:
		seelog.Infof("Unhandled event: %+v", event)
	}
}

func (channelToCallbacksAdapter) Log(text string) {
	seelog.Infof("Log: %+v", text)
}

func (adapter channelToCallbacksAdapter) OnStats(openvpnStats openvpn3.Statistics) {
	adapter.statisticsChannel <- consumer.SessionStatistics{
		BytesSent:     uint64(openvpnStats.BytesOut),
		BytesReceived: uint64(openvpnStats.BytesIn),
	}
}

// Openvpn3TunnelSetup is alias for openvpn3 tunnel setup interface exposed to Android/iOS interop
type Openvpn3TunnelSetup openvpn3.TunnelSetup

// ReconnectableSession interface exposing reconnect for an active session
type ReconnectableSession interface {
	Reconnect(afterSeconds int) error
}

type sessionTracker struct {
	session *openvpn3.Session
	mux     sync.Mutex
}

func (st *sessionTracker) sessionCreated(s *openvpn3.Session) {
	st.mux.Lock()
	st.session = s
	st.mux.Unlock()
}

// Reconnect reconnects active session after given time
func (st *sessionTracker) Reconnect(afterSeconds int) error {
	if st.session == nil {
		return errors.New("session not created yet")
	}

	return st.session.Reconnect(afterSeconds)
}

func (st *sessionTracker) handleState(stateEvent connection.StateEvent) {
	// On disconnected - remove session
	if stateEvent.State == connection.Disconnecting {
		st.session = nil
	}
}

// OverrideOpenvpnConnection replaces default openvpn connection factory with mobile related one returning session that can be reconnected
func (mobNode *MobileNode) OverrideOpenvpnConnection(tunnelSetup Openvpn3TunnelSetup) ReconnectableSession {

	st := &sessionTracker{}

	mobNode.di.EventBus.Subscribe(connection.StateEventTopic, st.handleState)

	mobNode.di.ConnectionRegistry.Register("openvpn", func(options connection.ConnectOptions, stateChannel connection.StateChannel, statisticsChannel connection.StatisticsChannel) (connection.Connection, error) {

		vpnClientConfig, err := openvpn.NewClientConfigFromSession(options.SessionConfig, "", "")
		if err != nil {
			return nil, err
		}

		profileContent, err := vpnClientConfig.ToConfigFileContent()
		if err != nil {
			return nil, err
		}

		config := openvpn3.NewConfig(profileContent)
		config.GuiVersion = "govpn 0.1"
		config.CompressionMode = "asym"
		config.ConnTimeout = 0 //essentially means - reconnect forever (but can always stopped with disconnect)

		signer := mobNode.di.SignerFactory(options.ConsumerID)

		username, password, err := session.SignatureCredentialsProvider(options.SessionID, signer)()
		if err != nil {
			return nil, err
		}

		credentials := openvpn3.UserCredentials{
			Username: username,
			Password: password,
		}

		session := openvpn3.NewMobileSession(config, credentials, channelToCallbacks(stateChannel, statisticsChannel), tunnelSetup)
		st.sessionCreated(session)

		return &sessionWrapper{
			session: session,
		}, nil
	})
	return st
}
