package dkg

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"

	"github.com/drand/kyber"
	"github.com/drand/kyber/share"
)

// Index is an alias to designate the index of a node. The index is used to
// evaluate the share of a node, and is thereafter fixed. A node will use the
// same index for generating a partial signature afterwards for example.
type Index = uint32

// Node represents the public key and its index amongt the list of participants.
// For a fresh DKG, the index can be anything but we usually take the index that
// corresponds to the position in the list of participants. For a resharing, if
// that node is a node that has already ran the DKG, we need to use the same
// index as it was given in the previous DKG in the list of OldNodes, in the DKG
// config.
type Node struct {
	Index  Index
	Public kyber.Point
}

func (n *Node) Equal(n2 *Node) bool {
	return n.Index == n2.Index && n.Public.Equal(n2.Public)
}

// Result is the struct that is outputted by the DKG protocol after it finishes.
// It contains both the list of nodes that successfully ran the protocol and the
// share of the node.
type Result struct {
	QUAL []Node
	Key  *DistKeyShare
}

func (r *Result) PublicEqual(r2 *Result) bool {
	if len(r.Key.Commits) != len(r2.Key.Commits) {
		return false
	}
	if len(r.QUAL) != len(r2.QUAL) {
		return false
	}
	lenC := len(r.Key.Commits)
	for i := 0; i < lenC; i++ {
		if !r.Key.Commits[i].Equal(r2.Key.Commits[i]) {
			return false
		}
	}
	for i := 0; i < len(r.QUAL); i++ {
		if !r.QUAL[i].Equal(&r2.QUAL[i]) {
			return false
		}
	}
	return true
}

// DistKeyShare holds the share of a distributed key for a participant.
type DistKeyShare struct {
	// Coefficients of the public polynomial holding the public key.
	Commits []kyber.Point
	// Share of the distributed secret which is private information.
	Share *share.PriShare
}

// Public returns the public key associated with the distributed private key.
func (d *DistKeyShare) Public() kyber.Point {
	return d.Commits[0]
}

// PriShare implements the dss.DistKeyShare interface so either pedersen or
// rabin dkg can be used with dss.
func (d *DistKeyShare) PriShare() *share.PriShare {
	return d.Share
}

// Commitments implements the dss.DistKeyShare interface so either pedersen or
// rabin dkg can be used with dss.
func (d *DistKeyShare) Commitments() []kyber.Point {
	return d.Commits
}

// Deal holds the Deal for one participant as well as the index of the issuing
// Dealer.
type Deal struct {
	// Index of the share holder
	ShareIndex uint32
	// encrypted share issued to the share holder
	EncryptedShare []byte
}

type DealBundle struct {
	DealerIndex uint32
	Deals       []Deal
	// Public coefficients of the public polynomial used to create the shares
	Public []kyber.Point
}

// Hash hashes the index, public coefficients and deals
func (d *DealBundle) Hash() []byte {
	// first order the deals in a  stable order
	sort.Slice(d.Deals, func(i, j int) bool {
		return d.Deals[i].ShareIndex < d.Deals[j].ShareIndex
	})
	h := sha256.New()
	binary.Write(h, binary.BigEndian, d.DealerIndex)
	for _, c := range d.Public {
		cbuff, _ := c.MarshalBinary()
		h.Write(cbuff)
	}
	for _, deal := range d.Deals {
		binary.Write(h, binary.BigEndian, deal.ShareIndex)
		h.Write(deal.EncryptedShare)
	}
	return h.Sum(nil)
}

// Response holds the Response from another participant as well as the index of
// the target Dealer.
type Response struct {
	// Index of the Dealer for which this response is for
	DealerIndex uint32
	Status      bool
}

type ResponseBundle struct {
	// Index of the share holder for which these reponses are for
	ShareIndex uint32
	Responses  []Response
}

// Hash hashes the share index and responses
func (r *ResponseBundle) Hash() []byte {
	// first order the response slice in a canonical order
	sort.Slice(r.Responses, func(i, j int) bool {
		return r.Responses[i].DealerIndex < r.Responses[j].DealerIndex
	})
	h := sha256.New()
	binary.Write(h, binary.BigEndian, r.ShareIndex)
	for _, resp := range r.Responses {
		binary.Write(h, binary.BigEndian, resp.DealerIndex)
		if resp.Status {
			binary.Write(h, binary.BigEndian, byte(1))
		} else {
			binary.Write(h, binary.BigEndian, byte(0))
		}
	}
	return h.Sum(nil)
}

func (b *ResponseBundle) String() string {
	var s = fmt.Sprintf("ShareHolder %d: ", b.ShareIndex)
	var arr []string
	for _, resp := range b.Responses {
		arr = append(arr, fmt.Sprintf("{dealer %d, status %v}", resp.DealerIndex, resp.Status))
	}
	s += "[" + strings.Join(arr, ",") + "]"
	return s
}

type JustificationBundle struct {
	DealerIndex    uint32
	Justifications []Justification
}

type Justification struct {
	ShareIndex uint32
	Share      kyber.Scalar
}

func (j *JustificationBundle) Hash() []byte {
	// sort them in a canonical order
	sort.Slice(j.Justifications, func(a, b int) bool {
		return j.Justifications[a].ShareIndex < j.Justifications[b].ShareIndex
	})
	h := sha256.New()
	binary.Write(h, binary.BigEndian, j.DealerIndex)
	for _, just := range j.Justifications {
		binary.Write(h, binary.BigEndian, just.ShareIndex)
		sbuff, _ := just.Share.MarshalBinary()
		h.Write(sbuff)
	}
	return h.Sum(nil)
}

type AuthDealBundle struct {
	Bundle    *DealBundle
	Signature []byte
}

type AuthResponseBundle struct {
	Bundle    *ResponseBundle
	Signature []byte
}

type AuthJustifBundle struct {
	Bundle    *JustificationBundle
	Signature []byte
}