package services

import "sync"

// Simple shared instance registry so background jobs can use the same MikroTik connection

var sharedMikroTikService *MikroTikService
var sharedRouterOSAPI *RouterOSAPI
var sharedRouterOSAPIMu sync.RWMutex

func SetSharedMikroTikService(svc *MikroTikService) {
	sharedMikroTikService = svc
}

func GetSharedMikroTikService() *MikroTikService {
	return sharedMikroTikService
}

func SetSharedRouterOSAPI(client *RouterOSAPI) {
	sharedRouterOSAPIMu.Lock()
	defer sharedRouterOSAPIMu.Unlock()
	sharedRouterOSAPI = client
}

func GetSharedRouterOSAPI() *RouterOSAPI {
	sharedRouterOSAPIMu.RLock()
	defer sharedRouterOSAPIMu.RUnlock()
	return sharedRouterOSAPI
}

func ClearSharedRouterOSAPI(client *RouterOSAPI) {
	sharedRouterOSAPIMu.Lock()
	defer sharedRouterOSAPIMu.Unlock()
	if sharedRouterOSAPI == client {
		sharedRouterOSAPI = nil
	}
}
