package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	m "snmp_exporter/internal/metrics"

	"github.com/sirupsen/logrus"
)

// These types have one following the other.
// We need to check indexes and sequences have them
// in the right order, so the exporter can handle them.
var combinedTypes = map[string]string{
	"InetAddress":            "InetAddressType",
	"InetAddressMissingSize": "InetAddressType",
	"LldpPortId":             "LldpPortIdSubtype",
}

// Helper to walk MIB nodes.
func walkNode(n *Node, f func(n *Node)) {
	f(n)
	for _, c := range n.Children {
		walkNode(c, f)
	}
}

// Transform the tree.
// func prepareTree(nodes *Node, logger *logrus.Entry) map[string]*Node {
// 	// Build a map from names and oids to nodes.
// 	nameToNode := map[string]*Node{}
// 	walkNode(nodes, func(n *Node) {
// 		nameToNode[n.Oid] = n
// 		nameToNode[n.Label] = n
// 	})

// 	// Trim down description to first sentence, removing extra whitespace.
// 	walkNode(nodes, func(n *Node) {
// 		s := strings.Join(strings.Fields(n.Description), " ")
// 		n.Description = strings.Split(s, ". ")[0]
// 	})

// 	// Fix indexes to "INTEGER" rather than an object name.
// 	// Example: snSlotsEntry in LANOPTICS-HUB-MIB.
// 	walkNode(nodes, func(n *Node) {
// 		indexes := []string{}
// 		for _, i := range n.Indexes {
// 			if i == "INTEGER" {
// 				// Use the TableEntry name.
// 				indexes = append(indexes, n.Label)
// 			} else {
// 				indexes = append(indexes, i)
// 			}
// 		}
// 		n.Indexes = indexes
// 	})

// 	// Copy over indexes based on augments.
// 	walkNode(nodes, func(n *Node) {
// 		if n.Augments == "" {
// 			return
// 		}
// 		augmented, ok := nameToNode[n.Augments]
// 		if !ok {
// 			logger.Warn("Can't find augmenting node", "augments", n.Augments, "node", n.Label)
// 			return
// 		}
// 		for _, c := range n.Children {
// 			c.Indexes = augmented.Indexes
// 			c.ImpliedIndex = augmented.ImpliedIndex
// 		}
// 		n.Indexes = augmented.Indexes
// 		n.ImpliedIndex = augmented.ImpliedIndex
// 	})

// 	// Copy indexes from table entries down to the entries.
// 	walkNode(nodes, func(n *Node) {
// 		if len(n.Indexes) != 0 {
// 			for _, c := range n.Children {
// 				c.Indexes = n.Indexes
// 				c.ImpliedIndex = n.ImpliedIndex
// 			}
// 		}
// 	})

// 	// Include both ASCII and UTF-8 in DisplayString, even though DisplayString
// 	// is technically only ASCII.
// 	displayStringRe := regexp.MustCompile(`^\d+[at]$`)

// 	// Apply various tweaks to the types.
// 	walkNode(nodes, func(n *Node) {
// 		// Set type on MAC addresses and strings.
// 		// RFC 2579
// 		switch n.Hint {
// 		case "1x:":
// 			n.Type = "PhysAddress48"
// 		}
// 		if displayStringRe.MatchString(n.Hint) {
// 			n.Type = "DisplayString"
// 		}

// 		// Some MIBs refer to RFC1213 for this, which is too
// 		// old to have the right hint set.
// 		if n.TextualConvention == "DisplayString" {
// 			n.Type = "DisplayString"
// 		}
// 		if n.TextualConvention == "PhysAddress" {
// 			n.Type = "PhysAddress48"
// 		}

// 		// Promote Opaque Float/Double textual convention to type.
// 		if n.TextualConvention == "Float" || n.TextualConvention == "Double" {
// 			n.Type = n.TextualConvention
// 		}

// 		// Convert RFC 2579 DateAndTime textual convention to type.
// 		if n.TextualConvention == "DateAndTime" {
// 			n.Type = "DateAndTime"
// 		}
// 		if n.TextualConvention == "ParseDateAndTime" {
// 			n.Type = "ParseDateAndTime"
// 		}
// 		if n.TextualConvention == "NTPTimeStamp" {
// 			n.Type = "NTPTimeStamp"
// 		}
// 		// Convert RFC 4001 InetAddress types textual convention to type.
// 		if n.TextualConvention == "InetAddressIPv4" || n.TextualConvention == "InetAddressIPv6" || n.TextualConvention == "InetAddress" {
// 			n.Type = n.TextualConvention
// 		}
// 		// Convert LLDP-MIB LldpPortId type textual convention to type.
// 		if n.TextualConvention == "LldpPortId" {
// 			n.Type = n.TextualConvention
// 		}
// 	})

// 	return nameToNode
// }

func prepareTree(nodes *Node, logger *logrus.Entry) map[string]*Node {
	nameToNode := buildNameToNodeMap(nodes)

	processNodes(nodes, nameToNode, logger)

	return nameToNode
}

func buildNameToNodeMap(nodes *Node) map[string]*Node {
	nameToNode := make(map[string]*Node)
	walkNode(nodes, func(n *Node) {
		nameToNode[n.Oid] = n
		nameToNode[n.Label] = n
	})
	return nameToNode
}

func processNodes(nodes *Node, nameToNode map[string]*Node, logger *logrus.Entry) {
	processDescriptions(nodes)
	processIndexes(nodes)
	processAugments(nodes, nameToNode, logger)
	copyIndexesToChildren(nodes)
	processNodeTypes(nodes)
}

func processDescriptions(nodes *Node) {
	walkNode(nodes, func(n *Node) {
		s := strings.Join(strings.Fields(n.Description), " ")
		n.Description = strings.Split(s, ". ")[0]
	})
}

func processIndexes(nodes *Node) {
	walkNode(nodes, func(n *Node) {
		indexes := make([]string, 0, len(n.Indexes))
		for _, i := range n.Indexes {
			if i == "INTEGER" {
				indexes = append(indexes, n.Label)
			} else {
				indexes = append(indexes, i)
			}
		}
		n.Indexes = indexes
	})
}

func processAugments(nodes *Node, nameToNode map[string]*Node, logger *logrus.Entry) {
	walkNode(nodes, func(n *Node) {
		if n.Augments == "" {
			return
		}
		augmented, ok := nameToNode[n.Augments]
		if !ok {
			logger.Warn("Can't find augmenting node", "augments", n.Augments, "node", n.Label)
			return
		}
		for _, c := range n.Children {
			c.Indexes = augmented.Indexes
			c.ImpliedIndex = augmented.ImpliedIndex
		}
		n.Indexes = augmented.Indexes
		n.ImpliedIndex = augmented.ImpliedIndex
	})
}

func copyIndexesToChildren(nodes *Node) {
	walkNode(nodes, func(n *Node) {
		if len(n.Indexes) > 0 {
			for _, c := range n.Children {
				c.Indexes = n.Indexes
				c.ImpliedIndex = n.ImpliedIndex
			}
		}
	})
}

func processNodeTypes(nodes *Node) {
	displayStringRe := regexp.MustCompile(`^\d+[at]$`)

	walkNode(nodes, func(n *Node) {
		processMacAddress(n)
		processDisplayString(n, displayStringRe)
		processTextualConventions(n)
	})
}

func processMacAddress(n *Node) {
	if n.Hint == "1x:" {
		n.Type = "PhysAddress48"
	}
}

func processDisplayString(n *Node, re *regexp.Regexp) {
	if re.MatchString(n.Hint) || n.TextualConvention == "DisplayString" {
		n.Type = "DisplayString"
	}
}

func processTextualConventions(n *Node) {
	switch n.TextualConvention {
	case "PhysAddress":
		n.Type = "PhysAddress48"
	case "Float", "Double":
		n.Type = n.TextualConvention
	case "DateAndTime", "ParseDateAndTime", "NTPTimeStamp":
		n.Type = n.TextualConvention
	case "InetAddressIPv4", "InetAddressIPv6", "InetAddress":
		n.Type = n.TextualConvention
	case "LldpPortId":
		n.Type = n.TextualConvention
	}
}

func metricType(t string) (string, bool) {
	if _, ok := combinedTypes[t]; ok {
		return t, true
	}
	switch t {
	case "gauge", "INTEGER", "GAUGE", "TIMETICKS", "UINTEGER", "UNSIGNED32", "INTEGER32":
		return "gauge", true
	case "counter", "COUNTER", "COUNTER64":
		return "counter", true
	case "OctetString", "OCTETSTR", "OBJID":
		return "OctetString", true
	case "BITSTRING":
		return "Bits", true
	case "InetAddressIPv4", "IpAddr", "IPADDR", "NETADDR":
		return "InetAddressIPv4", true
	case "PhysAddress48", "DisplayString", "Float", "Double", "InetAddressIPv6":
		return t, true
	case "DateAndTime":
		return t, true
	case "ParseDateAndTime":
		return t, true
	case "NTPTimeStamp":
		return t, true
	case "EnumAsInfo", "EnumAsStateSet":
		return t, true
	default:
		// Unsupported type.
		return "", false
	}
}

func metricAccess(a string) bool {
	switch a {
	case "ACCESS_READONLY", "ACCESS_READWRITE", "ACCESS_CREATE", "ACCESS_NOACCESS":
		return true
	default:
		// The others are inaccessible metrics.
		return false
	}
}

// Reduce a set of overlapping OID subtrees.
func minimizeOids(oids []string) []string {
	sort.Strings(oids)
	prevOid := ""
	minimized := []string{}
	for _, oid := range oids {
		if !strings.HasPrefix(oid+".", prevOid) || prevOid == "" {
			minimized = append(minimized, oid)
			prevOid = oid + "."
		}
	}
	return minimized
}

// Search node tree for the longest OID match.
func searchNodeTree(oid string, node *Node) *Node {
	if node == nil || !strings.HasPrefix(oid+".", node.Oid+".") {
		return nil
	}

	for _, child := range node.Children {
		match := searchNodeTree(oid, child)
		if match != nil {
			return match
		}
	}
	return node
}

type oidMetricType uint8

const (
	oidNotFound oidMetricType = iota
	oidScalar
	oidInstance
	oidSubtree
)

// Find node in SNMP MIB tree that represents the metric.
// func getMetricNode(oid string, node *Node, nameToNode map[string]*Node) (*Node, oidMetricType) {
// 	// Check if is a known OID/name.
// 	n, ok := nameToNode[oid]
// 	if ok {
// 		// Known node, check if OID is a valid metric or a subtree.
// 		_, ok = metricType(n.Type)
// 		if ok && metricAccess(n.Access) && len(n.Indexes) == 0 {
// 			return n, oidScalar
// 		}
// 		return n, oidSubtree
// 	}

// 	// Unknown OID/name, search Node tree for longest match.
// 	n = searchNodeTree(oid, node)
// 	if n == nil {
// 		return nil, oidNotFound
// 	}

// 	// Table instances must be a valid metric node and have an index.
// 	_, ok = metricType(n.Type)
// 	ok = ok && metricAccess(n.Access)
// 	if !ok || len(n.Indexes) == 0 {
// 		return nil, oidNotFound
// 	}
// 	return n, oidInstance
// }

// getMetricNode 在SNMP MIB树中查找表示指标的节点
// 返回:
//   - 找到的节点指针
//   - 节点类型枚举(oidScalar/oidSubtree/oidInstance/oidNotFound)
func getMetricNode(oid string, node *Node, nameToNode map[string]*Node) (*Node, oidMetricType) {
	// 1. 检查是否是已知OID/名称
	if n, exists := nameToNode[oid]; exists {
		return handleKnownNode(n)
	}

	// 2. 处理未知OID/名称的情况
	return handleUnknownNode(oid, node)
}

// handleKnownNode 处理已知节点的情况
func handleKnownNode(n *Node) (*Node, oidMetricType) {
	if isScalarMetric(n) {
		return n, oidScalar
	}
	return n, oidSubtree
}

// isScalarMetric 检查节点是否是有效的标量指标
func isScalarMetric(n *Node) bool {
	_, validType := metricType(n.Type)
	return validType && metricAccess(n.Access) && len(n.Indexes) == 0
}

// handleUnknownNode 处理未知节点的情况
func handleUnknownNode(oid string, node *Node) (*Node, oidMetricType) {
	n := searchNodeTree(oid, node)
	if n == nil || !isValidTableInstance(n) {
		return nil, oidNotFound
	}
	return n, oidInstance
}

// isValidTableInstance 检查节点是否是有效的表实例
func isValidTableInstance(n *Node) bool {
	_, validType := metricType(n.Type)
	return validType && metricAccess(n.Access) && len(n.Indexes) > 0
}

// In the case of multiple nodes with the same label try to return the node
// where the OID matches in every branch apart from the last one.
func getIndexNode(lookup string, nameToNode map[string]*Node, metricOid string) *Node {
	for _, node := range nameToNode {
		if node.Label != lookup {
			continue
		}

		oid := strings.Split(metricOid, ".")
		oidPrefix := strings.Join(oid[:len(oid)-1], ".")

		if strings.HasPrefix(node.Oid, oidPrefix) {
			return node
		}
	}

	// If no node matches, revert to previous behavior.
	return nameToNode[lookup]
}

// func generateConfigModule(cfg *ModuleConfig, node *Node, nameToNode map[string]*Node, logger *slog.Logger) (*m.Module, error) {
// 	out := &m.Module{}
// 	needToWalk := map[string]struct{}{}
// 	tableInstances := map[string][]string{}

// 	// Apply type overrides for the current module.
// 	for name, params := range cfg.Overrides {
// 		if params.Type == "" {
// 			continue
// 		}
// 		// Find node to override.
// 		n, ok := nameToNode[name]
// 		if !ok {
// 			logger.Warn("Could not find node to override type", "node", name)
// 			continue
// 		}
// 		// params.Type validated at generator configuration.
// 		n.Type = params.Type
// 	}

// 	// Remove redundant OIDs to be walked.
// 	toWalk := []string{}
// 	for _, oid := range cfg.Walk {
// 		if strings.HasPrefix(oid, ".") {
// 			return nil, fmt.Errorf("invalid OID %s, prefix of '.' should be removed", oid)
// 		}
// 		// Resolve name to OID if possible.
// 		n, ok := nameToNode[oid]
// 		if ok {
// 			toWalk = append(toWalk, n.Oid)
// 		} else {
// 			toWalk = append(toWalk, oid)
// 		}
// 	}
// 	toWalk = minimizeOids(toWalk)

// 	// Find all top-level nodes.
// 	metricNodes := map[*Node]struct{}{}
// 	for _, oid := range toWalk {
// 		metricNode, oidType := getMetricNode(oid, node, nameToNode)
// 		switch oidType {
// 		case oidNotFound:
// 			return nil, fmt.Errorf("cannot find oid '%s' to walk", oid)
// 		case oidSubtree:
// 			needToWalk[oid] = struct{}{}
// 		case oidInstance:
// 			// Add a trailing period to the OID to indicate a "Get" instead of a "Walk".
// 			needToWalk[oid+"."] = struct{}{}
// 			// Save instance index for lookup.
// 			index := strings.Replace(oid, metricNode.Oid, "", 1)
// 			tableInstances[metricNode.Oid] = append(tableInstances[metricNode.Oid], index)
// 		case oidScalar:
// 			// Scalar OIDs must be accessed using index 0.
// 			needToWalk[oid+".0."] = struct{}{}
// 		}
// 		metricNodes[metricNode] = struct{}{}
// 	}
// 	// Sort the metrics by OID to make the output deterministic.
// 	metrics := make([]*Node, 0, len(metricNodes))
// 	for key := range metricNodes {
// 		metrics = append(metrics, key)
// 	}
// 	sort.Slice(metrics, func(i, j int) bool {
// 		return metrics[i].Oid < metrics[j].Oid
// 	})

// 	// Find all the usable metrics.
// 	for _, metricNode := range metrics {
// 		walkNode(metricNode, func(n *Node) {
// 			t, ok := metricType(n.Type)
// 			if !ok {
// 				return // Unsupported type.
// 			}

// 			if !metricAccess(n.Access) {
// 				return // Inaccessible metrics.
// 			}

// 			metric := &m.Metric{
// 				Name:       sanitizeLabelName(n.Label),
// 				Oid:        n.Oid,
// 				Type:       t,
// 				Help:       n.Description + " - " + n.Oid,
// 				Indexes:    []*m.Index{},
// 				Lookups:    []*m.Lookup{},
// 				EnumValues: n.EnumValues,
// 			}

// 			if cfg.Overrides[metric.Name].Ignore {
// 				return // Ignored metric.
// 			}

// 			// Afi (Address family)
// 			prevType := ""
// 			// Safi (Subsequent address family, e.g. Multicast/Unicast)
// 			prev2Type := ""
// 			for count, i := range n.Indexes {
// 				index := &m.Index{Labelname: i}
// 				indexNode, ok := nameToNode[i]
// 				if !ok {
// 					logger.Warn("Could not find index for node", "node", n.Label, "index", i)
// 					return
// 				}
// 				index.Type, ok = metricType(indexNode.Type)
// 				if !ok {
// 					logger.Warn("Can't handle index type on node", "node", n.Label, "index", i, "type", indexNode.Type)
// 					return
// 				}
// 				index.FixedSize = indexNode.FixedSize
// 				if n.ImpliedIndex && count+1 == len(n.Indexes) {
// 					index.Implied = true
// 				}
// 				index.EnumValues = indexNode.EnumValues

// 				// Convert (InetAddressType,InetAddress) to (InetAddress)
// 				if subtype, ok := combinedTypes[index.Type]; ok {
// 					if prevType == subtype {
// 						metric.Indexes = metric.Indexes[:len(metric.Indexes)-1]
// 					} else if prev2Type == subtype {
// 						metric.Indexes = metric.Indexes[:len(metric.Indexes)-2]
// 					} else {
// 						logger.Warn("Can't handle index type on node, missing preceding", "node", n.Label, "type", index.Type, "missing", subtype)
// 						return
// 					}
// 				}
// 				prev2Type = prevType
// 				prevType = indexNode.TextualConvention
// 				metric.Indexes = append(metric.Indexes, index)
// 			}
// 			out.Metrics = append(out.Metrics, metric)
// 		})
// 	}

// 	// Build an map of all oid targeted by a filter to access it easily later.
// 	filterMap := map[string][]string{}

// 	for _, filter := range cfg.Filters.Static {
// 		for _, oid := range filter.Targets {
// 			n, ok := nameToNode[oid]
// 			if ok {
// 				oid = n.Oid
// 			}
// 			filterMap[oid] = filter.Indices
// 		}
// 	}

// 	// Apply lookups.
// 	for _, metric := range out.Metrics {
// 		toDelete := []string{}

// 		// Build a list of lookup labels which are required as index.
// 		requiredAsIndex := []string{}
// 		for _, lookup := range cfg.Lookups {
// 			requiredAsIndex = append(requiredAsIndex, lookup.SourceIndexes...)
// 		}

// 		for _, lookup := range cfg.Lookups {
// 			foundIndexes := 0
// 			// See if all lookup indexes are present.
// 			for _, index := range metric.Indexes {
// 				for _, lookupIndex := range lookup.SourceIndexes {
// 					if index.Labelname == lookupIndex {
// 						foundIndexes++
// 					}
// 				}
// 			}
// 			if foundIndexes == len(lookup.SourceIndexes) {
// 				if _, ok := nameToNode[lookup.Lookup]; !ok {
// 					return nil, fmt.Errorf("unknown index '%s'", lookup.Lookup)
// 				}
// 				indexNode := getIndexNode(lookup.Lookup, nameToNode, metric.Oid)
// 				typ, ok := metricType(indexNode.Type)
// 				if !ok {
// 					return nil, fmt.Errorf("unknown index type %s for %s", indexNode.Type, lookup.Lookup)
// 				}
// 				l := &m.Lookup{
// 					Labelname: sanitizeLabelName(indexNode.Label),
// 					Type:      typ,
// 					Oid:       indexNode.Oid,
// 				}
// 				for _, oldIndex := range lookup.SourceIndexes {
// 					l.Labels = append(l.Labels, sanitizeLabelName(oldIndex))
// 				}
// 				metric.Lookups = append(metric.Lookups, l)

// 				// If lookup label is used as source index in another lookup,
// 				// we need to add this new label as another index.
// 				for _, sourceIndex := range requiredAsIndex {
// 					if sourceIndex == l.Labelname {
// 						idx := &m.Index{Labelname: l.Labelname, Type: l.Type}
// 						metric.Indexes = append(metric.Indexes, idx)
// 						break
// 					}
// 				}

// 				// Make sure we walk the lookup OID(s).
// 				if len(tableInstances[metric.Oid]) > 0 {
// 					for _, index := range tableInstances[metric.Oid] {
// 						needToWalk[indexNode.Oid+index+"."] = struct{}{}
// 					}
// 				} else {
// 					needToWalk[indexNode.Oid] = struct{}{}
// 				}
// 				// We apply the same filter to metric.Oid if the lookup oid is filtered.
// 				indices, found := filterMap[indexNode.Oid]
// 				if found {
// 					delete(needToWalk, metric.Oid)
// 					for _, index := range indices {
// 						needToWalk[metric.Oid+"."+index+"."] = struct{}{}
// 					}
// 				}
// 				if lookup.DropSourceIndexes {
// 					// Avoid leaving the old labelname around.
// 					toDelete = append(toDelete, lookup.SourceIndexes...)
// 				}
// 			}
// 		}
// 		for _, l := range toDelete {
// 			metric.Lookups = append(metric.Lookups, &m.Lookup{
// 				Labelname: sanitizeLabelName(l),
// 			})
// 		}
// 	}

// 	// Ensure index label names are sane.
// 	for _, metric := range out.Metrics {
// 		for _, index := range metric.Indexes {
// 			index.Labelname = sanitizeLabelName(index.Labelname)
// 		}
// 	}

// 	// Check that the object before an InetAddress is an InetAddressType.
// 	// If not, change it to an OctetString.
// 	for _, metric := range out.Metrics {
// 		if metric.Type == "InetAddress" || metric.Type == "InetAddressMissingSize" {
// 			// Get previous oid.
// 			oids := strings.Split(metric.Oid, ".")
// 			i, _ := strconv.Atoi(oids[len(oids)-1])
// 			oids[len(oids)-1] = strconv.Itoa(i - 1)
// 			prevOid := strings.Join(oids, ".")
// 			if prevObj, ok := nameToNode[prevOid]; !ok || prevObj.TextualConvention != "InetAddressType" {
// 				metric.Type = "OctetString"
// 			} else {
// 				// Make sure the InetAddressType is included.
// 				if len(tableInstances[metric.Oid]) > 0 {
// 					for _, index := range tableInstances[metric.Oid] {
// 						needToWalk[prevOid+index+"."] = struct{}{}
// 					}
// 				} else {
// 					needToWalk[prevOid] = struct{}{}
// 				}
// 			}
// 		}
// 	}

// 	// Apply module config overrides to their corresponding metrics.
// 	for name, params := range cfg.Overrides {
// 		for _, metric := range out.Metrics {
// 			if name == metric.Name || name == metric.Oid {
// 				metric.RegexpExtracts = params.RegexpExtracts
// 				metric.DateTimePattern = params.DateTimePattern
// 				metric.Offset = params.Offset
// 				metric.Scale = params.Scale
// 				if params.Help != "" {
// 					metric.Help = params.Help
// 				}
// 				if params.Name != "" {
// 					metric.Name = params.Name
// 				}
// 			}
// 		}
// 	}

// 	// Apply filters.
// 	for _, filter := range cfg.Filters.Static {
// 		// Delete the oid targeted by the filter, as we won't walk the whole table.
// 		for _, oid := range filter.Targets {
// 			n, ok := nameToNode[oid]
// 			if ok {
// 				oid = n.Oid
// 			}
// 			delete(needToWalk, oid)
// 			for _, index := range filter.Indices {
// 				needToWalk[oid+"."+index+"."] = struct{}{}
// 			}
// 		}
// 	}

// 	out.Filters = cfg.Filters.Dynamic

// 	oids := []string{}
// 	for k := range needToWalk {
// 		oids = append(oids, k)
// 	}
// 	// Remove redundant OIDs and separate Walk and Get OIDs.
// 	for _, k := range minimizeOids(oids) {
// 		if k[len(k)-1:] == "." {
// 			out.Get = append(out.Get, k[:len(k)-1])
// 		} else {
// 			out.Walk = append(out.Walk, k)
// 		}
// 	}
// 	return out, nil
// }

func generateConfigModule(cfg *ModuleConfig, node *Node, nameToNode map[string]*Node, logger *logrus.Entry) (*m.Module, error) {
	out := &m.Module{}
	needToWalk := map[string]struct{}{}
	tableInstances := map[string][]string{}

	// 1. 应用类型覆盖
	if err := applyTypeOverrides(cfg, nameToNode, logger); err != nil {
		return nil, err
	}

	// 2. 处理需要遍历的OID
	toWalk, err := processWalkOIDs(cfg, nameToNode)
	if err != nil {
		return nil, err
	}

	// 3. 查找并处理指标节点
	metricNodes, err := findAndProcessMetricNodes(toWalk, node, nameToNode, logger, needToWalk, tableInstances)
	if err != nil {
		return nil, err
	}

	// 4. 处理所有指标
	if err := processAllMetrics(out, metricNodes, nameToNode, cfg, logger, needToWalk, tableInstances); err != nil {
		return nil, err
	}

	// 5. 处理查找和过滤
	if err := processLookupsAndFilters(out, cfg, nameToNode, needToWalk, tableInstances); err != nil {
		return nil, err
	}

	// 6. 处理特殊类型检查
	checkSpecialTypes(out, nameToNode, needToWalk, tableInstances)

	// 7. 应用模块配置覆盖
	applyModuleOverrides(out, cfg)

	// 8. 处理最终OID列表
	processFinalOIDs(out, needToWalk)

	out.Filters = cfg.Filters.Dynamic
	return out, nil
}

// 1. 应用类型覆盖
func applyTypeOverrides(cfg *ModuleConfig, nameToNode map[string]*Node, logger *logrus.Entry) error {
	for name, params := range cfg.Overrides {
		if params.Type == "" {
			continue
		}
		n, ok := nameToNode[name]
		if !ok {
			logger.Warn("Could not find node to override type", "node", name)
			continue
		}
		n.Type = params.Type
	}
	return nil
}

// 2. 处理需要遍历的OID
func processWalkOIDs(cfg *ModuleConfig, nameToNode map[string]*Node) ([]string, error) {
	toWalk := []string{}
	for _, oid := range cfg.Walk {
		if strings.HasPrefix(oid, ".") {
			return nil, fmt.Errorf("invalid OID %s, prefix of '.' should be removed", oid)
		}
		if n, ok := nameToNode[oid]; ok {
			toWalk = append(toWalk, n.Oid)
		} else {
			toWalk = append(toWalk, oid)
		}
	}
	return minimizeOids(toWalk), nil
}

// 3. 查找并处理指标节点
func findAndProcessMetricNodes(toWalk []string, node *Node, nameToNode map[string]*Node,
	logger *logrus.Entry, needToWalk map[string]struct{}, tableInstances map[string][]string) (map[*Node]struct{}, error) {

	metricNodes := map[*Node]struct{}{}
	for _, oid := range toWalk {
		metricNode, oidType := getMetricNode(oid, node, nameToNode)
		switch oidType {
		case oidNotFound:
			return nil, fmt.Errorf("cannot find oid '%s' to walk", oid)
		case oidSubtree:
			needToWalk[oid] = struct{}{}
		case oidInstance:
			needToWalk[oid+"."] = struct{}{}
			index := strings.Replace(oid, metricNode.Oid, "", 1)
			tableInstances[metricNode.Oid] = append(tableInstances[metricNode.Oid], index)
		case oidScalar:
			needToWalk[oid+".0."] = struct{}{}
		}
		metricNodes[metricNode] = struct{}{}
	}
	return metricNodes, nil
}

// 4. 处理所有指标
func processAllMetrics(out *m.Module, metricNodes map[*Node]struct{}, nameToNode map[string]*Node,
	cfg *ModuleConfig, logger *logrus.Entry, needToWalk map[string]struct{}, tableInstances map[string][]string) error {

	metrics := sortMetricNodes(metricNodes)
	for _, metricNode := range metrics {
		if err := processSingleMetric(out, metricNode, nameToNode, cfg, logger, needToWalk, tableInstances); err != nil {
			return err
		}
	}
	return nil
}

// 辅助函数：排序指标节点
func sortMetricNodes(metricNodes map[*Node]struct{}) []*Node {
	metrics := make([]*Node, 0, len(metricNodes))
	for key := range metricNodes {
		metrics = append(metrics, key)
	}
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Oid < metrics[j].Oid
	})
	return metrics
}

// 处理单个指标
func processSingleMetric(out *m.Module, metricNode *Node, nameToNode map[string]*Node,
	cfg *ModuleConfig, logger *logrus.Entry, needToWalk map[string]struct{}, tableInstances map[string][]string) error {

	walkNode(metricNode, func(n *Node) {
		t, ok := metricType(n.Type)
		if !ok {
			return // 不支持的类型
		}

		if !metricAccess(n.Access) {
			return // 不可访问的指标
		}

		metric := &m.Metric{
			Name:       sanitizeLabelName(n.Label),
			Oid:        n.Oid,
			Type:       t,
			Help:       n.Description + " - " + n.Oid,
			Indexes:    []*m.Index{},
			Lookups:    []*m.Lookup{},
			EnumValues: n.EnumValues,
		}

		if cfg.Overrides[metric.Name].Ignore {
			return // 忽略的指标
		}

		// 处理索引
		prevType := ""
		prev2Type := ""
		for count, i := range n.Indexes {
			index := &m.Index{Labelname: i}
			indexNode, ok := nameToNode[i]
			if !ok {
				logger.Warn("找不到节点的索引", "node", n.Label, "index", i)
				return
			}
			index.Type, ok = metricType(indexNode.Type)
			if !ok {
				logger.Warn("无法处理节点上的索引类型", "node", n.Label, "index", i, "type", indexNode.Type)
				return
			}
			index.FixedSize = indexNode.FixedSize
			if n.ImpliedIndex && count+1 == len(n.Indexes) {
				index.Implied = true
			}
			index.EnumValues = indexNode.EnumValues

			// 处理组合类型 (如 InetAddress + InetAddressType)
			if subtype, ok := combinedTypes[index.Type]; ok {
				if prevType == subtype {
					metric.Indexes = metric.Indexes[:len(metric.Indexes)-1]
				} else if prev2Type == subtype {
					metric.Indexes = metric.Indexes[:len(metric.Indexes)-2]
				} else {
					logger.Warn("无法处理节点上的索引类型，缺少前置类型",
						"node", n.Label, "type", index.Type, "missing", subtype)
					return
				}
			}
			prev2Type = prevType
			prevType = indexNode.TextualConvention
			metric.Indexes = append(metric.Indexes, index)
		}
		out.Metrics = append(out.Metrics, metric)
	})
	return nil
}

// 处理查找和过滤
func processLookupsAndFilters(out *m.Module, cfg *ModuleConfig, nameToNode map[string]*Node,
	needToWalk map[string]struct{}, tableInstances map[string][]string) error {

	// 构建过滤映射
	filterMap := buildFilterMap(cfg, nameToNode)

	for _, metric := range out.Metrics {
		toDelete := []string{}
		requiredAsIndex := getRequiredIndexes(cfg.Lookups)

		for _, lookup := range cfg.Lookups {
			if !hasAllRequiredIndexes(metric, lookup) {
				continue
			}

			// 处理查找节点
			indexNode, typ, err := processLookupNode(lookup, nameToNode, metric.Oid)
			if err != nil {
				return err
			}

			// 添加查找
			l := &m.Lookup{
				Labelname: sanitizeLabelName(indexNode.Label),
				Type:      typ,
				Oid:       indexNode.Oid,
				Labels:    getSanitizedLabels(lookup.SourceIndexes),
			}
			metric.Lookups = append(metric.Lookups, l)

			// 处理额外索引
			processAdditionalIndexes(metric, requiredAsIndex, l)

			// 处理需要遍历的OID
			processWalkOIDsForLookup(metric, indexNode, needToWalk, tableInstances)

			// 应用过滤
			applyFilterToMetric(metric, filterMap, indexNode, needToWalk)

			// 处理需要删除的索引
			if lookup.DropSourceIndexes {
				toDelete = append(toDelete, lookup.SourceIndexes...)
			}
		}

		// 清理需要删除的索引
		cleanupDeletedIndexes(metric, toDelete)
	}

	// 确保索引标签名称合法
	sanitizeIndexLabels(out)
	return nil
}

// 构建过滤映射
func buildFilterMap(cfg *ModuleConfig, nameToNode map[string]*Node) map[string][]string {
	filterMap := make(map[string][]string)
	for _, filter := range cfg.Filters.Static {
		for _, oid := range filter.Targets {
			if n, ok := nameToNode[oid]; ok {
				oid = n.Oid
			}
			filterMap[oid] = filter.Indices
		}
	}
	return filterMap
}

// 获取需要的索引
func getRequiredIndexes(lookups []*Lookup) []string {
	var required []string
	for _, lookup := range lookups {
		required = append(required, lookup.SourceIndexes...)
	}
	return required
}

// 检查是否包含所有需要的索引
func hasAllRequiredIndexes(metric *m.Metric, lookup *Lookup) bool {
	found := 0
	for _, index := range metric.Indexes {
		for _, lookupIndex := range lookup.SourceIndexes {
			if index.Labelname == lookupIndex {
				found++
			}
		}
	}
	return found == len(lookup.SourceIndexes)
}

// 处理查找节点
func processLookupNode(lookup *Lookup, nameToNode map[string]*Node, metricOid string) (*Node, string, error) {
	if _, ok := nameToNode[lookup.Lookup]; !ok {
		return nil, "", fmt.Errorf("未知索引 '%s'", lookup.Lookup)
	}
	indexNode := getIndexNode(lookup.Lookup, nameToNode, metricOid)
	typ, ok := metricType(indexNode.Type)
	if !ok {
		return nil, "", fmt.Errorf("未知索引类型 %s 对应 %s", indexNode.Type, lookup.Lookup)
	}
	return indexNode, typ, nil
}

// 获取清理后的标签
func getSanitizedLabels(sourceIndexes []string) []string {
	var labels []string
	for _, oldIndex := range sourceIndexes {
		labels = append(labels, sanitizeLabelName(oldIndex))
	}
	return labels
}

// 处理额外索引
func processAdditionalIndexes(metric *m.Metric, requiredAsIndex []string, l *m.Lookup) {
	for _, sourceIndex := range requiredAsIndex {
		if sourceIndex == l.Labelname {
			idx := &m.Index{Labelname: l.Labelname, Type: l.Type}
			metric.Indexes = append(metric.Indexes, idx)
			break
		}
	}
}

// 处理查找相关的需要遍历的OID
func processWalkOIDsForLookup(metric *m.Metric, indexNode *Node,
	needToWalk map[string]struct{}, tableInstances map[string][]string) {

	if len(tableInstances[metric.Oid]) > 0 {
		for _, index := range tableInstances[metric.Oid] {
			needToWalk[indexNode.Oid+index+"."] = struct{}{}
		}
	} else {
		needToWalk[indexNode.Oid] = struct{}{}
	}
}

// 应用过滤到指标
func applyFilterToMetric(metric *m.Metric, filterMap map[string][]string,
	indexNode *Node, needToWalk map[string]struct{}) {

	if indices, found := filterMap[indexNode.Oid]; found {
		delete(needToWalk, metric.Oid)
		for _, index := range indices {
			needToWalk[metric.Oid+"."+index+"."] = struct{}{}
		}
	}
}

// 清理已删除的索引
func cleanupDeletedIndexes(metric *m.Metric, toDelete []string) {
	for _, l := range toDelete {
		metric.Lookups = append(metric.Lookups, &m.Lookup{
			Labelname: sanitizeLabelName(l),
		})
	}
}

// 确保索引标签名称合法
func sanitizeIndexLabels(out *m.Module) {
	for _, metric := range out.Metrics {
		for _, index := range metric.Indexes {
			index.Labelname = sanitizeLabelName(index.Labelname)
		}
	}
}

// 6. 特殊类型检查
func checkSpecialTypes(out *m.Module, nameToNode map[string]*Node,
	needToWalk map[string]struct{}, tableInstances map[string][]string) {

	for _, metric := range out.Metrics {
		if metric.Type == "InetAddress" || metric.Type == "InetAddressMissingSize" {
			// 获取前一个OID
			oids := strings.Split(metric.Oid, ".")
			i, _ := strconv.Atoi(oids[len(oids)-1])
			oids[len(oids)-1] = strconv.Itoa(i - 1)
			prevOid := strings.Join(oids, ".")

			// 检查前一个节点是否是InetAddressType
			prevObj, ok := nameToNode[prevOid]
			if !ok || prevObj.TextualConvention != "InetAddressType" {
				metric.Type = "OctetString"
			} else {
				// 确保包含InetAddressType
				if len(tableInstances[metric.Oid]) > 0 {
					for _, index := range tableInstances[metric.Oid] {
						needToWalk[prevOid+index+"."] = struct{}{}
					}
				} else {
					needToWalk[prevOid] = struct{}{}
				}
			}
		}
	}
}

// 7. 应用模块配置覆盖
func applyModuleOverrides(out *m.Module, cfg *ModuleConfig) {
	for name, params := range cfg.Overrides {
		for _, metric := range out.Metrics {
			if name == metric.Name || name == metric.Oid {
				// 应用正则表达式提取
				if params.RegexpExtracts != nil {
					metric.RegexpExtracts = params.RegexpExtracts
				}
				// 应用日期时间格式
				if params.DateTimePattern != "" {
					metric.DateTimePattern = params.DateTimePattern
				}
				// 应用偏移量
				if params.Offset != 0 {
					metric.Offset = params.Offset
				}
				// 应用缩放比例
				if params.Scale != 0 {
					metric.Scale = params.Scale
				}
				// 应用帮助信息
				if params.Help != "" {
					metric.Help = params.Help
				}
				// 应用名称覆盖
				if params.Name != "" {
					metric.Name = params.Name
				}
			}
		}
	}
}

// 8. 处理最终OID列表
func processFinalOIDs(out *m.Module, needToWalk map[string]struct{}) {
	// 收集所有需要遍历的OID
	oids := make([]string, 0, len(needToWalk))
	for k := range needToWalk {
		oids = append(oids, k)
	}

	// 最小化OID列表，去除冗余
	minimized := minimizeOids(oids)

	// 根据OID结尾的"."区分Walk和Get请求
	for _, oid := range minimized {
		if strings.HasSuffix(oid, ".") {
			// 以"."结尾表示Get请求，去掉结尾的"."
			out.Get = append(out.Get, strings.TrimSuffix(oid, "."))
		} else {
			// 不以"."结尾表示Walk请求
			out.Walk = append(out.Walk, oid)
		}
	}
}

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func sanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}
