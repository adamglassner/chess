// Random names for the computer opponents. Each of the six strengths keeps
// its portrait, and every game it gets a fresh name that fits the face and
// the role (Mrs. for the librarian, Brother for the archivist, and so on).
//
// (c) 2026 Ironite
package main

import "math/rand/v2"

type nameSet struct {
	title  string   // "Mrs.", "Brother"... ("" = first name only)
	names  []string // first names, or surnames when there is a title
	suffix string   // "the Page"
}

var nameSets = map[string]nameSet{
	"pip":      {"", []string{"Pip", "Tobias", "Ned", "Milo", "Alfie", "Barnaby", "Kit", "Toby", "Rupert", "Jasper"}, "the Page"},
	"hollis":   {"Mrs.", []string{"Hollis", "Pemberton", "Ashby", "Whitcombe", "Larkin", "Fairfax", "Chadwick", "Bellamy", "Thistle", "Merriweather"}, ""},
	"ambrose":  {"Brother", []string{"Ambrose", "Anselm", "Benedict", "Jerome", "Cuthbert", "Aldric", "Bede", "Columba", "Edmund", "Gregory"}, ""},
	"whitlock": {"Judge", []string{"Whitlock", "Harrow", "Blackwood", "Thorne", "Pennington", "Graves", "Ashcombe", "Hartley", "Ravenhill", "Stoneleigh"}, ""},
	"vance":    {"Professor", []string{"Vance", "Marlowe", "Okafor", "Lindqvist", "Sinclair", "Devereux", "Abernathy", "Castellano", "Adebayo", "Rowntree"}, ""},
	"master":   {"Master", []string{"Casimir", "Aurelio", "Ignatius", "Osric", "Leopold", "Bartholomew", "Thaddeus", "Valentin", "Emeric", "Lazarus"}, ""},
}

// oppName is a drawn name: Short for messages ("Mrs. Ashby"), Full for the
// plaque ("Pip the Page", "The Old Master Osric").
type oppName struct {
	Short string `json:"short"`
	Full  string `json:"full"`
}

func randomName(level string, avoid string) oppName {
	ns, ok := nameSets[level]
	if !ok {
		ns = nameSets["pip"]
	}
	var n string
	for i := 0; i < 8; i++ {
		n = ns.names[rand.IntN(len(ns.names))]
		if n != avoid {
			break
		}
	}
	short := n
	if ns.title != "" {
		short = ns.title + " " + n
	}
	full := short
	if ns.suffix != "" {
		full = n + " " + ns.suffix
	}
	if level == "master" {
		full = "The Old Master " + n
	}
	return oppName{short, full}
}

// drawNames picks a name for each strength (shown on the title screen and
// used for the next game against that opponent).
func drawNames() map[string]oppName {
	m := map[string]oppName{}
	for _, f := range folks {
		m[f.ID] = randomName(f.ID, "")
	}
	return m
}
