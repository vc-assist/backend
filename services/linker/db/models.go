// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package db

type ExplicitLink struct {
	Leftset  string
	Leftkey  string
	Rightset string
	Rightkey string
}

type KnownKey struct {
	Setname  string
	Value    string
	Lastseen int64
}

type KnownSet struct {
	Setname string
}
