/****************************************************
Copyright 2019 The tesraevent Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*****************************************************/

/***************************************************
Copyright 2016 https://github.com/AsynkronIT/protoactor-go

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*****************************************************/
package router

import (
	"log"

	"github.com/TesraSupernet/tesraevent/actor"
	"github.com/serialx/hashring"
)

type Hasher interface {
	Hash() string
}

type consistentHashGroupRouter struct {
	GroupRouter
}

type consistentHashPoolRouter struct {
	PoolRouter
}

type hashmapContainer struct {
	hashring  *hashring.HashRing
	routeeMap map[string]*actor.PID
}
type consistentHashRouterState struct {
	hmc *hashmapContainer
}

func (state *consistentHashRouterState) SetRoutees(routees *actor.PIDSet) {
	//lookup from node name to PID
	hmc := hashmapContainer{}
	hmc.routeeMap = make(map[string]*actor.PID)
	nodes := make([]string, routees.Len())
	routees.ForEach(func(i int, pid actor.PID) {
		nodeName := pid.Address + "@" + pid.Id
		nodes[i] = nodeName
		hmc.routeeMap[nodeName] = &pid
	})
	//initialize hashring for mapping message keys to node names
	hmc.hashring = hashring.New(nodes)
	state.hmc = &hmc
}

func (state *consistentHashRouterState) GetRoutees() *actor.PIDSet {
	var routees actor.PIDSet
	for _, v := range state.hmc.routeeMap {
		routees.Add(v)
	}
	return &routees
}

func (state *consistentHashRouterState) RouteMessage(message interface{}) {
	_, uwpMsg, _ := actor.UnwrapEnvelope(message)
	switch msg := uwpMsg.(type) {
	case Hasher:
		key := msg.Hash()
		hmc := state.hmc

		node, ok := hmc.hashring.GetNode(key)
		if !ok {
			log.Printf("[ROUTING] Consistent has router failed to derminate routee: %v", key)
			return
		}
		if routee, ok := hmc.routeeMap[node]; ok {
			routee.Tell(message)
		} else {
			log.Println("[ROUTING] Consisten router failed to resolve node", node)
		}
	default:
		log.Println("[ROUTING] Message must implement router.Hasher", msg)
	}
}

func (state *consistentHashRouterState) InvokeRouterManagementMessage(msg ManagementMessage, sender *actor.PID) {

}

func NewConsistentHashPool(size int) *actor.Props {
	return actor.FromSpawnFunc(spawner(&consistentHashPoolRouter{PoolRouter{PoolSize: size}}))
}

func NewConsistentHashGroup(routees ...*actor.PID) *actor.Props {
	return actor.FromSpawnFunc(spawner(&consistentHashGroupRouter{GroupRouter{Routees: actor.NewPIDSet(routees...)}}))
}

func (config *consistentHashPoolRouter) CreateRouterState() Interface {
	return &consistentHashRouterState{}
}

func (config *consistentHashGroupRouter) CreateRouterState() Interface {
	return &consistentHashRouterState{}
}
