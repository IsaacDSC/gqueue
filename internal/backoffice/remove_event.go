package backoffice

import (
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
)

/*
implementar CRUD completo
implementar conexão com PG ao invés de Mongodb
implementar job de validação se o serviço está conectado ou não
*/
func RemoveEvent(cc cachemanager.Cache, repo Repository) httpsvc.HttpHandle {
	return httpsvc.HttpHandle{}
}
