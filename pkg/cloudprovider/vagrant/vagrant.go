/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vagrant_cloud

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	neturl "net/url"
	"sort"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/cloudprovider"
)

// VagrantCloud is an implementation of Interface, TCPLoadBalancer and Instances for developer managed Vagrant cluster
type VagrantCloud struct {
	saltURL  string
	saltUser string
	saltPass string
	saltAuth string
}

func init() {
	cloudprovider.RegisterCloudProvider("vagrant", func() (cloudprovider.Interface, error) { return newVagrantCloud() })
}

// SaltToken is an authorization token required by Salt REST API
type SaltToken struct {
	Token string `json:"token"`
	User  string `json:"user"`
	EAuth string `json:"eauth"`
}

// SaltLoginResponse is the response object for a /login operation against Salt REST API
type SaltLoginResponse struct {
	Data []SaltToken `json:"return"`
}

// SaltMinion is a machine managed by the Salt service
type SaltMinion struct {
	Roles []string `json:"roles"`
	IP    string   `json:"minion_ip"`
	Host  string   `json:"host"`
}

// SaltMinions is a map of minion name to machine information
type SaltMinions map[string]SaltMinion

// SaltMinionsResponse is the response object for a /minions operation against Salt REST API
type SaltMinionsResponse struct {
	Minions []SaltMinions `json:"return"`
}

// newVagrantCloud creates a new instance of VagrantCloud configured to talk to the Salt REST API.
func newVagrantCloud() (*VagrantCloud, error) {
	return &VagrantCloud{
		saltURL:  "http://127.0.0.1:8000",
		saltUser: "vagrant",
		saltPass: "vagrant",
		saltAuth: "pam",
	}, nil
}

// TCPLoadBalancer returns an implementation of TCPLoadBalancer for Vagrant cloud
func (v *VagrantCloud) TCPLoadBalancer() (cloudprovider.TCPLoadBalancer, bool) {
	return nil, false
}

// Instances returns an implementation of Instances for Vagrant cloud
func (v *VagrantCloud) Instances() (cloudprovider.Instances, bool) {
	return v, true
}

// Zones returns an implementation of Zones for Vagrant cloud
func (v *VagrantCloud) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

// IPAddress returns the address of a particular machine instance
func (v *VagrantCloud) IPAddress(instance string) (net.IP, error) {
	// since the instance now is the IP in the vagrant env, this is trivial no-op
	return net.ParseIP(instance), nil
}

// saltMinionsByRole filters a list of minions that have a matching role
func (v *VagrantCloud) saltMinionsByRole(minions []SaltMinion, role string) []SaltMinion {
	var filteredMinions []SaltMinion
	for _, value := range minions {
		sort.Strings(value.Roles)
		if pos := sort.SearchStrings(value.Roles, role); pos < len(value.Roles) {
			filteredMinions = append(filteredMinions, value)
		}
	}
	return filteredMinions
}

// saltMinions invokes the Salt API for minions using provided token
func (v *VagrantCloud) saltMinions(token SaltToken) ([]SaltMinion, error) {
	var minions []SaltMinion

	url := v.saltURL + "/minions"
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("X-Auth-Token", token.Token)

	client := &http.Client{}
	resp, err := client.Do(req)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return minions, err
	}

	var minionsResp SaltMinionsResponse
	if err = json.Unmarshal(body, &minionsResp); err != nil {
		return minions, err
	}

	for _, value := range minionsResp.Minions[0] {
		minions = append(minions, value)
	}

	return minions, nil
}

// saltLogin invokes the Salt API to get an authorization token
func (v *VagrantCloud) saltLogin() (SaltToken, error) {
	url := v.saltURL + "/login"
	data := neturl.Values{
		"username": {v.saltUser},
		"password": {v.saltPass},
		"eauth":    {v.saltAuth},
	}

	var token SaltToken
	resp, err := http.PostForm(url, data)
	if err != nil {
		return token, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return token, err
	}

	var loginResp SaltLoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return token, err
	}

	if len(loginResp.Data) == 0 {
		return token, errors.New("No token found in response")
	}

	return loginResp.Data[0], nil
}

// List enumerates the set of minions instances known by the cloud provider
func (v *VagrantCloud) List(filter string) ([]string, error) {
	token, err := v.saltLogin()
	if err != nil {
		return nil, err
	}

	minions, err := v.saltMinions(token)
	if err != nil {
		return nil, err
	}

	filteredMinions := v.saltMinionsByRole(minions, "kubernetes-pool")
	var instances []string
	for _, instance := range filteredMinions {
		instances = append(instances, instance.IP)
	}

	return instances, nil
}
