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
	"encoding/json"
	dmtf "github.com/ODIM-Project/ODIM/lib-dmtf/model"
	"github.com/ODIM-Project/ODIM/lib-utilities/errors"
)

// GetFabricFromDB will collect fabric data from Onisk DB
func GetFabricFromDB(key string) (*Fabric, *errors.Error) {
	var resource Fabric
	resourceData, err := readFromDB(TableFabric, key)
	if err != nil {
		return nil, errors.PackError(err.ErrNo(), "error while trying to get ACI plugin fabric details: ", err.Error())
	}
	if errs := json.Unmarshal([]byte(resourceData), &resource); errs != nil {
		return nil, errors.PackError(errors.UndefinedErrorType, errs)
	}
	return &resource, nil
}

// GetSwitchFromDB will collect switch data from Onisk DB
func GetSwitchFromDB(key string) (*dmtf.Switch, *errors.Error) {
	var resource dmtf.Switch
	resourceData, err := readFromDB(TableSwitch, key)
	if err != nil {
		return nil, errors.PackError(err.ErrNo(), "error while trying to get ACI plugin switch details: ", err.Error())
	}
	if errs := json.Unmarshal([]byte(resourceData), &resource); errs != nil {
		return nil, errors.PackError(errors.UndefinedErrorType, errs)
	}
	return &resource, nil
}

// GetSwitchPortsFromDB will collect ports of all switces from Onisk DB
func GetSwitchPortsFromDB(key string) (SwitchPorts, *errors.Error) {
	var resource SwitchPorts
	resourceData, err := readFromDB(TableSwitchPorts, key)
	if err != nil {
		return nil, errors.PackError(err.ErrNo(), "error while trying to get ACI plugin ports of swtich details: ", err.Error())
	}
	if errs := json.Unmarshal([]byte(resourceData), &resource); errs != nil {
		return nil, errors.PackError(errors.UndefinedErrorType, errs)
	}
	return resource, nil
}

// GetPortFromDB will collect port data from Onisk DB
func GetPortFromDB(key string) (*dmtf.Port, *errors.Error) {
	var resource dmtf.Port
	resourceData, err := readFromDB(TablePort, key)
	if err != nil {
		return nil, errors.PackError(err.ErrNo(), "error while trying to get ACI plugin ports details: ", err.Error())
	}
	if errs := json.Unmarshal([]byte(resourceData), &resource); errs != nil {
		return nil, errors.PackError(errors.UndefinedErrorType, errs)
	}
	return &resource, nil
}
