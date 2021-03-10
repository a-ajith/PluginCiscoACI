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

package capmodel

import (
	"fmt"
	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	"github.com/ODIM-Project/ODIM/lib-utilities/errors"
)

const (
	// TableFabric is the table for storing switch and pod ids
	TableFabric = "ACIFabric"
	// TableSwitch is the table for storing switch information
	TableSwitch = "ACISwitch"
	// TableSwitchPorts is the table for storing ports of each switch
	TableSwitchPorts = "ACISwitchPorts"
	// TablePort is the table for storing port information
	TablePort = "ACIPort"
)

// Fabric ACI holds data of switch id and pod id
type Fabric struct {
	SwitchData []string `json:"switchData"`
	PodID      string   `json:"podID"`
}

// SwitchPorts holds ports of each switch
type SwitchPorts map[string][]string

// SaveToDB will save any resource data into the database
func SaveToDB(table string, key string, data interface{}) error {
	connPool, err := common.GetDBConnection(common.OnDisk)
	if err != nil {
		return fmt.Errorf("error while trying to connecting to DB: %v", err.Error())
	}
	if err = connPool.Create(table, key, data); err != nil {
		return fmt.Errorf("error while trying to create new %v resource: %v", table, err.Error())
	}
	return nil
}

func readFromDB(table, key string) (string, *errors.Error) {
	conn, err := common.GetDBConnection(common.OnDisk)
	if err != nil {
		return "", errors.PackError(err.ErrNo(), err)
	}
	return conn.Read(table, key)
}
