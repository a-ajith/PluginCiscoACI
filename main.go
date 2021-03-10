//(C) Copyright [2020] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.
package main

import (
	"encoding/json"
	dmtfmodel "github.com/ODIM-Project/ODIM/lib-dmtf/model"
	dc "github.com/ODIM-Project/ODIM/lib-messagebus/datacommunicator"
	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	lutilconf "github.com/ODIM-Project/ODIM/lib-utilities/config"
	"github.com/ODIM-Project/PluginCiscoACI/capdata"
	"github.com/ODIM-Project/PluginCiscoACI/caphandler"
	"github.com/ODIM-Project/PluginCiscoACI/capmessagebus"
	"github.com/ODIM-Project/PluginCiscoACI/capmiddleware"
	"github.com/ODIM-Project/PluginCiscoACI/capmodel"
	"github.com/ODIM-Project/PluginCiscoACI/caputilities"
	"github.com/ODIM-Project/PluginCiscoACI/config"
	"github.com/ODIM-Project/PluginCiscoACI/constants"
	"github.com/ciscoecosystem/aci-go-client/models"
	iris "github.com/kataras/iris/v12"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var subscriptionInfo []capmodel.Device
var log = logrus.New()

// TokenObject will contains the generated token and public key of odimra
type TokenObject struct {
	AuthToken string `json:"authToken"`
	PublicKey []byte `json:"publicKey"`
}

func main() {
	// verifying the uid of the user
	if uid := os.Geteuid(); uid == 0 {
		log.Fatal("Plugin Service should not be run as the root user")
	}

	if err := config.SetConfiguration(); err != nil {
		log.Fatal("while reading from config, PluginCiscoACI got" + err.Error())
	}

	if err := dc.SetConfiguration(config.Data.MessageBusConf.MessageQueueConfigFilePath); err != nil {
		log.Fatal("while trying to set messagebus configuration, PluginCiscoACI got: " + err.Error())
	}

	// CreateJobQueue defines the queue which will act as an infinite buffer
	// In channel is an entry or input channel and the Out channel is an exit or output channel
	caphandler.In, caphandler.Out = common.CreateJobQueue()

	// RunReadWorkers will create a worker pool for doing a specific task
	// which is passed to it as Publish method after reading the data from the channel.
	go common.RunReadWorkers(caphandler.Out, capmessagebus.Publish, 1)
	intializeACIData()
	intializePluginStatus()
	app()
}

func app() {
	app := routers()
	go func() {
		eventsrouters()
	}()
	conf := &lutilconf.HTTPConfig{
		Certificate:   &config.Data.KeyCertConf.Certificate,
		PrivateKey:    &config.Data.KeyCertConf.PrivateKey,
		CACertificate: &config.Data.KeyCertConf.RootCACertificate,
		ServerAddress: config.Data.PluginConf.Host,
		ServerPort:    config.Data.PluginConf.Port,
	}
	pluginServer, err := conf.GetHTTPServerObj()
	if err != nil {
		log.Fatal("while initializing plugin server, PluginCiscoACI got: " + err.Error())
	}
	app.Run(iris.Server(pluginServer))
}

func routers() *iris.Application {
	app := iris.New()
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		path := r.URL.Path
		if len(path) > 1 && path[len(path)-1] == '/' && path[len(path)-2] != '/' {
			path = path[:len(path)-1]
			r.RequestURI = path
			r.URL.Path = path
		}
		next(w, r)
	})

	pluginRoutes := app.Party("/ODIM/v1")
	pluginRoutes.Post("/validate", capmiddleware.BasicAuth, caphandler.Validate)
	pluginRoutes.Post("/Sessions", caphandler.CreateSession)
	pluginRoutes.Post("/Subscriptions", capmiddleware.BasicAuth, caphandler.CreateEventSubscription)
	pluginRoutes.Delete("/Subscriptions", capmiddleware.BasicAuth, caphandler.DeleteEventSubscription)
	pluginRoutes.Get("/Status", capmiddleware.BasicAuth, caphandler.GetPluginStatus)
	pluginRoutes.Post("/Startup", capmiddleware.BasicAuth, caphandler.GetPluginStartup)
	fabricRoutes := pluginRoutes.Party("/Fabrics", capmiddleware.BasicAuth)
	fabricRoutes.Get("/", caphandler.GetFabricResource)
	fabricRoutes.Get("/{id}", caphandler.GetFabricData)
	fabricRoutes.Get("/{id}/Switches", caphandler.GetSwitchCollection)
	fabricRoutes.Get("/{id}/Switches/{rid}", caphandler.GetSwitchInfo)
	fabricRoutes.Get("/{id}/Switches/{switchID}/Ports", caphandler.GetPortCollection)
	fabricRoutes.Get("/{id}/Switches/{switchID}/Ports/{portID}", caphandler.GetPortInfo)
	fabricRoutes.Get("/{id}/Zones", caphandler.GetFabricResource)
	fabricRoutes.Post("/{id}/Zones", caphandler.GetFabricResource)
	fabricRoutes.Get("/{id}/Zones/{rid}", caphandler.GetFabricResource)
	fabricRoutes.Delete("/{id}/Zones/{rid}", caphandler.GetFabricResource)
	fabricRoutes.Patch("/{id}/Zones/{rid}", caphandler.GetFabricResource)
	fabricRoutes.Get("/{id}/AddressPools", caphandler.GetAddressPoolCollection)
	fabricRoutes.Post("/{id}/AddressPools", caphandler.CreateAddressPool)
	fabricRoutes.Get("/{id}/AddressPools/{rid}", caphandler.GetAddressPoolInfo)
	fabricRoutes.Delete("/{id}/AddressPools/{rid}", caphandler.DeleteAddressPoolInfo)

	managers := pluginRoutes.Party("/Managers")
	managers.Get("/", caphandler.GetManagersCollection)
	managers.Get("/{id}", caphandler.GetManagersInfo)
	taskmon := pluginRoutes.Party("/taskmon")
	taskmon.Get("/{TaskID}", caphandler.GetTaskMonitor)

	task := pluginRoutes.Party("/TaskService")
	task.Get("/", caphandler.GetTaskService)
	task.Get("/Tasks", caphandler.GetTaskService)
	task.Get("/Tasks/{TaskID}", caphandler.GetTaskService)
	task.Get("/Tasks/{TaskID}/SubTasks", caphandler.GetTaskService)
	task.Get("/Tasks/{TaskID}/SubTasks/{subTaskID}", caphandler.GetTaskService)
	task.Delete("/Tasks/{TaskID}", caphandler.GetTaskService)

	return app
}

func eventsrouters() {
	app := iris.New()
	app.Post(config.Data.EventConf.DestURI, caphandler.RedfishEvents)
	conf := &lutilconf.HTTPConfig{
		Certificate:   &config.Data.KeyCertConf.Certificate,
		PrivateKey:    &config.Data.KeyCertConf.PrivateKey,
		CACertificate: &config.Data.KeyCertConf.RootCACertificate,
		ServerAddress: config.Data.EventConf.ListenerHost,
		ServerPort:    config.Data.EventConf.ListenerPort,
	}
	evtServer, err := conf.GetHTTPServerObj()
	if err != nil {
		log.Fatal("while initializing event server, PluginCiscoACI got: " + err.Error())
	}
	app.Run(iris.Server(evtServer))
}

// intializePluginStatus sets plugin status
func intializePluginStatus() {
	caputilities.Status.Available = "no"
	caputilities.Status.Uptime = time.Now().Format(time.RFC3339)
}

// intializeACIData reads required fabric,switch and port data from aci and stored it in the data store
func intializeACIData() {
	capdata.FabricDataStore.Data = make(map[string]*capdata.Fabric)
	capdata.SwitchDataStore.Data = make(map[string]*dmtfmodel.Switch, 0)
	capdata.SwitchToPortDataStore = make(map[string][]string)
	capdata.PortDataStore = make(map[string]*dmtfmodel.Port)
	capdata.AddressPoolDataStore = make(map[string]*capdata.AddressPoolsData)
	aciNodesData, err := caputilities.GetFabricNodeData()
	if err != nil {
		log.Fatal("while intializing ACI Data  PluginCiscoACI got: " + err.Error())
	}
	for _, aciNodeData := range aciNodesData {
		switchID := uuid.NewV4().String() + ":" + aciNodeData.NodeId
		capdata.FabricDataStore.Lock.Lock()
		fabricID := config.Data.RootServiceUUID + ":" + aciNodeData.FabricId
		if data, ok := capdata.FabricDataStore.Data[fabricID]; ok {
			data.SwitchData = append(data.SwitchData, switchID)
			data.PodID = aciNodeData.PodId
		} else {

			capdata.FabricDataStore.Data[fabricID] = &capdata.Fabric{
				SwitchData: []string{
					switchID,
				},
				PodID: aciNodeData.PodId,
			}
		}
		capdata.FabricDataStore.Lock.Unlock()
		switchData := getSwitchData(aciNodeData, switchID)
		capdata.SwitchDataStore.Lock.Lock()
		capdata.SwitchDataStore.Data[switchID] = switchData
		capdata.SwitchDataStore.Lock.Unlock()
		// adding logic to collect the ports data
		portData, err := caputilities.GetPortData(aciNodeData.PodId, aciNodeData.NodeId)
		if err != nil {
			log.Fatal("while intializing ACI Port  Data  PluginCiscoACI got: " + err.Error())
		}
		parsePortData(portData, switchID)

	}

	// TODO:
	// registering the for the aci events

	//updating the plugin status
	caputilities.Status.Available = "yes"
	// Send resource added event odim
	capdata.FabricDataStore.Lock.RLock()
	for fabricID := range capdata.FabricDataStore.Data {
		var event = common.Event{
			EventID:   uuid.NewV4().String(),
			MessageID: constants.ResourceCreatedMessageID,
			EventType: "ResourceAdded",
			OriginOfCondition: &common.Link{
				Oid: "/ODIM/v1/Fabrics/" + fabricID,
			},
		}
		var events = []common.Event{event}
		var messageData = common.MessageData{
			Name:      "Fabric added event",
			Context:   "/redfish/v1/$metadata#Event.Event",
			OdataType: constants.EventODataType,
			Events:    events,
		}
		data, _ := json.Marshal(messageData)
		eventData := common.Events{
			IP:      config.Data.LoadBalancerConf.Host,
			Request: data,
		}
		capmessagebus.Publish(eventData)
	}
	capdata.FabricDataStore.Lock.RUnlock()

	return
}

// parsePortData parses the portData and stores it  in the inmemory
func parsePortData(portResponseData *capmodel.PortCollectionResponse, switchID string) {
	var portData []string
	for _, imdata := range portResponseData.IMData {
		portAttributes := imdata.PhysicalInterface.Attributes
		id := portAttributes["id"].(string)
		id = strings.Replace(id, "/", "-", -1)
		portID := uuid.NewV4().String() + ":" + id
		portData = append(portData, portID)
		portInfo := dmtfmodel.Port{
			ODataContext:          "/ODIM/v1/$metadata#Port.Port",
			ODataType:             "#Port.v1_3_0.Port",
			ID:                    portID,
			Name:                  "Port-" + portAttributes["id"].(string),
			PortID:                portAttributes["id"].(string),
			PortProtocol:          "Ethernet",
			PortType:              "BidirectionalPort",
			LinkNetworkTechnology: "Ethernet",
		}
		mtu, err := strconv.Atoi(portAttributes["mtu"].(string))
		if err != nil {
			log.Error("Unable to get mtu for the port" + portID)
		}
		portInfo.MaxFrameSize = mtu
		capdata.PortDataStore[portID] = &portInfo
	}
	capdata.SwitchToPortDataStore[switchID] = portData
}

func getSwitchData(fabricNodeData *models.FabricNodeMember, switchID string) *dmtfmodel.Switch {
	switchUUIDData := strings.Split(switchID, ":")
	var switchData = dmtfmodel.Switch{
		ODataContext: "/ODIM/v1/$metadata#Switch.Switch",
		ODataType:    "#Switch.v1_4_0.Switch",
		ID:           switchID,
		Name:         fabricNodeData.Name,
		SwitchType:   "Ethernet",
		UUID:         switchUUIDData[0],
		SerialNumber: fabricNodeData.Serial,
	}
	podID, err := strconv.Atoi(fabricNodeData.PodId)
	if err != nil {
		log.Fatal("Converstion of PODID" + fabricNodeData.PodId + " failed")
	}
	nodeID, err := strconv.Atoi(fabricNodeData.NodeId)
	if err != nil {
		log.Fatal("Converstion of NodeID" + fabricNodeData.NodeId + " failed")
	}
	log.Info("Getting the switchData for NodeID" + fabricNodeData.NodeId)
	switchRespData, err := caputilities.GetSwitchInfo(podID, nodeID)
	if err != nil {
		log.Fatal("Unable to get the Switch info:" + err.Error())
	}
	switchData.FirmwareVersion = switchRespData.SystemAttributes.Version
	switchChassisData, err := caputilities.GetSwitchChassisInfo(fabricNodeData.PodId, fabricNodeData.NodeId)
	if err != nil {
		log.Fatal("Unable to get the Switch Chassis info for node " + fabricNodeData.NodeId + " :" + err.Error())
	}
	switchData.Manufacturer = switchChassisData.IMData[0].SwitchChassisData.Attributes["vendor"].(string)
	switchData.Model = switchChassisData.IMData[0].SwitchChassisData.Attributes["model"].(string)
	return &switchData
}
