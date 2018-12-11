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

package session

import (
	"time"

	"github.com/mysteriumnetwork/node/consumer"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	node_session "github.com/mysteriumnetwork/node/session"
)

const (
	// SessionStatusNew means that newly created session object is written to storage
	SessionStatusNew = "New"
	// SessionStatusCompleted means that session object is updated on connection disconnect event
	SessionStatusCompleted = "Completed"
)

// NewHistory creates a new session history datapoint
func NewHistory(sessionID node_session.ID, proposal dto.ServiceProposal) *History {
	return &History{
		SessionID:       sessionID,
		ProviderID:      identity.FromAddress(proposal.ProviderID),
		ServiceType:     proposal.ServiceType,
		ProviderCountry: proposal.ServiceDefinition.GetLocation().Country,
		Started:         time.Now().UTC(),
		Status:          SessionStatusNew,
	}
}

// History holds structure for saving session history
type History struct {
	SessionID       node_session.ID `storm:"id"`
	ProviderID      identity.Identity
	ServiceType     string
	ProviderCountry string
	Started         time.Time
	Status          string
	Updated         time.Time
	DataStats       consumer.SessionStatistics // is updated on disconnect event
}

// GetDuration returns delta in seconds (TimeUpdated - TimeStarted)
func (se *History) GetDuration() uint64 {
	if se.Status == SessionStatusCompleted {
		return uint64(se.Updated.Sub(se.Started).Seconds())
	}
	return 0
}