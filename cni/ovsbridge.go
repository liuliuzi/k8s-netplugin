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
    "encoding/json"
    "fmt"
    "os"
    "runtime"
    "strings"

    "github.com/containernetworking/cni/pkg/skel"
    "github.com/containernetworking/cni/pkg/types"
    "github.com/containernetworking/cni/pkg/version"
    osexec "os/exec"
    "syscall"
)

const defaultBrName = "cni0"
const vlanip        = "192.168.255.126/24"
const dockerPath    = "/usr/bin/docker"
const ovsPath       = "/usr/bin/ovs-vsctl"
const ipPath        = "/bin/ip"
const pipeworkPath  = "/usr/bin/pipework"

type NetConf struct {
    types.NetConf
    BrName       string `json:"bridge"`
    IsGW         bool   `json:"isGateway"`
    IsDefaultGW  bool   `json:"isDefaultGateway"`
    ForceAddress bool   `json:"forceAddress"`
    IPMasq       bool   `json:"ipMasq"`
    MTU          int    `json:"mtu"`
    HairpinMode  bool   `json:"hairpinMode"`
}

var ErrExecutableNotFound = osexec.ErrNotFound

// Interface is an interface that presents a subset of the os/exec API.  Use this
// when you want to inject fakeable/mockable exec behavior.
type Interface interface {
    // Command returns a Cmd instance which can be used to run a single command.
    // This follows the pattern of package os/exec.
    Command(cmd string, args ...string) Cmd

    // LookPath wraps os/exec.LookPath
    LookPath(file string) (string, error)
}

// Cmd is an interface that presents an API that is very similar to Cmd from os/exec.
// As more functionality is needed, this can grow.  Since Cmd is a struct, we will have
// to replace fields with get/set method pairs.
type Cmd interface {
    // CombinedOutput runs the command and returns its combined standard output
    // and standard error.  This follows the pattern of package os/exec.
    CombinedOutput() ([]byte, error)
    // Output runs the command and returns standard output, but not standard err
    Output() ([]byte, error)
    SetDir(dir string)
}

// ExitError is an interface that presents an API similar to os.ProcessState, which is
// what ExitError from os/exec is.  This is designed to make testing a bit easier and
// probably loses some of the cross-platform properties of the underlying library.
type ExitError interface {
    String() string
    Error() string
    Exited() bool
    ExitStatus() int
}

// Implements Interface in terms of really exec()ing.
type executor struct{}

// New returns a new Interface which will os/exec to run commands.
func New() Interface {
    return &executor{}
}

// Command is part of the Interface interface.
func (executor *executor) Command(cmd string, args ...string) Cmd {
    return (*cmdWrapper)(osexec.Command(cmd, args...))
}

// LookPath is part of the Interface interface
func (executor *executor) LookPath(file string) (string, error) {
    return osexec.LookPath(file)
}

// Wraps exec.Cmd so we can capture errors.
type cmdWrapper osexec.Cmd

func (cmd *cmdWrapper) SetDir(dir string) {
    cmd.Dir = dir
}

// CombinedOutput is part of the Cmd interface.
func (cmd *cmdWrapper) CombinedOutput() ([]byte, error) {
    out, err := (*osexec.Cmd)(cmd).CombinedOutput()
    if err != nil {
        return out, handleError(err)
    }
    return out, nil
}

func (cmd *cmdWrapper) Output() ([]byte, error) {
    out, err := (*osexec.Cmd)(cmd).Output()
    if err != nil {
        return out, handleError(err)
    }
    return out, nil
}

func handleError(err error) error {
    if ee, ok := err.(*osexec.ExitError); ok {
        // Force a compile fail if exitErrorWrapper can't convert to ExitError.
        var x ExitError = &exitErrorWrapper{ee}
        return x
    }
    if ee, ok := err.(*osexec.Error); ok {
        if ee.Err == osexec.ErrNotFound {
            return ErrExecutableNotFound
        }
    }
    return err
}

// exitErrorWrapper is an implementation of ExitError in terms of os/exec ExitError.
// Note: standard exec.ExitError is type *os.ProcessState, which already implements Exited().
type exitErrorWrapper struct {
    *osexec.ExitError
}

// ExitStatus is part of the ExitError interface.
func (eew exitErrorWrapper) ExitStatus() int {
    ws, ok := eew.Sys().(syscall.WaitStatus)
    if !ok {
        panic("can't call ExitStatus() on a non-WaitStatus exitErrorWrapper")
    }
    return ws.ExitStatus()
}

func init() {
    // this ensures that main runs only on main thread (thread group leader).
    // since namespace ops (unshare, setns) are done for a single thread, we
    // must ensure that the goroutine does not jump from OS thread to thread
    runtime.LockOSThread()
}

func loadNetConf(bytes []byte) (*NetConf, error) {
    n := &NetConf{
        BrName: defaultBrName,
    }
    if err := json.Unmarshal(bytes, n); err != nil {
        return nil, fmt.Errorf("failed to load netconf: %v", err)
    }
    return n, nil
}

func addVeth(n *NetConf,containerID string,vlanTag string) ( error) {
    out, err := New().Command(pipeworkPath, n.BrName, containerID,vlanip,"@"+vlanTag).CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to add veth %q: %v", out, err)
    }
    return  nil
}

func ensureBridge(brName string) (error) {
    out, err := New().Command(ovsPath, "show").CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to ls bridge %q: %v", out, err)
    }
    if string(out) != "" {
        outlines:=strings.Split(string(out), "\n")
        for _, numline := range outlines {
            if strings.Contains(numline, brName){
                return nil
            }
        }
        _, err := New().Command(ovsPath, "add-br",brName).CombinedOutput()
        if err != nil {
            return fmt.Errorf("failed to ls bridge %q: %v", out, err)
        }
    }
    return nil
}

func setupBridge(n *NetConf) (error) {
    // create bridge if necessary
    err := ensureBridge(n.BrName)
    if err != nil {
        return fmt.Errorf("failed to create bridge %q: %v", n.BrName, err)
    }
    return nil
}

func cmdAdd(args *skel.CmdArgs) error {
    n, err := loadNetConf(args.StdinData)
    if err != nil {
        return err
    }
    err= setupBridge(n)
    if err != nil {
        return err
    }
    containerID:=os.Getenv("CNI_CONTAINERID")
    podName:=os.Getenv("K8S_POD_NAME")
    podNameList:=strings.Split(podName,"-")
    vlanTag:=podNameList[0]
    err= addVeth(n,containerID,vlanTag)
    if err != nil {
        return err
    }
    return nil
}

func getNspid(containerID string) (string, error){
    out, err := New().Command(dockerPath,"inspect","--format='{{ .State.Pid }}'",containerID).CombinedOutput()
    if err != nil {
        return containerID,fmt.Errorf("failed to ls bridge %q: %v", out, err)
    }
    return string(out), err
}

func cmdDel(args *skel.CmdArgs) error {
    _, err := loadNetConf(args.StdinData)
    if err != nil {
        return err
    }
    containerID:=os.Getenv("CNI_CONTAINERID")
    NSPID, err := getNspid(containerID)
    if err != nil {
        return fmt.Errorf("failed to get NSPID %q", NSPID)
    }
    outlines:=strings.Split(string(NSPID), "\n")
    NSPID=outlines[0]
    CONTAINER_IFNAME:="eth1"
    vethname:="v"+CONTAINER_IFNAME+"pl"+NSPID
    _, err = New().Command(ipPath, "link", "del", vethname).CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to ip link del %q: %v", vethname, err)
    }
    _, err = New().Command(ovsPath, "del-port", vethname).CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to del port %q: %v", vethname, err)
    }
    return nil
}

func main() {
    skel.PluginMain(cmdAdd, cmdDel, version.Legacy)
}
