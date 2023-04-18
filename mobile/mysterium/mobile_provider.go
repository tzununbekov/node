//go:build mobile_provider

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
	"strings"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/services"
	"github.com/rs/zerolog/log"
)

// DefaultProviderNodeOptions returns default options.
func DefaultProviderNodeOptions() *MobileNodeOptions {
	options := DefaultNodeOptionsByNetwork(string(config.Mainnet))
	options.IsProvider = true
	options.TequilapiSecured = true
	return options
}

func (mb *MobileNode) unlockIdentity(adr, passphrase string) string {
	chainID := config.GetInt64(config.FlagChainID)
	id, err := mb.identitySelector.UseOrCreate(adr, passphrase, chainID)
	if err != nil {
		return ""
	}
	return id.Address
}

// StartProvider starts all provider services (provider mode)
func (mb *MobileNode) StartProvider() {
	providerID := mb.unlockIdentity(
		config.FlagIdentity.Value,
		config.FlagIdentityPassphrase.Value,
	)
	log.Info().Msgf("Unlocked identity: %v", providerID)

	activeServices := "wireguard,scraping,data_transfer"
	serviceTypes := strings.Split(activeServices, ",")

	for _, serviceType := range serviceTypes {
		serviceOpts, err := services.GetStartOptions(serviceType)
		if err != nil {
			log.Error().Err(err).Msg("GetStartOptions failed")
			return
		}

		_, err = mb.servicesManager.Start(identity.Identity{Address: providerID}, serviceType, serviceOpts.AccessPolicyList, serviceOpts.TypeOptions)
		if err != nil {
			log.Error().Err(err).Msg("servicesManager.Start failed")
			return
		}
	}
}

// StopProvider stops all provider services, started by StartProvider
func (mb *MobileNode) StopProvider() {
	for _, srv := range mb.servicesManager.List(true) {
		if srv.State() != servicestate.Running {
			continue
		}

		err := mb.servicesManager.Stop(srv.ID)
		if err != nil {
			log.Error().Err(err).Msg("servicesManager.Stop failed")
			return
		}
	}
}

// SetFlagLauncherVersion sets LauncherVersion flag value, which is reported to Prometheus
func SetFlagLauncherVersion(val string) {
	config.Current.SetDefault(config.FlagLauncherVersion.Name, val)
}