package interior

// beingRefs は spawn する敵の Ref。表に無い Ref は置かない。
var beingRefs = map[string]bool{
	"occupant": true,
}

// IsBeingRef は being の Ref が spawn されるかを返す。
func IsBeingRef(ref string) bool {
	return beingRefs[ref]
}
