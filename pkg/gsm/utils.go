package gsm

import "strings"

func extractNetwork(cops string) string {
	// Filter network name
	start := strings.Index(cops, "\"")
	end := strings.LastIndex(cops, "\"")
	network := strings.ToLower(strings.TrimSpace(cops[start+1 : end]))
	if strings.Contains(network, "vietnamobile") {
		network = "Vietnamobile"
	} else if strings.Contains(network, "viettel") {
		network = "Viettel"
	} else if strings.Contains(network, "mobifone") {
		network = "Mobifone"
	} else if strings.Contains(network, "vinaphone") {
		network = "Vinaphone"
	} else if strings.Contains(network, "vina") {
		network = "Vinaphone"
	} else if strings.Contains(network, "itel") {
		network = "iTel"
	} else {
		network = "Unknown"
	}
	return network
}
