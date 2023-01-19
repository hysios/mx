// License: MIT License
// package provision
package provisioning

import (
	"reflect"
	"sync"
)

type Provisioning struct {
	registry sync.Map
}

var provisioning = &Provisioning{}

// Provision provision 引导安装流程入口
func Provision(provision any) {
	provisioning.Provision(provision)
}

// Init provision 初始化对象
func Init(obj any) {
	provisioning.Init(obj)
}

// Init provision 初始化对象
func (p *Provisioning) Init(obj any) {
	var t = reflect.TypeOf(obj)

	provisioners := p.GetProvisioners(t)
	for _, provisioner := range provisioners {
		if !provisioner.IsValid() {
			panic("provisioner not found")
		}
		provisioner.Call([]reflect.Value{reflect.ValueOf(obj)})
	}
}

// Provision provision 引导安装流程入口
func (p *Provisioning) Provision(provisioner any) {
	// using reflect get provisioner callback func arg type
	var (
		pro = reflect.ValueOf(provisioner)
		t   = reflect.TypeOf(provisioner)
	)
	if t.Kind() != reflect.Func {
		panic("provisioner must be a func")
	}

	// using reflect get provisioner callback func arg type
	if t.NumIn() != 1 {
		panic("provisioner must be a func with a can recived arg")
	}

	// using reflect get provisioner callback func arg type
	argType := t.In(0)
	if old, load := p.registry.LoadOrStore(argType, []reflect.Value{pro}); load {
		if old, ok := old.([]reflect.Value); ok {
			p.registry.Store(argType, append(old, pro))
		}
	}
}

func (p *Provisioning) GetProvisioners(argType reflect.Type) []reflect.Value {
	if v, ok := p.registry.Load(argType); ok {
		return v.([]reflect.Value)
	}
	return nil
}
