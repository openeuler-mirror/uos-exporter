package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

type VgExporter struct {
	lv_count            *prometheus.Desc
	max_lv              *prometheus.Desc
	max_pv              *prometheus.Desc
	pv_count            *prometheus.Desc
	snap_count          *prometheus.Desc
	vg_clustered        *prometheus.Desc
	vg_exported         *prometheus.Desc
	vg_extendable       *prometheus.Desc
	vg_extent_count     *prometheus.Desc
	vg_extent_size      *prometheus.Desc
	vg_free             *prometheus.Desc
	vg_free_count       *prometheus.Desc
	vg_mda_copies       *prometheus.Desc
	vg_mda_count        *prometheus.Desc
	vg_mda_free         *prometheus.Desc
	vg_mda_size         *prometheus.Desc
	vg_mda_used_count   *prometheus.Desc
	vg_missing_pv_count *prometheus.Desc
	vg_partial          *prometheus.Desc
	vg_seqno            *prometheus.Desc
	vg_shared           *prometheus.Desc
	vg_size             *prometheus.Desc
}

var vgnamespace = "vg"


// TODO: implement functions
