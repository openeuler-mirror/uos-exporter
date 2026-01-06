package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	crmMonElemEnabled = "summary,nodes,node_attributes,clones,resources,resources_group,failures,bans"
)

type crmMonCollector struct {
	crmMonInfo                        *prometheus.Desc
	crmMonLastUpdate                  *prometheus.Desc
	crmMonLastChange                  *prometheus.Desc
	crmMonDCPresent                   *prometheus.Desc
	crmMonDCQuorum                    *prometheus.Desc
	crmMonNodesConfigured             *prometheus.Desc
	crmMonResourcesConfigured         *prometheus.Desc
	crmMonResourcesDisabled           *prometheus.Desc
	crmMonResourcesBlocked            *prometheus.Desc
	crmMonStonith                     *prometheus.Desc
	crmMonSymmetricCluster            *prometheus.Desc
	crmMonMaintenanceMode             *prometheus.Desc
	crmMonNodeID                      *prometheus.Desc
	crmMonNodeOnline                  *prometheus.Desc
	crmMonNodeStandby                 *prometheus.Desc
	crmMonNodeStandbyOnFail           *prometheus.Desc
	crmMonNodeMaintenance             *prometheus.Desc
	crmMonNodePending                 *prometheus.Desc
	crmMonNodeUnclean                 *prometheus.Desc
	crmMonNodeShutdown                *prometheus.Desc
	crmMonNodeExpectedUp              *prometheus.Desc
	crmMonNodeIsDC                    *prometheus.Desc
	crmMonNodeResourcesRunning        *prometheus.Desc
	crmMonNodeAttribute               *prometheus.Desc
	crmMonResourceActive              *prometheus.Desc
	crmMonResourceOrphaned            *prometheus.Desc
	crmMonResourceBlocked             *prometheus.Desc
	crmMonResourceManaged             *prometheus.Desc
	crmMonResourceFailed              *prometheus.Desc
	crmMonResourceFailureIgnored      *prometheus.Desc
	crmMonResourcesGroup              *prometheus.Desc
	crmMonResourceGroupActive         *prometheus.Desc
	crmMonResourceGroupOrphaned       *prometheus.Desc
	crmMonResourceGroupBlocked        *prometheus.Desc
	crmMonResourceGroupManaged        *prometheus.Desc
	crmMonResourceGroupFailed         *prometheus.Desc
	crmMonResourceGroupFailureIgnored *prometheus.Desc
	crmMonResourceCloneMultistate     *prometheus.Desc
	crmMonResourceClonePromoted       *prometheus.Desc
	crmMonResourceCloneActive         *prometheus.Desc
	crmMonResourceCloneOrphaned       *prometheus.Desc
	crmMonResourceCloneBlocked        *prometheus.Desc
	crmMonResourceCloneManaged        *prometheus.Desc
	crmMonResourceCloneFailed         *prometheus.Desc
	crmMonResourceCloneFailureIgnored *prometheus.Desc
	crmMonResourceCloneNumActive      *prometheus.Desc
	crmMonResourceCloneNumPromoted    *prometheus.Desc
	crmMonFailuresCount               *prometheus.Desc
	crmMonFailureDescription          *prometheus.Desc
	crmMonBansCount                   *prometheus.Desc
	crmMonBanDescription              *prometheus.Desc
}


// TODO: implement functions
