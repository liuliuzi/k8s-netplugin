// Copyright 2014 CNI authors
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

)

const defaultBrName = "cni0"
const vlanip        = "192.168.255.126/24"
const dockerPath    = "/usr/bin/docker"

type NetConf struct {
}

var ErrExecutableNotFound = osexec.ErrNotFound

// Interface is an interface that presents a subset of the os/exec API.  Use this
// when you want to inject fakeable/mockable exec behavior.
type Interface interface {

}

// Cmd is an interface that presents an API that is very similar to Cmd from os/exec.
// As more functionality is needed, this can grow.  Since Cmd is a struct, we will have
// to replace fields with get/set method pairs.
type Cmd interface {
}

// ExitError is an interface that presents an API similar to os.ProcessState, which is
// what ExitError from os/exec is.  This is designed to make testing a bit easier and
// probably loses some of the cross-platform properties of the underlying library.
type ExitError interface {
}

// Implements Interface in terms of really exec()ing.
type executor struct{}

// New returns a new Interface which will os/exec to run commands.
func New() Interface {
}

// Command is part of the Interface interface.
func (executor *executor) Command(cmd string, args ...string) Cmd {
}

// LookPath is part of the Interface interface
func (executor *executor) LookPath(file string) (string, error) {
}

// Wraps exec.Cmd so we can capture errors.
type cmdWrapper osexec.Cmd

func (cmd *cmdWrapper) SetDir(dir string) {
}

// CombinedOutput is part of the Cmd interface.
func (cmd *cmdWrapper) CombinedOutput() ([]byte, error) {
}

func (cmd *cmdWrapper) Output() ([]byte, error) {
}

func handleError(err error) error {
}

// exitErrorWrapper is an implementation of ExitError in terms of os/exec ExitError.
// Note: standard exec.ExitError is type *os.ProcessState, which already implements Exited().
type exitErrorWrapper struct {
}

// ExitStatus is part of the ExitError interface.
func (eew exitErrorWrapper) ExitStatus() int {
}

func init() {
}

func loadNetConf(bytes []byte) (*NetConf, error) {
}

func addVeth(n *NetConf,containerID string,vlanTag string) ( error) {
}

func ensureBridge(brName string) (error) {
            }
}

func setupBridge(n *NetConf) (error) {
}

func cmdAdd(args *skel.CmdArgs) error {
}

func getNspid(containerID string) (string, error){
}

func cmdDel(args *skel.CmdArgs) error {
    NSPID, err := getNspid(containerID)
    CONTAINER_IFNAME:="eth1"
    vethname:="v"+CONTAINER_IFNAME+"pl"+NSPID
    _, err = New().Command(ipPath, "link", "del", vethname).CombinedOutput()
    _, err = New().Command(ovsPath, "del-port", vethname).CombinedOutput()
}

func main() {
}
