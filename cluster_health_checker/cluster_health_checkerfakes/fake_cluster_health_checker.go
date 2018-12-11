// Code generated by counterfeiter. DO NOT EDIT.
package cluster_health_checkerfakes

import (
	sync "sync"

	cluster_health_checker "github.com/cloudfoundry/galera-init/cluster_health_checker"
)

type FakeClusterHealthChecker struct {
	HealthyClusterStub        func() bool
	healthyClusterMutex       sync.RWMutex
	healthyClusterArgsForCall []struct {
	}
	healthyClusterReturns struct {
		result1 bool
	}
	healthyClusterReturnsOnCall map[int]struct {
		result1 bool
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeClusterHealthChecker) HealthyCluster() bool {
	fake.healthyClusterMutex.Lock()
	ret, specificReturn := fake.healthyClusterReturnsOnCall[len(fake.healthyClusterArgsForCall)]
	fake.healthyClusterArgsForCall = append(fake.healthyClusterArgsForCall, struct {
	}{})
	fake.recordInvocation("HealthyCluster", []interface{}{})
	fake.healthyClusterMutex.Unlock()
	if fake.HealthyClusterStub != nil {
		return fake.HealthyClusterStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.healthyClusterReturns
	return fakeReturns.result1
}

func (fake *FakeClusterHealthChecker) HealthyClusterCallCount() int {
	fake.healthyClusterMutex.RLock()
	defer fake.healthyClusterMutex.RUnlock()
	return len(fake.healthyClusterArgsForCall)
}

func (fake *FakeClusterHealthChecker) HealthyClusterCalls(stub func() bool) {
	fake.healthyClusterMutex.Lock()
	defer fake.healthyClusterMutex.Unlock()
	fake.HealthyClusterStub = stub
}

func (fake *FakeClusterHealthChecker) HealthyClusterReturns(result1 bool) {
	fake.healthyClusterMutex.Lock()
	defer fake.healthyClusterMutex.Unlock()
	fake.HealthyClusterStub = nil
	fake.healthyClusterReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeClusterHealthChecker) HealthyClusterReturnsOnCall(i int, result1 bool) {
	fake.healthyClusterMutex.Lock()
	defer fake.healthyClusterMutex.Unlock()
	fake.HealthyClusterStub = nil
	if fake.healthyClusterReturnsOnCall == nil {
		fake.healthyClusterReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.healthyClusterReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakeClusterHealthChecker) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.healthyClusterMutex.RLock()
	defer fake.healthyClusterMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeClusterHealthChecker) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ cluster_health_checker.ClusterHealthChecker = new(FakeClusterHealthChecker)
