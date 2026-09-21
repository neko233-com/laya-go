package engine

import "sort"

func sortStrings(s []string) { sort.Strings(s) }

func sortModelInfo(in []apitypesModelInfo) {
	sort.Slice(in, func(i, j int) bool { return in[i].ID < in[j].ID })
}
