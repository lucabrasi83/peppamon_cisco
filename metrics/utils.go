package metrics

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/lucabrasi83/peppamon_cisco/logging"
)

// matchRegexpIOSXEVersion is a convenience function that converts the Cisco 'show version'
// into the actual IOS-XE version
func matchRegexpIOSXEVersion(v string) string {
	r, err := regexp.Compile(`Version (.*?),`)

	if err != nil {
		logging.PeppaMonLog("info",
			"Failed to get IOS-XE Version information with string %v and error %v", v, err)

		return "N/A"
	}

	m := r.FindSubmatch([]byte(v))

	// We expect a byte slice of 2 elements if the Regex matches and return the last element as the matching subgroup
	if len(m) == 2 {
		return string(m[1])
	}
	return v
}

// intToIP4 is a helper function that converts base 10 IP address into a string
func intToIP4(ipInt int64) string {

	// need to do two bit shifting and “0xff” masking
	b0 := strconv.FormatInt((ipInt>>24)&0xff, 10)
	b1 := strconv.FormatInt((ipInt>>16)&0xff, 10)
	b2 := strconv.FormatInt((ipInt>>8)&0xff, 10)
	b3 := strconv.FormatInt(ipInt&0xff, 10)

	ipOctets := []string{b0, b1, b2, b3}

	return strings.Join(ipOctets, ".")
}

func ospfNbrStatusToNum(status string) float64 {

	nbrStatusMap := map[string]float64{
		"ospf-nbr-down":           1,
		"ospf-nbr-attempt":        2,
		"ospf-nbr-init":           3,
		"ospf-nbr-two-way":        4,
		"ospf-nbr-exchange-start": 5,
		"ospf-nbr-exchange":       6,
		"ospf-nbr-loading":        7,
		"ospf-nbr-full":           8,
	}

	return nbrStatusMap[status]
}

// mapBgpNeighborFSMToInteger is a helper function to map the neighbor FSM status to an integer for Grafana dashboards
func mapBgpNeighborFSMToInteger(status string) string {

	// Neighbor FSM Status mapped to Integer for Grafana cell coloring
	// Workaround until Grafana allows mapping of colors to string values
	peerStatusMap := map[string]string{
		"fsm-idle":        "0",
		"fsm-connect":     "1",
		"fsm-active":      "2",
		"fsm-opensent":    "3",
		"fsm-openconfirm": "4",
		"fsm-established": "5",
	}

	return peerStatusMap[status]
}

// convTOStoDSCP is a convenience function to convert the ToS value as a DSCP and returns the DSCP class
func convTOStoDSCP(tos int) string {

	// DSCP Decimal Value
	dscpDecimal := tos >> 2

	dscpMapDecToClass := map[int]string{
		8:  "CS1",
		10: "AF11",
		12: "AF12",
		14: "AF13",
		16: "CS2",
		18: "AF21",
		20: "AF22",
		22: "AF23",
		24: "CS3",
		26: "AF31",
		28: "AF32",
		30: "AF33",
		32: "CS4",
		34: "AF41",
		36: "AF42",
		38: "AF43",
		40: "CS5",
		46: "EF",
		48: "CS6",
		56: "CS7",
	}

	return dscpMapDecToClass[dscpDecimal]

}

// convIPSlaTagToDesc is a helper function to convert an IP SLA Tag into a meaningful description
// It is decomposing the standard model tag in the configuration to describe the IP SLA
// Class Of Service and Destination Hostname
func convIPSlaTagToDesc(tag string) (string, string) {

	if tag == "" {
		return "N/A", "N/A"
	}

	tagSplit := strings.Split(tag, "_")

	if len(tagSplit) == 3 && strings.Contains(tagSplit[0], "COS") {
		cos := tagSplit[0]
		dstHost := strings.Split(tagSplit[2], ":")[0]

		return cos, dstHost
	}

	return "N/A", "N/A"

}
