/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package client

import (
	"fmt"
	"math/big"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// NewClient returns a new instance of Client
func NewClient(ip string, port int) *Client {
	return &Client{
		http: newHTTPClient(
			fmt.Sprintf("http://%s:%d", ip, port),
			"goclient-v0.1",
		),
	}
}

// Client is able perform remote requests to Tequilapi server
type Client struct {
	http httpClientInterface
}

// AuthAuthenticate authenticates user and issues auth token
func (client *Client) AuthAuthenticate(request contract.AuthRequest) (res contract.AuthResponse, err error) {
	response, err := client.http.Post("/auth/authenticate", request)
	if err != nil {
		return res, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	if err != nil {
		return res, err
	}

	client.http.SetToken(res.Token)
	return res, nil
}

// AuthLogin authenticates user and sets cookie with issued auth token
func (client *Client) AuthLogin(request contract.AuthRequest) (res contract.AuthResponse, err error) {
	response, err := client.http.Post("/auth/login", request)
	if err != nil {
		return res, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	if err != nil {
		return res, err
	}

	client.http.SetToken(res.Token)
	return res, nil
}

// AuthLogout Clears authentication cookie
func (client *Client) AuthLogout() error {
	response, err := client.http.Delete("/auth/logout", nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// AuthChangePassword changes user password
func (client *Client) AuthChangePassword(request contract.ChangePasswordRequest) error {
	response, err := client.http.Put("/auth/password", request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// GetIdentities returns a list of client identities
func (client *Client) GetIdentities() (ids []contract.IdentityRefDTO, err error) {
	response, err := client.http.Get("identities", url.Values{})
	if err != nil {
		return
	}
	defer response.Body.Close()

	var list contract.ListIdentitiesResponse
	err = parseResponseJSON(response, &list)

	return list.Identities, err
}

// NewIdentity creates a new client identity
func (client *Client) NewIdentity(passphrase string) (id contract.IdentityRefDTO, err error) {
	response, err := client.http.Post("identities", contract.IdentityCreateRequest{Passphrase: &passphrase})
	if err != nil {
		return
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &id)
	return id, err
}

// CurrentIdentity unlocks and returns the last used, new or first identity
func (client *Client) CurrentIdentity(identity, passphrase string) (id contract.IdentityRefDTO, err error) {
	response, err := client.http.Put("identities/current", contract.IdentityCurrentRequest{
		Address:    &identity,
		Passphrase: &passphrase,
	})
	if err != nil {
		return
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &id)
	return id, err
}

// Identity returns identity status with current balance
func (client *Client) Identity(identityAddress string) (id contract.IdentityDTO, err error) {
	path := fmt.Sprintf("identities/%s", identityAddress)

	response, err := client.http.Get(path, nil)
	if err != nil {
		return id, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &id)
	return id, err
}

// IdentityRegistrationStatus returns information of identity needed to register it on blockchain
func (client *Client) IdentityRegistrationStatus(address string) (contract.IdentityRegistrationResponse, error) {
	response, err := client.http.Get("identities/"+address+"/registration", url.Values{})
	if err != nil {
		return contract.IdentityRegistrationResponse{}, err
	}
	defer response.Body.Close()

	status := contract.IdentityRegistrationResponse{}
	err = parseResponseJSON(response, &status)
	return status, err
}

// GetTransactorFees returns the transactor fees
func (client *Client) GetTransactorFees() (contract.FeesDTO, error) {
	fees := contract.FeesDTO{}

	res, err := client.http.Get("transactor/fees", nil)
	if err != nil {
		return fees, err
	}
	defer res.Body.Close()

	err = parseResponseJSON(res, &fees)
	return fees, err
}

// RegisterIdentity registers identity
func (client *Client) RegisterIdentity(address, beneficiary string, stake, fee *big.Int, token *string) error {
	payload := contract.IdentityRegisterRequest{
		Stake:         stake,
		Fee:           fee,
		Beneficiary:   beneficiary,
		ReferralToken: token,
	}

	response, err := client.http.Post("identities/"+address+"/register", payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("expected 202 got %v", response.StatusCode)
	}

	return nil
}

// ConnectionCreate initiates a new connection to a host identified by providerID
func (client *Client) ConnectionCreate(consumerID, providerID, hermesID, serviceType string, options contract.ConnectOptions) (status contract.ConnectionInfoDTO, err error) {
	response, err := client.http.Put("connection", contract.ConnectionCreateRequest{
		ConsumerID:     consumerID,
		ProviderID:     providerID,
		HermesID:       hermesID,
		ServiceType:    serviceType,
		ConnectOptions: options,
	})
	if err != nil {
		return contract.ConnectionInfoDTO{}, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// ConnectionDestroy terminates current connection
func (client *Client) ConnectionDestroy() (err error) {
	response, err := client.http.Delete("connection", nil)
	if err != nil {
		return
	}
	defer response.Body.Close()

	return nil
}

// ConnectionStatistics returns statistics about current connection
func (client *Client) ConnectionStatistics() (statistics contract.ConnectionStatisticsDTO, err error) {
	response, err := client.http.Get("connection/statistics", url.Values{})
	if err != nil {
		return statistics, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &statistics)
	return statistics, err
}

// ConnectionStatus returns connection status
func (client *Client) ConnectionStatus() (status contract.ConnectionInfoDTO, err error) {
	response, err := client.http.Get("connection", url.Values{})
	if err != nil {
		return status, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// ConnectionIP returns public ip
func (client *Client) ConnectionIP() (ip contract.IPDTO, err error) {
	response, err := client.http.Get("connection/ip", url.Values{})
	if err != nil {
		return ip, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &ip)
	return ip, err
}

// ConnectionLocation returns current location
func (client *Client) ConnectionLocation() (location contract.LocationDTO, err error) {
	response, err := client.http.Get("connection/location", url.Values{})
	if err != nil {
		return location, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &location)
	return location, err
}

// Healthcheck returns a healthcheck info
func (client *Client) Healthcheck() (healthcheck contract.HealthCheckDTO, err error) {
	response, err := client.http.Get("healthcheck", url.Values{})
	if err != nil {
		return
	}

	defer response.Body.Close()
	err = parseResponseJSON(response, &healthcheck)
	return healthcheck, err
}

// OriginLocation returns original location
func (client *Client) OriginLocation() (location contract.LocationDTO, err error) {
	response, err := client.http.Get("location", url.Values{})
	if err != nil {
		return location, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &location)
	return location, err
}

// ProposalsByType fetches proposals by given type
func (client *Client) ProposalsByType(serviceType string) ([]contract.ProposalDTO, error) {
	queryParams := url.Values{}
	queryParams.Add("service_type", serviceType)
	return client.proposals(queryParams)
}

// Proposals returns all available proposals for services
func (client *Client) Proposals() ([]contract.ProposalDTO, error) {
	return client.proposals(url.Values{})
}

func (client *Client) proposals(query url.Values) ([]contract.ProposalDTO, error) {
	response, err := client.http.Get("proposals", query)
	if err != nil {
		return []contract.ProposalDTO{}, err
	}
	defer response.Body.Close()

	var proposals contract.ListProposalsResponse
	err = parseResponseJSON(response, &proposals)
	return proposals.Proposals, err
}

// ProposalsByPrice returns all available proposals within the given price range
func (client *Client) ProposalsByPrice(lowerTime, upperTime, lowerGB, upperGB *big.Int) ([]contract.ProposalDTO, error) {
	values := url.Values{}
	values.Add("upper_time_price_bound", fmt.Sprintf("%v", upperTime))
	values.Add("lower_time_price_bound", fmt.Sprintf("%v", lowerTime))
	values.Add("upper_gb_price_bound", fmt.Sprintf("%v", upperGB))
	values.Add("lower_gb_price_bound", fmt.Sprintf("%v", lowerGB))
	return client.proposals(values)
}

// Unlock allows using identity in following commands
func (client *Client) Unlock(identity, passphrase string) error {
	path := fmt.Sprintf("identities/%s/unlock", identity)

	response, err := client.http.Put(path, contract.IdentityUnlockRequest{Passphrase: &passphrase})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// Payout registers payout address for identity
func (client *Client) Payout(identity, ethAddress string) error {
	path := fmt.Sprintf("identities/%s/payout", identity)
	payload := struct {
		EthAddress string `json:"eth_address"`
	}{
		ethAddress,
	}

	response, err := client.http.Put(path, payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// Stop kills mysterium client
func (client *Client) Stop() error {
	emptyPayload := struct{}{}
	response, err := client.http.Post("/stop", emptyPayload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// Sessions returns all sessions from history
func (client *Client) Sessions() (sessions contract.SessionListResponse, err error) {
	response, err := client.http.Get("sessions", url.Values{})
	if err != nil {
		return sessions, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &sessions)
	return sessions, err
}

// SessionsByServiceType returns sessions from history filtered by type
func (client *Client) SessionsByServiceType(serviceType string) (contract.SessionListResponse, error) {
	sessions, err := client.Sessions()
	sessions = filterSessionsByType(serviceType, sessions)
	return sessions, err
}

// SessionsByStatus returns sessions from history filtered by their status
func (client *Client) SessionsByStatus(status string) (contract.SessionListResponse, error) {
	sessions, err := client.Sessions()
	sessions = filterSessionsByStatus(status, sessions)
	return sessions, err
}

// Services returns all running services
func (client *Client) Services() (services contract.ServiceListResponse, err error) {
	response, err := client.http.Get("services", url.Values{})
	if err != nil {
		return services, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &services)
	return services, err
}

// Service returns a service information by the requested id
func (client *Client) Service(id string) (service contract.ServiceInfoDTO, err error) {
	response, err := client.http.Get("services/"+id, url.Values{})
	if err != nil {
		return service, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &service)
	return service, err
}

// ServiceStart starts an instance of the service.
func (client *Client) ServiceStart(request contract.ServiceStartRequest) (service contract.ServiceInfoDTO, err error) {
	response, err := client.http.Post("services", request)
	if err != nil {
		return service, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &service)
	return service, err
}

// ServiceStop stops the running service instance by the requested id.
func (client *Client) ServiceStop(id string) error {
	path := fmt.Sprintf("services/%s", id)
	response, err := client.http.Delete(path, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// NATStatus returns status of NAT traversal
func (client *Client) NATStatus() (status contract.NATStatusDTO, err error) {
	response, err := client.http.Get("nat/status", nil)
	if err != nil {
		return status, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &status)
	return status, err
}

// filterSessionsByType removes all sessions of irrelevant types
func filterSessionsByType(serviceType string, sessions contract.SessionListResponse) contract.SessionListResponse {
	matches := 0
	for _, s := range sessions.Items {
		if s.ServiceType == serviceType {
			sessions.Items[matches] = s
			matches++
		}
	}
	sessions.Items = sessions.Items[:matches]
	return sessions
}

// filterSessionsByStatus removes all sessions with non matching status
func filterSessionsByStatus(status string, sessions contract.SessionListResponse) contract.SessionListResponse {
	matches := 0
	for _, s := range sessions.Items {
		if s.Status == status {
			sessions.Items[matches] = s
			matches++
		}
	}
	sessions.Items = sessions.Items[:matches]
	return sessions
}

// Settle requests the settling of hermes promises
func (client *Client) Settle(providerID, hermesID identity.Identity, waitForBlockchain bool) error {
	settleRequest := contract.SettleRequest{
		ProviderID: providerID.Address,
		HermesID:   hermesID.Address,
	}

	path := "transactor/settle/"
	if waitForBlockchain {
		path += "sync"
	} else {
		path += "async"
	}

	response, err := client.http.Post(path, settleRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not settle promise")
	}
	return nil
}

// SettleIntoStake requests the settling of accountant promises into a stake increase
func (client *Client) SettleIntoStake(providerID, hermesID identity.Identity, waitForBlockchain bool) error {
	settleRequest := contract.SettleRequest{
		ProviderID: providerID.Address,
		HermesID:   hermesID.Address,
	}

	path := "transactor/stake/increase/"
	if waitForBlockchain {
		path += "sync"
	} else {
		path += "async"
	}

	response, err := client.http.Post(path, settleRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not settle promise")
	}
	return nil
}

// DecreaseStake requests the decrease of stake via the transactor.
func (client *Client) DecreaseStake(ID identity.Identity, amount, transactorFee *big.Int) error {
	decreaseRequest := contract.DecreaseStakeRequest{
		ID:            ID.Address,
		Amount:        amount,
		TransactorFee: transactorFee,
	}

	path := "transactor/stake/decrease"

	response, err := client.http.Post(path, decreaseRequest)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return errors.Wrap(err, "could not decrease stake")
	}
	return nil
}

// SettleWithBeneficiary set new beneficiary address for the provided identity.
func (client *Client) SettleWithBeneficiary(address, beneficiary, hermesID string) error {
	payload := contract.SettleWithBeneficiaryRequest{
		SettleRequest: contract.SettleRequest{
			ProviderID: address,
			HermesID:   hermesID,
		},
		Beneficiary: beneficiary,
	}
	response, err := client.http.Post("identities/"+address+"/beneficiary", payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("expected 202 got %v", response.StatusCode)
	}

	return nil
}

// Beneficiary gets beneficiary address for the provided identity.
func (client *Client) Beneficiary(address string) (res contract.IdentityBeneficiaryResponse, err error) {
	response, err := client.http.Get("identities/"+address+"/beneficiary", nil)
	if err != nil {
		return contract.IdentityBeneficiaryResponse{}, err
	}
	defer response.Body.Close()

	err = parseResponseJSON(response, &res)
	return res, err
}

// SetMMNApiKey sets MMN's API key in config and registers node to MMN
func (client *Client) SetMMNApiKey(data contract.MMNApiKeyRequest) error {
	response, err := client.http.Post("mmn/api-key", data)

	// non 200 status codes return a generic error and we can't use it, instead
	// the response contains validation JSON which we can use to extract the error
	if err != nil && response == nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode == 200 {
		return nil
	}

	// TODO this should probably be wrapped and moved into the validation package
	type validationResponse struct {
		Message string                              `json:"message"`
		Errors  map[string][]*validation.FieldError `json:"errors"`
	}
	res := validationResponse{}
	err = parseResponseJSON(response, &res)
	if err != nil {
		return err
	}

	if res.Errors != nil && res.Errors["api_key"] != nil && res.Errors["api_key"][0] != nil {
		return errors.New((res.Errors["api_key"][0]).Message)
	}

	return nil
}

// IdentityReferralCode returns a referral token for the given identity.
func (client *Client) IdentityReferralCode(identity string) (contract.ReferralTokenResponse, error) {
	response, err := client.http.Get(fmt.Sprintf("identities/%v/referral", identity), nil)
	if err != nil {
		return contract.ReferralTokenResponse{}, err
	}
	defer response.Body.Close()

	res := contract.ReferralTokenResponse{}
	err = parseResponseJSON(response, &res)
	return res, err
}

// OrderCreate creates a new order for currency exchange in pilvytis
func (client *Client) OrderCreate(identity string, order contract.OrderRequest) (contract.OrderResponse, error) {
	resp, err := client.http.Post(fmt.Sprintf("identity/%s/pilvytis/order", identity), order)
	if err != nil {
		return contract.OrderResponse{}, err
	}
	defer resp.Body.Close()

	var res contract.OrderResponse
	return res, parseResponseJSON(resp, &res)
}

// OrderGet returns a single order istance given it's ID.
func (client *Client) OrderGet(identity string, id uint64) (contract.OrderResponse, error) {
	path := fmt.Sprintf("identity/%s/pilvytis/order/%d", identity, id)
	resp, err := client.http.Get(path, nil)
	if err != nil {
		return contract.OrderResponse{}, err
	}
	defer resp.Body.Close()

	var res contract.OrderResponse
	return res, parseResponseJSON(resp, &res)
}

// OrderGetAll returns all order istances for a given identity
func (client *Client) OrderGetAll(identity string) ([]contract.OrderResponse, error) {
	path := fmt.Sprintf("identity/%s/pilvytis/order", identity)
	resp, err := client.http.Get(path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res []contract.OrderResponse
	return res, parseResponseJSON(resp, &res)
}
