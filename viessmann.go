package viessmann

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const base = "https://api.viessmann.com/iot/v1/equipment/"

type Api struct {
	ClientId     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
	accessToken  string
	valid        time.Time
	mu           sync.Mutex
}

type token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type Location struct {
	Latitude  float64
	Longitude float64
	TimeZone  string
}

type Address struct {
	Street      string
	HouseNumber string
	Zip         string
	City        string
	Country     string
	Location    Location `json:"geolocation"`
}

type Installation struct {
	api          *Api
	Id           int
	Description  string
	Address      Address
	RegisteredAt time.Time
	UpdatedAt    time.Time
	Status       string `json:"aggregatedStatus"`
	Type         string `json:"installationType"`
}

type installationWrapper struct {
	Data Installation
}

type installationListWrapper struct {
	Data []Installation
}

type Gateway struct {
	api                   *Api
	Serial                string
	Version               string
	FailedFirmwareUpdates int `json:"firmwareUpdateFailureCounter"`
	AutoUpdate            bool
	CreatedAt             time.Time
	ProducedAt            time.Time
	LastStatusChanged     time.Time
	TargetRealm           string
	Status                string `json:"aggregatedStatus"`
	Type                  string `json:"gatewayType"`
	RegisteredAt          time.Time
	InstallationId        int
}

type gatewayWrapper struct {
	Data Gateway
}

type gatewayListWrapper struct {
	Data []Gateway
}

type Device struct {
	api                *Api
	Id                 string
	BoilerSerial       string
	BoilerSerialEditor string
	CreatedAt          time.Time
	EditedAt           time.Time
	ModelId            string
	Status             string
	DeviceType         string
	Roles              []string
	GatewaySerial      string
	InstallationId     int
}

type deviceListWrapper struct {
	Data []Device
}

type Parameter struct {
	Type        string
	Required    bool
	Constraints map[string]interface{}
}

type Command struct {
	Name       string
	Uri        string
	Executable bool `json:"isExecutable"`
	Params     map[string]Parameter
}

type Feature struct {
	api        *Api
	Name       string `json:"feature"`
	Uri        string
	Properties map[string]map[string]interface{}
	Enabled    bool `json:"isEnabled"`
	Ready      bool `json:"isReady"`
	Timestamp  time.Time
	Commands   map[string]Command
}

type featureListWrapper struct {
	Data []Feature
}

func (v *Api) Installation(id string) (Installation, error) {
	var i installationWrapper
	err := v.get(fmt.Sprintf("installations/%s", id), &i)
	if err != nil {
		return Installation{}, err
	}
	i.Data.api = v
	return i.Data, nil
}

func (v *Api) Installations() ([]Installation, error) {
	var i installationListWrapper
	err := v.get("installations", &i)
	if err != nil {
		return nil, err
	}
	for index := range i.Data {
		i.Data[index].api = v
	}
	return i.Data, nil
}

func (i Installation) Gateway(serial string) (Gateway, error) {
	var g gatewayWrapper
	err := i.api.get(fmt.Sprintf("installations/%d/%s", i.Id, serial), &g)
	if err != nil {
		return Gateway{}, err
	}
	g.Data.api = i.api
	return g.Data, nil
}

func (i Installation) Gateways() ([]Gateway, error) {
	var g gatewayListWrapper
	err := i.api.get(fmt.Sprintf("installations/%d/gateways", i.Id), &g)
	if err != nil {
		return nil, err
	}
	for index := range g.Data {
		g.Data[index].api = i.api
	}
	return g.Data, nil
}

func (g Gateway) Devices() ([]Device, error) {
	var d deviceListWrapper
	err := g.api.get(fmt.Sprintf("installations/%d/gateways/%s/devices", g.InstallationId, g.Serial), &d)
	if err != nil {
		return nil, err
	}
	for index := range d.Data {
		d.Data[index].api = g.api
		d.Data[index].GatewaySerial = g.Serial
		d.Data[index].InstallationId = g.InstallationId
	}
	return d.Data, nil
}

func (d Device) Features() ([]Feature, error) {
	var f featureListWrapper
	err := d.api.get(fmt.Sprintf("installations/%d/gateways/%s/devices/%s/features", d.InstallationId, d.GatewaySerial, d.Id), &f)
	if err != nil {
		return nil, err
	}
	return f.Data, nil
}

func (v *Api) refreshAccessToken() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.valid.After(time.Now().Add(10 * time.Minute)) {
		return nil
	}
	data := url.Values{}
	data.Set("client_id", v.ClientId)
	data.Set("refresh_token", v.RefreshToken)
	data.Set("grant_type", "refresh_token")
	res, err := http.Post("https://iam.viessmann.com/idp/v2/token",
		"application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	_ = res.Body.Close()

	var t token
	err = json.Unmarshal(body, &t)
	if err != nil {
		return err
	}

	v.accessToken = t.AccessToken
	v.valid = time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)

	return nil
}

func (v *Api) get(path string, t interface{}) error {
	err := v.refreshAccessToken()
	if err != nil {
		return err
	}
	req, _ := http.NewRequest("GET", base+path, nil)
	req.Header.Set("Authorization", "Bearer "+v.accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%d: %s", res.StatusCode, body)
	}

	err = json.Unmarshal(body, t)
	return err
}
