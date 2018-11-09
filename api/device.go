package api

import (
	"github.com/pkg/errors"
)

type DeviceExtra struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type Device struct {
	DeviceId         int           `json:"deviceId"`
	Name             string           `json:"name"`
	CustomerId       int           `json:"customerId"`
	PrimaryIP        string           `json:"primaryIP"`
	Status           string           `json:"status"`
	Extra            []DeviceExtra    `json:"extra"`
	DataCenterId     int           `json:"dcid"`
	FirewallPolicies []FirewallPolicy `json:"firewallPolicies`
	StdName          string           `json:"stdName"`
	DateCreated      int64            `json:"dateCreated"`
	DateModified     int64            `json:"dateModified`
	TrafficShared    bool             `json:"trafficShared,omitempty`
	TrafficCurrent   string           `json:"trafficCurrent`
	TrafficEstimated int              `json:"trafficEstimated"`
	TrafficMB        int              `json:"trafficMB"`
	DCName           string           `json:"dcName"`
	// TODO assets, dcDescription, ips, networkRoutes, ports,
}

type deviceGetRequest struct {
	request
	DeviceId int `json:"deviceId"`
}

type deviceGetResponse struct {
	response
	Device *Device `json:"device"`
}

type deviceUpdateRequest struct {
	request
	DeviceId         int           `json:"deviceId"`
	FirewallPolicies []FirewallPolicy `json:"firewallPolicies"`
}

type deviceUpdateResponse struct {
	response
}

func (c *client) DeviceGet(deviceId int) (*Device, error) {
	if deviceId == 0 {
		return nil, errors.New("deviceId parameter is required.")
	}

	req := &deviceGetRequest{
		request:  c.newRequest("device.get"),
		DeviceId: deviceId,
	}

	var resp deviceGetResponse
	err := c.httpPostJson(req, &resp)
	if err != nil {
		return nil, errors.Wrapf(err, "DeviceGet request failed for device Id '%d'.", deviceId)
	}

	if resp.Code != "OK" || resp.Device == nil {
		return nil, newApiError(resp.response, nil)
	}

	return resp.Device, nil
}

//  Note that if you want to delete an existing policy, you need to have it's policy set to DELETED
// (instead of ALLOW/REJECT/DISABLED) in the firewallPolicies array
func (c *client) DeviceUpdateFirewall(deviceId int, firewallPolicies []FirewallPolicy) error {
	if deviceId == 0 {
		return errors.New("deviceId parameter is required")
	}
	if firewallPolicies == nil || len(firewallPolicies) == 0 {
		return errors.New("must update with Firewall Policies")
	}

	req := &deviceUpdateRequest{
		request:          c.newRequest("device.firewall.update"),
		DeviceId:         deviceId,
		FirewallPolicies: firewallPolicies,
	}

	var resp deviceUpdateResponse
	err := c.httpPostJson(req, &resp)
	if err != nil {
		return errors.Wrapf(err, "UpdateFirewall request failed for device Id '%d'.", deviceId)
	}

	if resp.Code != "OK" {
		return newApiError(resp.response, nil)
	}

	return nil
}
