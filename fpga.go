// Copyright 2018 Xilinx Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io/ioutil"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	"path"
	"regexp"
	"strconv"
)

const (
	SysfsDevices    = "/sys/bus/pci/devices/0000:"
	DevDir          = "/dev"
	MgmtPrefix      = "/dev/"
	UserPrefix      = "/dev/dri/"
	UserPostfix     = "/drm"
	DeviceIDPostfix = "/device"
	DSAverPostfix   = "/VBNV"
	DSAtsPostfix    = "/timestamp"
	DSAPrefix       = "/rom.m."
	MgmtFunc        = ".1"
	UserFunc        = ".0"
	MgmtRE          = `^xclmgmt[0-9]+$`
	UserRE          = `^renderD[0-9]+$`
)

type Pairs struct {
	Mgmt string
	User string
}

type Device struct {
	index     string
	shellVer  string
	timestamp string
	BDF       string // this is for user pf
	deviceID  string //devid of the user pf
	Healthy   string
	Nodes     Pairs
}

func GetBD(mgmtBDFdec string) (string, error) {
	input, err := strconv.Atoi(mgmtBDFdec)
	if err != nil {
		fmt.Println("strconv failed\n")
		return "", err
	}
	bus := input / 256
	dev := (input - bus*256) / 8
	fun := input - bus*256 - dev*8
	if fun != 1 {
		return "", fmt.Errorf("xilinx fpga mgmt func should be 1\n")
	}
	userBD := fmt.Sprintf("%02x:%02x", bus, dev)
	return userBD, nil
}

func GetUserPF(userBDF string) (string, error) {
	userRE := regexp.MustCompile(UserRE)
	dir := SysfsDevices + userBDF + UserPostfix
	userFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("Can't read folder %s \n", dir)
	}
	for _, userFile := range userFiles {
		fname := userFile.Name()

		if !userRE.MatchString(fname) {
			continue
		}
		return fname, nil
	}
	return "", nil
}

func GetFileContent(file string) (string, error) {
	if buf, err := ioutil.ReadFile(file); err != nil {
		return "", fmt.Errorf("Can't read file %s \n", file)
	} else {
		return string(buf), nil
	}
}

func GetDevices() ([]Device, error) {
	mgmtRE := regexp.MustCompile(MgmtRE)
	var devices []Device
	mgmtFiles, err := ioutil.ReadDir(DevDir)
	if err != nil {
		return nil, fmt.Errorf("Can't read folder %s \n", DevDir)
	}

	for _, mgmtFile := range mgmtFiles {
		fname := mgmtFile.Name()

		if !mgmtRE.MatchString(fname) {
			continue
		}

		re := regexp.MustCompile("[^0-9]")
		mgmtBDFdec := re.ReplaceAllString(fname, "")
		mgmt := path.Join(MgmtPrefix, fname)

		BDstr, err := GetBD(mgmtBDFdec)
		if err != nil {
			return nil, err
		}
		userpf, err := GetUserPF(BDstr + UserFunc)
		if err != nil {
			return nil, err
		}
		user := path.Join(UserPrefix, userpf)

		file := SysfsDevices + BDstr + MgmtFunc + DSAPrefix + mgmtBDFdec + DSAverPostfix
		dsaVer, err := GetFileContent(file)
		if err != nil {
			return nil, err
		}
		file = SysfsDevices + BDstr + MgmtFunc + DSAPrefix + mgmtBDFdec + DSAtsPostfix
		dsaTs, err := GetFileContent(file)
		if err != nil {
			return nil, err
		}
		file = SysfsDevices + BDstr + UserFunc + DeviceIDPostfix
		devid, err := GetFileContent(file)
		if err != nil {
			return nil, err
		}

		//TODO: check temp, power, fan speed etc, to give a healthy level
		//so far, return Healthy
		healthy := pluginapi.Healthy
		devices = append(devices, Device{
			index:     strconv.Itoa(len(devices) + 1),
			shellVer:  dsaVer,
			timestamp: dsaTs,
			BDF:       BDstr + UserFunc,
			deviceID:  devid,
			Healthy:   healthy,
			Nodes: Pairs{
				Mgmt: mgmt,
				User: user,
			},
		})
	}
	return devices, nil
}

/*
func main() {
	devices, err := GetDevices()
	if err != nil {
		fmt.Printf("%s !!!\n", err)
		return
	}
	for _, device := range devices {
		fmt.Printf("%v", device)
	}
}
*/
